#!/usr/bin/env python3
"""
Free EPS-growth metric using Yahoo Finance (yahooquery) only — with precise control
of EPS-surprise aggregation and optional winsorization.

What’s included:
- Projection choice: --projection-mode constant | glide
- Manual EPS path: --manual-eps "2026:4.17,2027:5.63,2028:6.65"
- Prints BOTH arithmetic avg% and CAGR% across FY range
- Revision-breadth adjustment
- EPS TTM and Forward P/E
- **Current price** (new)

Install:
  pip install yahooquery

Examples:
  # Raw surprise sum, glide to 22% terminal, 4-year path:
  python3 eps_metric_free.py --symbol NVDA --projection-mode glide --terminal-growth 0.22 --horizon 4 --debug

  # Match an external consensus path exactly:
  python3 eps_metric_free.py --symbol NVDA     --manual-eps "2026:4.17,2027:5.63,2028:6.65" --force-years 2026-2028 --debug

  # Original constant projection (now also prints CAGR):
  python3 eps_metric_free.py --symbol NVDA --projection-mode constant --horizon 5 --debug
"""
import argparse
import datetime as dt
import json
import math
import sys
from typing import Dict, List, Optional, Tuple

try:
    from yahooquery import Ticker  # type: ignore
except Exception:
    print("Missing dependency: yahooquery. Install with: pip install yahooquery")
    raise

# -------------------- math helpers --------------------
def winsorize(x: float, ceil_abs: Optional[float]) -> float:
    if ceil_abs is None:
        return x
    return max(min(x, ceil_abs), -ceil_abs)

def ewma_avg_lastN(values: List[float], N: int = 4, halflife: float = 2.0) -> Optional[float]:
    """EWMA average of the newest N values (values expected newest->oldest)."""
    seq = [v for v in values if v is not None][:N]
    if not seq:
        return None
    lam = math.log(2.0) / halflife  # half every 'halflife' steps
    weights = [math.exp(-lam * i) for i in range(len(seq))]  # i=0 newest
    wsum = sum(weights)
    return sum(v * w for v, w in zip(seq, weights)) / wsum if wsum else None

def yoy_series(vals: List[float]) -> List[float]:
    """Return list of YoY percent changes between consecutive values (in % units)."""
    out: List[float] = []
    for i in range(1, len(vals)):
        prev, now = vals[i-1], vals[i]
        if prev is None or now is None or prev == 0:
            continue
        out.append((now - prev) / prev * 100.0)
    return out

def avg(lst: List[float]) -> Optional[float]:
    return (sum(lst) / len(lst)) if lst else None

# -------------------- Yahoo: EPS surprises (momentum) --------------------
def yahoo_last_surprises(symbol: str, winsor: Optional[float] = None) -> Tuple[List[float], List[str], Dict]:
    """
    Build surprise% list (newest->oldest) from Yahoo's earningsChart.quarterly.
    surprise% = (actual - estimate) / |estimate| * 100
    Returns (surprises%_final, quarters_labels, debug).
    Debug includes both raw and winsorized series for transparency.
    """
    dbg: Dict = {}
    raw_oldest_first: List[float] = []
    quarters_oldest_first: List[str] = []

    try:
        t = Ticker(symbol)
        e = t.earnings
        entry = e.get(symbol, {}) if isinstance(e, dict) else {}
        earnings_section = entry.get("earnings") or entry
        chart = (earnings_section.get("earningsChart") or {}) if isinstance(earnings_section, dict) else {}
        quarterly = chart.get("quarterly") or []
        if not isinstance(quarterly, list):
            quarterly = []

        for q in quarterly:
            if not isinstance(q, dict):
                continue
            sp = None
            try:
                if isinstance(q.get("surprisePercent"), (int, float)):
                    sp = float(q["surprisePercent"])  # already in %
                else:
                    act = q.get("actual")
                    est = q.get("estimate")
                    if act is not None and est is not None and est != 0:
                        sp = (float(act) - float(est)) / abs(float(est)) * 100.0
            except Exception:
                sp = None
            if sp is not None:
                raw_oldest_first.append(float(sp))
                quarters_oldest_first.append(str(q.get("date", "")))
    except Exception as ex:
        dbg["error"] = f"yahoo surprises fetch failed: {ex}"

    # Flip to newest->oldest for downstream
    raw_newest_first = list(reversed(raw_oldest_first))
    quarters_newest_first = list(reversed(quarters_oldest_first))

    # Apply winsor if requested (None = no clipping)
    if winsor is not None:
        clipped_newest_first = [winsorize(x, winsor) for x in raw_newest_first]
        final_series = clipped_newest_first
    else:
        clipped_newest_first = raw_newest_first[:]  # identical when no winsor
        final_series = raw_newest_first

    dbg["surprise%_raw_newest_first"] = raw_newest_first[:8]
    if winsor is not None:
        dbg["surprise%_winsor_newest_first"] = clipped_newest_first[:8]
    dbg["quarters_used"] = quarters_newest_first[:8]

    return final_series, quarters_newest_first, dbg

def sum_last4_surprise_percent(values: List[float]) -> Optional[float]:
    seq = [v for v in values if v is not None]
    if len(seq) < 4:
        return None
    return sum(seq[:4])  # newest->oldest

# -------------------- Yahoo core: EPS, growth, revisions, price, EPS TTM --------------------
def _fallback_longterm_from_analysis(t: Ticker) -> Optional[float]:
    """Try to find a 5y/long-term growth value in t.analysis. Return decimal or None."""
    try:
        an = t.analysis
        if not isinstance(an, dict):
            return None

        def scan(obj):
            found = []
            if isinstance(obj, dict):
                for k, v in obj.items():
                    if isinstance(v, (dict, list)):
                        found.extend(scan(v))
                    else:
                        if isinstance(v, str) and "%" in v:
                            ks = k.lower()
                            if "5" in ks or "long" in ks:
                                try:
                                    found.append(float(v.strip("%")) / 100.0)
                                except Exception:
                                    pass
            elif isinstance(obj, list):
                for it in obj:
                    found.extend(scan(it))
            return found

        cands = scan(an)
        if cands:
            for c in cands:
                if -0.9 <= c <= 3.0:
                    return c
    except Exception:
        pass
    return None

def parse_yahoo_core(symbol: str) -> Tuple[
    Optional[float], Optional[float], Optional[float], Dict, Dict, Optional[int]
]:
    """
    Returns:
      cur_year_eps, next_year_eps, long_term_growth_decimal, debug_trend, revisions_by_period, fiscal_base_year
    """
    dbg: Dict = {}
    revisions: Dict[str, Dict[str, int]] = {}

    t = Ticker(symbol)
    et = t.earnings_trend
    if not isinstance(et, dict) or symbol not in et:
        dbg["error"] = "earnings_trend not available"
        return None, None, None, dbg, revisions, None

    entry = et.get(symbol, {})
    trend = entry.get("trend") or []

    cur_year_eps = None
    next_year_eps = None
    long_term_growth = None
    fiscal_base_year: Optional[int] = None

    for item in trend:
        period = (item.get("period") or "").lower()
        ee = item.get("earningsEstimate") or {}
        growth = item.get("growth")

        try:
            if isinstance(growth, str) and "%" in growth:
                growth = float(growth.strip("%")) / 100.0
            if isinstance(growth, (int, float)) and period in ("5y", "longterm", "long-term"):
                long_term_growth = float(growth)  # decimal
        except Exception:
            pass

        avg_eps = ee.get("avg")
        if avg_eps is None:
            eps_trend = item.get("epsTrend") or {}
            avg_eps = eps_trend.get("avg") or eps_trend.get("mean")

        if period in ("0y", "currentyear", "curyear", "fy0"):
            try:
                cur_year_eps = float(avg_eps)
            except Exception:
                pass
            try:
                ed = (item.get("endDate") or "")[:4]
                if ed.isdigit():
                    fiscal_base_year = int(ed)
            except Exception:
                pass
        elif period in ("1y", "+1y", "nextyear", "f+1y", "fy+1"):
            try:
                next_year_eps = float(avg_eps)
            except Exception:
                pass

        rev = item.get("epsRevisions") or {}
        try:
            revisions[period] = {
                "up7": int(rev.get("upLast7days") or 0),
                "up30": int(rev.get("upLast30days") or 0),
                "down7": int(rev.get("downLast7Days") or 0),
                "down30": int(rev.get("downLast30days") or 0),
            }
        except Exception:
            pass

    if long_term_growth is None:
        long_term_growth = _fallback_longterm_from_analysis(t)

    dbg["raw_trend"] = trend
    dbg["parsed_cur_year_eps"] = cur_year_eps
    dbg["parsed_next_year_eps"] = next_year_eps
    dbg["parsed_long_term_growth(5y)"] = long_term_growth

    return cur_year_eps, next_year_eps, long_term_growth, dbg, revisions, fiscal_base_year

def blended_forward_growth(cur_eps: float, next_eps: float, long_term: Optional[float],
                           use_blend: bool = True) -> float:
    """
    Returns a DECIMAL growth rate used to project years 2..N.
    g1 = next/cur - 1 (near-term). If long_term exists and use_blend, blend 60/40 (g1/lt).
    Clamp to [-90%, +300%].
    """
    g1 = (next_eps / cur_eps - 1.0) if cur_eps else 0.0
    if long_term is not None and use_blend:
        g = 0.60 * g1 + 0.40 * float(long_term)
    else:
        g = g1
    return max(min(g, 3.0), -0.9)

def build_eps_path(cur_eps: float, next_eps: float, g: float,
                   years_needed: int = 5, base_year: Optional[int] = None) -> List[Tuple[int, float]]:
    if base_year is None:
        base_year = dt.date.today().year
    seq = [cur_eps, next_eps]
    last = next_eps
    while len(seq) < years_needed:
        last = last * (1.0 + g)
        seq.append(last)
    return [(base_year + i, float(seq[i])) for i in range(len(seq))]

# --------- NEW: glide projection + manual EPS + avg & CAGR helpers ----------
def parse_manual_eps(s: Optional[str]) -> Optional[List[Tuple[int, float]]]:
    """
    Parse --manual-eps like: "2026:4.17,2027:5.63,2028:6.65"
    Returns sorted [(year, eps), ...]
    """
    if not s:
        return None
    pairs = []
    for tok in s.split(","):
        if ":" not in tok:
            continue
        y, v = tok.split(":", 1)
        try:
            pairs.append((int(y.strip()), float(v.strip())))
        except Exception:
            pass
    pairs.sort(key=lambda x: x[0])
    return pairs or None

def avg_and_cagr_from_pairs(pairs: List[Tuple[int, float]]):
    """
    Returns (arith_avg_pct, cagr_pct, yoy_list_pct, used_pairs_sorted)
    """
    if not pairs or len(pairs) < 2:
        return None, None, [], []
    pairs = sorted(pairs, key=lambda x: x[0])
    vals = [v for _, v in pairs]
    ys = yoy_series(vals)  # list of % values
    arith = avg(ys)
    cagr = None
    if vals[0] is not None and vals[0] != 0 and vals[-1] is not None:
        years = len(vals) - 1
        if years > 0:
            cagr = ((vals[-1] / vals[0]) ** (1.0 / years) - 1.0) * 100.0
    return arith, cagr, ys, pairs

def build_eps_path_glide(cur_eps: float, next_eps: float, *,
                         years_needed: int = 5,
                         base_year: Optional[int] = None,
                         long_term: Optional[float] = None,
                         terminal_growth: Optional[float] = None) -> List[Tuple[int, float]]:
    """
    Glide from g1=(next/cur-1) toward gT over the horizon.
    - If long_term is available, use it for gT; else use terminal_growth (default 22%).
    - First step is exactly g1 (cur->next). Then interpolate linearly to gT.
    """
    if base_year is None:
        base_year = dt.date.today().year
    g1 = (next_eps / cur_eps - 1.0) if cur_eps else 0.0
    gT = long_term if (long_term is not None) else (terminal_growth if terminal_growth is not None else min(0.25, g1 * 0.6))

    # clamp
    g1 = max(min(g1, 3.0), -0.9)
    gT = max(min(gT, 3.0), -0.9)

    seq = [cur_eps, next_eps]
    remain = max(0, years_needed - 2)
    for i in range(remain):
        w = 1.0 if remain <= 1 else (i + 1) / float(remain)  # 0->1 across remaining steps
        gi = (1.0 - w) * g1 + w * gT
        gi = max(min(gi, 3.0), -0.9)
        seq.append(seq[-1] * (1.0 + gi))
    return [(base_year + i, float(seq[i])) for i in range(len(seq))]

def filter_pairs_by_years(pairs: Optional[List[Tuple[int, float]]],
                          force_years: Optional[Tuple[int, int]]) -> Optional[List[Tuple[int, float]]]:
    if not pairs:
        return pairs
    if not force_years:
        return sorted(pairs, key=lambda x: x[0])
    lo, hi = force_years
    out = [(y, v) for (y, v) in pairs if lo <= y <= hi]
    return sorted(out, key=lambda x: x[0]) or None

# -------------------- Revisions / Price / EPS TTM --------------------
def revision_breadth_adj(revisions: Dict[str, Dict[str, int]]) -> float:
    """
    Small % adjustment (±5% max) from analyst revisions (prefer +1y; else 0y).
    adj% = 5 * (up30 - down30) / max(1, up30 + down30)
    """
    choose = None
    for key in ("+1y", "1y", "nextyear", "f+1y", "fy+1", "0y"):
        if key in revisions:
            choose = revisions[key]
            break
    if not choose:
        return 0.0
    up = int(choose.get("up30", 0))
    down = int(choose.get("down30", 0))
    denom = max(1, up + down)
    raw = 5.0 * (up - down) / denom
    return max(min(raw, 5.0), -5.0)

def yahoo_price(symbol: str) -> Optional[float]:
    try:
        t = Ticker(symbol)
        p = t.price
        if isinstance(p, dict) and symbol in p:
            d = p[symbol]
            for key in ("regularMarketPrice", "postMarketPrice", "preMarketPrice"):
                v = d.get(key)
                if v is not None:
                    return float(v)
    except Exception:
        pass
    return None

def yahoo_eps_ttm_value(symbol: str) -> Optional[float]:
    """Get EPS (TTM) via yahooquery."""
    try:
        t = Ticker(symbol)

        p = t.price
        if isinstance(p, dict) and symbol in p:
            d = p[symbol] or {}
            for k in ("epsTrailingTwelveMonths", "trailingEps", "trailingEPS"):
                v = d.get(k)
                if v is not None:
                    return float(v)

        ks = t.key_stats
        if isinstance(ks, dict) and symbol in ks:
            d = ks[symbol] or {}
            for k in ("trailingEps", "epsTrailingTwelveMonths", "trailingEPS"):
                v = d.get(k)
                if v is not None:
                    return float(v)

        am = t.all_modules
        if isinstance(am, dict) and symbol in am:
            dks = (am[symbol] or {}).get("defaultKeyStatistics") or {}
            for k in ("trailingEps", "trailingEPS", "epsTrailingTwelveMonths"):
                v = dks.get(k)
                if v is not None:
                    return float(v)
    except Exception:
        pass
    return None

# -------------------- CLI parsing helpers --------------------
def parse_force_years(s: Optional[str]) -> Optional[Tuple[int, int]]:
    if not s:
        return None
    try:
        a, b = s.replace(" ", "").split("-")
        return int(a), int(b)
    except Exception:
        return None

def parse_winsor(s: Optional[str]) -> Optional[float]:
    if s is None:
        return None
    s = s.strip().lower()
    if s in ("none", "off", "no", "false", "0"):
        return None
    try:
        val = float(s)
        if val <= 0:
            return None
        return val
    except Exception:
        return None

def parse_weights(s: str) -> Dict[str, float]:
    out = {"momentum": 0.5, "forward": 0.4, "revisions": 0.1}
    if not s:
        return out
    for part in s.split(","):
        if "=" in part:
            k, v = part.split("=", 1)
            try:
                out[k.strip()] = float(v.strip())
            except Exception:
                pass
    return out

# -------------------- Public API --------------------
def compute_metrics(symbol: str, *,
                    momentum_mode: str = "sum",
                    winsor: Optional[str] = "none",
                    no_blend_longterm: bool = False,
                    weights: str = "momentum=0.5,forward=0.4,revisions=0.1",
                    force_years: Optional[str] = None,
                    projection_mode: str = "constant",
                    terminal_growth: float = 0.22,
                    horizon: int = 5,
                    manual_eps: Optional[str] = None,
                    debug: bool = False) -> Dict[str, object]:
    sym = symbol.upper()
    force_range = parse_force_years(force_years)
    winsor_val = parse_winsor(winsor)

    # ---- Surprises (Yahoo) ----
    surprises_final, quarters, dbg_surp = yahoo_last_surprises(sym, winsor=winsor_val)

    momentum = None
    momentum_label = ""
    if momentum_mode == "sum":
        momentum = sum_last4_surprise_percent(surprises_final)
        if winsor_val is None:
            momentum_label = "Last4 surprise% sum (Yahoo, no winsor)"
        else:
            momentum_label = f"Last4 surprise% sum (Yahoo, winsor ±{winsor_val:g}%)"
    else: # ewma_avg
        momentum = ewma_avg_lastN(surprises_final, N=4, halflife=2.0)
        if winsor_val is None:
            momentum_label = "Last4 surprise% EWMA (Yahoo, no winsor)"
        else:
            momentum_label = f"Last4 surprise% EWMA (Yahoo, winsor ±{winsor_val:g}%)"

    # ---- Yahoo EPS core + revisions + price + EPS TTM ----
    cur_eps, next_eps, long_term, dbg_y, revisions, fiscal_year = parse_yahoo_core(sym)

    # Build EPS path
    pairs: Optional[List[Tuple[int, float]]] = None
    if manual_eps:
        pairs = parse_manual_eps(manual_eps)
    elif (cur_eps is not None) and (next_eps is not None):
        if projection_mode == "glide":
            pairs = build_eps_path_glide(
                cur_eps, next_eps,
                years_needed=horizon,
                base_year=fiscal_year,
                long_term=None if no_blend_longterm else long_term,
                terminal_growth=terminal_growth,
            )
        else: # constant
            g = blended_forward_growth(cur_eps, next_eps, long_term, use_blend=(not no_blend_longterm))
            pairs = build_eps_path(cur_eps, next_eps, g, years_needed=horizon, base_year=fiscal_year)

    pairs = filter_pairs_by_years(pairs, force_range)

    forward_arith, forward_cagr, yoy_list, used_pairs = avg_and_cagr_from_pairs(pairs or [])

    # Price / forward P/E
    price = yahoo_price(sym)
    fwd_pe = (price / next_eps) if (price is not None and next_eps and next_eps != 0) else None

    # EPS TTM (Yahoo)
    eps_ttm = yahoo_eps_ttm_value(sym)

    # ---- Revision breadth small adj ----
    rev_adj = revision_breadth_adj(revisions)

    # ---- Final metric with weights (use arithmetic forward avg%) ----
    W = parse_weights(weights)
    final = None
    if (momentum is not None) or (forward_arith is not None) or (rev_adj != 0.0):
        m = momentum if momentum is not None else 0.0
        f = forward_arith if forward_arith is not None else 0.0
        r = rev_adj
        denom = (W.get("momentum", 0) + W.get("forward", 0) + W.get("revisions", 0)) or 1.0
        final = (W.get("momentum", 0) * m + W.get("forward", 0) * f + W.get("revisions", 0) * r) / denom

    return {
        "symbol": sym,
        "momentum_label": momentum_label,
        "momentum_percent": momentum,
        "forward_yoy_avg_percent": forward_arith,
        "forward_cagr_percent": forward_cagr,
        "revision_breadth_adj_percent": rev_adj,
        "final_metric_percent": final,
        "eps_current": cur_eps,
        "eps_next": next_eps,
        "eps_ttm": eps_ttm,
        "price": price,
        "fwd_pe": fwd_pe,
        "fiscal_base_year": fiscal_year,
        "pairs_used": used_pairs,
        "yoy_list_percent": yoy_list,
        "debug_surprises": dbg_surp if debug else None,
        "debug_eps_path": dbg_y if debug else None,
        "revisions": revisions if debug else None,
    }

# -------------------- Main --------------------
def main():
    ap = argparse.ArgumentParser(description="EPS-growth metric using Yahoo only, with EWMA, optional winsor, revisions, fiscal-year labeling.")
    ap.add_argument("--symbol", required=True)
    ap.add_argument("--baseline", type=float, default=None, help="Optional baseline to compare (e.g., TradingView-like)")
    ap.add_argument("--force-years", type=str, default=None, help="Optional filter range (e.g., 2026-2029)")
    ap.add_argument("--debug", action="store_true")

    # knobs
    ap.add_argument("--momentum-mode", choices=["sum", "ewma_avg"], default="sum",
                    help="How to aggregate last 4 surprise%% (default: sum)")
    ap.add_argument("--winsor", default="none",
                    help="Cap per-quarter surprise%% at ±N; use 'none' to disable (default: none)")
    ap.add_argument("--no-blend-longterm", action="store_true",
                    help="Disable blending near-term growth with 5y long-term growth (if available)")
    ap.add_argument("--weights", type=str, default="momentum=0.5,forward=0.4,revisions=0.1",
                    help='Comma list like momentum=0.5,forward=0.4,revisions=0.1')

    # NEW knobs
    ap.add_argument("--projection-mode", choices=["constant", "glide"], default="constant",
                    help="How to project beyond next FY (default: constant)")
    ap.add_argument("--terminal-growth", type=float, default=0.22,
                    help="Terminal growth (decimal) if long-term is missing; used by --projection-mode glide")
    ap.add_argument("--horizon", type=int, default=5,
                    help="Number of years to build in the EPS path (default: 5)")
    ap.add_argument("--manual-eps", type=str, default=None,
                    help='Override path with explicit pairs, e.g. "2026:4.17,2027:5.63,2028:6.65"')

    args = ap.parse_args()
    
    try:
        res = compute_metrics(
            symbol=args.symbol,
            momentum_mode=args.momentum_mode,
            winsor=args.winsor,
            no_blend_longterm=args.no_blend_longterm,
            weights=args.weights,
            force_years=args.force_years,
            projection_mode=args.projection_mode,
            terminal_growth=args.terminal_growth,
            horizon=args.horizon,
            manual_eps=args.manual_eps,
            debug=args.debug,
        )
    except Exception as e:
        print(f"Fatal error processing {args.symbol}: {e}", file=sys.stderr)
        if args.debug:
            import traceback
            traceback.print_exc()
        sys.exit(2)

    # ---- Print ----
    print(f"\nSymbol: {res['symbol']}")
    if args.force_years:
        force_range = parse_force_years(args.force_years)
        print(f"Forced forward year range: {force_range[0]}-{force_range[1]}")
    print("-" * 84)
    print(f"{ 'Component':26} {'Value'}")
    print("-" * 84)

    momentum_val = f"{res['momentum_percent']:.2f}%" if res['momentum_percent'] is not None else "N/A"
    print(f"{res['momentum_label']:26} {momentum_val}")

    hlabel = ""
    used_pairs = res['pairs_used']
    if used_pairs:
        hlabel = f" FY{used_pairs[0][0]}–FY{used_pairs[-1][0]}"

    fwd_arith_val = f"{res['forward_yoy_avg_percent']:.2f}%" if res['forward_yoy_avg_percent'] is not None else "N/A"
    print(f"{ 'Forward YoY avg% (arith)':26} {fwd_arith_val}{hlabel}")

    fwd_cagr_val = f"{res['forward_cagr_percent']:.2f}%" if res['forward_cagr_percent'] is not None else "N/A"
    print(f"{ 'Forward CAGR%':26} {fwd_cagr_val}{hlabel}")

    rev_adj_val = f"{res['revision_breadth_adj_percent']:+.2f}%"
    print(f"{ 'Revision breadth adj%':26} {rev_adj_val}")

    if res['price'] is not None:
        price_val = f"${res['price']:.2f}"
        print(f"{ 'Current price':26} {price_val}")

    if res['fwd_pe'] is not None:
        fwd_pe_val = f"{res['fwd_pe']:.2f}"
        print(f"{ 'Price / fwd EPS (P/E)':26} {fwd_pe_val}")

    if res['eps_ttm'] is not None:
        eps_ttm_val = f"{res['eps_ttm']:.2f}"
        print(f"{ 'EPS TTM':26} {eps_ttm_val}")

    final_val = f"{res['final_metric_percent']:.2f}%" if res['final_metric_percent'] is not None else "N/A"
    print(f"{ 'Final metric%':26} {final_val}")
    print("-" * 84)

    if args.baseline is not None and res['final_metric_percent'] is not None:
        diff = abs(res['final_metric_percent'] - args.baseline)
        print(f"Baseline: {args.baseline:.2f}%  |  Δ = {diff:.2f}%")

    if args.debug:
        print("\n--- Debug: Yahoo surprises ---")
        dbg_surp = res['debug_surprises']
        if dbg_surp:
            if "surprise%_raw_newest_first" in dbg_surp:
                print("surprise%_raw_newest_first :", dbg_surp["surprise%_raw_newest_first"])
            if "surprise%_winsor_newest_first" in dbg_surp:
                print("surprise%_winsor_newest_first :", dbg_surp["surprise%_winsor_newest_first"])
            print("quarters_used :", dbg_surp.get("quarters_used"))

        print("\n--- Debug: Yahoo EPS path ---")
        dbg_eps = res['debug_eps_path']
        print("pairs_used(year,eps):", res['pairs_used'] if res['pairs_used'] else None)
        print("yoy_list%:", res['yoy_list_percent'])
        if dbg_eps:
            for k, v in dbg_eps.items():
                if k not in ("pairs_used",):
                    print(k, ":", v)
        print("revisions(+1y preferred):", res['revisions'])



if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print("Fatal error:", e, file=sys.stderr)
        sys.exit(2)