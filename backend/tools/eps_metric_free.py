#!/usr/bin/env python3
"""
Free EPS-growth metric using Alpha Vantage (quarterly EPS surprises) + Yahoo Finance (yahooquery).

This module exposes `compute_metrics(symbol, ...)` and a small CLI.

Env:
  export ALPHAVANTAGE_KEY="YOUR_KEY"

Install (inside repo root):
  pip install -r backend/tools/requirements.txt

CLI example:
  python3 backend/tools/eps_metric_free.py --symbol NVDA --json
"""
import argparse
import datetime as dt
import json
import math
import os
from typing import Dict, List, Optional, Tuple

import requests
import dotenv

dotenv.load_dotenv()

try:
    from yahooquery import Ticker  # type: ignore
except Exception:  # pragma: no cover - optional runtime dependency
    raise RuntimeError("Missing dependency: yahooquery. Install with: pip install -r backend/tools/requirements.txt")

# -------------------- HTTP helper --------------------
def http_get(url: str, params: Dict[str, str]) -> dict:
    r = requests.get(url, params=params, timeout=20)
    r.raise_for_status()
    try:
        return r.json()
    except json.JSONDecodeError as e:  # pragma: no cover - network content
        raise RuntimeError(f"Non-JSON response from {url}: {r.text[:200]}") from e

# -------------------- math helpers --------------------
def winsorize(x: float, ceil_abs: Optional[float]) -> float:
    if ceil_abs is None:
        return x
    return max(min(x, ceil_abs), -ceil_abs)

def ewma_avg_lastN(values: List[float], N: int = 4, halflife: float = 2.0) -> Optional[float]:
    seq = [v for v in values if v is not None][:N]
    if not seq:
        return None
    lam = math.log(2.0) / halflife  # half every 'halflife' steps
    weights = [math.exp(-lam * i) for i in range(len(seq))]  # i=0 newest
    wsum = sum(weights)
    return sum(v * w for v, w in zip(seq, weights)) / wsum if wsum else None

def yoy_series(vals: List[float]) -> List[float]:
    out: List[float] = []
    for i in range(1, len(vals)):
        prev, now = vals[i-1], vals[i]
        if prev is None or now is None or prev == 0:
            continue
        out.append((now - prev) / prev * 100.0)
    return out

def avg(lst: List[float]) -> Optional[float]:
    return (sum(lst) / len(lst)) if lst else None

# -------------------- Alpha Vantage: EPS surprises --------------------
def av_last_surprises(symbol: str, key: str, recalc: bool = True,
                      winsor: Optional[float] = None) -> Tuple[List[float], List[str], List[Optional[float]], Dict]:
    """Returns (surprises%, quarters, reported_eps, dbg).

    All series are newest->oldest per AV docs. reported_eps contains the raw
    Alpha Vantage reportedEPS values (floats when available, else None).
    """
    dbg = {}
    data = http_get("https://www.alphavantage.co/query",
                    {"function": "EARNINGS", "symbol": symbol, "apikey": key})
    rows = data.get("quarterlyEarnings", []) or []
    surprises: List[float] = []
    quarters: List[str] = []
    rep_eps: List[Optional[float]] = []
    for r in rows:
        sp = None
        rep = r.get("reportedEPS")
        # Track reported EPS raw value (float or None)
        try:
            rep_eps.append(float(rep)) if rep not in (None, "None") else rep_eps.append(None)
        except Exception:
            rep_eps.append(None)
        if recalc:
            est = r.get("estimatedEPS")
            try:
                if rep not in (None, "None") and est not in (None, "None", 0, "0"):
                    sp = (float(rep) - float(est)) / float(est) * 100.0
            except Exception:
                sp = None
        else:
            sp_str = r.get("surprisePercentage")
            try:
                if sp_str is not None and sp_str != "None":
                    sp = float(sp_str)
                else:
                    surprise = r.get("surprise"); est = r.get("estimatedEPS")
                    if surprise is not None and est not in (None, "None", 0, "0"):
                        sp = float(surprise) / float(est) * 100.0
            except Exception:
                sp = None
        if sp is not None:
            sp = winsorize(float(sp), winsor) if winsor is not None else float(sp)
            surprises.append(sp)
            quarters.append(str(r.get("fiscalDateEnding")))
    dbg["surprise%_newest_first"] = surprises[:8]
    dbg["quarters_used"] = quarters[:8]
    return surprises, quarters, rep_eps, dbg

def sum_last4_surprise_percent(values: List[float]) -> Optional[float]:
    seq = [v for v in values if v is not None]
    if len(seq) < 4:
        return None
    return sum(seq[:4])  # newest->oldest

# -------------------- Yahoo: EPS, growth, revisions, price --------------------
def _fallback_longterm_from_analysis(t: Ticker) -> Optional[float]:
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
    """Returns: cur_year_eps, next_year_eps, long_term_growth_decimal, debug_trend, revisions_by_period, fiscal_base_year"""
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

def forward_yoy_avg_from_pairs(pairs: List[Tuple[int, float]],
                               force_years: Optional[Tuple[int, int]]) -> Tuple[Optional[float], List[float], List[Tuple[int, float]]]:
    if not pairs:
        return None, [], []
    if force_years:
        lo, hi = force_years
        pairs = [(y, v) for (y, v) in pairs if lo <= y <= hi]
    pairs.sort(key=lambda x: x[0])
    vals = [v for (_, v) in pairs]
    ys = yoy_series(vals)  # percents
    return avg(ys), ys, pairs

def revision_breadth_adj(revisions: Dict[str, Dict[str, int]]) -> float:
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
    """Return a best-effort last price.

    Strategy:
    1) Try yahooquery's Ticker.price (may log crumb warnings but still often works).
    2) Fallback to Yahoo's public quote endpoint (no crumb required).
    """
    sym = symbol.upper().strip()
    # 1) yahooquery
    try:
        t = Ticker(sym)
        p = t.price
        if isinstance(p, dict) and sym in p:
            d = p[sym]
            for key in ("regularMarketPrice", "postMarketPrice", "preMarketPrice"):
                v = d.get(key)
                if v is not None:
                    return float(v)
    except Exception:
        pass

    # 2) Raw HTTP fallback
    try:
        url = "https://query1.finance.yahoo.com/v7/finance/quote"
        headers = {
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
            "Accept": "application/json",
        }
        r = requests.get(url, params={"symbols": sym}, headers=headers, timeout=15)
        r.raise_for_status()
        data = r.json()
        res = (data.get("quoteResponse", {}) or {}).get("result", []) or []
        if res:
            d = res[0]
            for key in ("regularMarketPrice", "postMarketPrice", "preMarketPrice"):
                v = d.get(key)
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
                    winsor: float = 60.0,
                    no_blend_longterm: bool = False,
                    weights: str = "momentum=0.5,forward=0.4,revisions=0.1",
                    force_years: Optional[str] = None,
                    debug: bool = False) -> Dict[str, object]:
    sym = symbol.upper()
    force_range = parse_force_years(force_years)
    av_key = os.getenv("ALPHAVANTAGE_KEY")
    if not av_key:
        raise RuntimeError("Set ALPHAVANTAGE_KEY in your environment.")

    # Surprises
    surprises, quarters, dbg_av = [], [], {}
    momentum_label = "Last4 surprise% (AV)"
    momentum = None
    eps_ttm = None
    try:
        surprises, quarters, rep_eps, dbg_av = av_last_surprises(sym, av_key, recalc=True, winsor=winsor)
        if momentum_mode == "sum":
            momentum = sum_last4_surprise_percent(surprises)
            momentum_label = "Last4 surprise% sum (AV)"
        else:
            momentum = ewma_avg_lastN(surprises, N=4, halflife=2.0)
            momentum_label = "Last4 surprise% EWMA (AV)"
        # Compute TTM EPS as sum of last 4 reported EPS values
        if rep_eps:
            vals = [v for v in rep_eps if isinstance(v, (int, float))]
            if len(vals) >= 4:
                eps_ttm = float(sum(vals[:4]))
    except Exception as ex:  # pragma: no cover - network
        dbg_av = {"error": str(ex)}

    # Yahoo EPS path + revisions
    pairs = None
    forward = None
    yoy_list: List[float] = []
    used_pairs: List[Tuple[int, float]] = []
    price = None
    fwd_pe = None
    cur_eps = None
    next_eps = None
    fiscal_year = None
    dbg_y = {}
    try:
        cur_eps, next_eps, long_term, dbg_y, revisions, fiscal_year = parse_yahoo_core(sym)
        if cur_eps is not None and next_eps is not None:
            g = blended_forward_growth(cur_eps, next_eps, long_term, use_blend=(not no_blend_longterm))
            pairs = build_eps_path(cur_eps, next_eps, g, years_needed=5, base_year=fiscal_year)
            forward, yoy_list, used_pairs = forward_yoy_avg_from_pairs(pairs, force_range)
        price = yahoo_price(sym)
        if price is not None and next_eps not in (None, 0):
            fwd_pe = price / float(next_eps)
        rev_adj = revision_breadth_adj(revisions)
    except Exception as ex:  # pragma: no cover - network
        dbg_y = {"error": str(ex)}
        revisions = {}
        rev_adj = 0.0

    # Final metric with weights
    W = parse_weights(weights)
    final = None
    if any(v is not None for v in (momentum, forward)) or rev_adj != 0.0:
        m = momentum if momentum is not None else 0.0
        f = forward if forward is not None else 0.0
        r = rev_adj
        denom = (W.get("momentum", 0) + W.get("forward", 0) + W.get("revisions", 0)) or 1.0
        final = (W.get("momentum", 0) * m + W.get("forward", 0) * f + W.get("revisions", 0) * r) / denom

    # If AV path failed to produce eps_ttm, try Yahoo earningsChart quarterly actuals
    if eps_ttm is None:
        try:
            t = Ticker(sym)
            e = t.earnings
            if isinstance(e, dict) and sym in e:
                q = ((e.get(sym) or {}).get("earningsChart") or {}).get("quarterly") or []
                # take the last 4 'actual' values (newest last in yahooquery output)
                actuals = []
                for it in q:
                    v = it.get("actual")
                    if isinstance(v, (int, float)):
                        actuals.append(float(v))
                if len(actuals) >= 4:
                    eps_ttm = float(sum(actuals[-4:]))
        except Exception:
            pass

    return {
        "symbol": sym,
        "momentum_label": momentum_label,
        "momentum_percent": momentum,
        "forward_yoy_avg_percent": forward,
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
        "debug_alpha": dbg_av if debug else None,
        "debug_yahoo": dbg_y if debug else None,
    }


def cli_main():  # pragma: no cover - CLI glue
    ap = argparse.ArgumentParser(description="Free EPS-growth metric with AV + Yahoo, with EWMA, winsor, revisions, fiscal-year labeling.")
    ap.add_argument("--symbol", required=True)
    ap.add_argument("--baseline", type=float, default=None, help="Optional baseline to compare (e.g., TradingView-like)")
    ap.add_argument("--force-years", type=str, default=None, help="Optional filter range (e.g., 2026-2029)")
    ap.add_argument("--debug", action="store_true")
    ap.add_argument("--json", action="store_true", help="Emit JSON instead of table")
    ap.add_argument("--momentum-mode", choices=["sum", "ewma_avg"], default="sum", help="How to aggregate last 4 surprise%% (default: sum)")
    ap.add_argument("--winsor", type=float, default=60.0, help="Cap per-quarter surprise%% at ±N (default: 60)")
    ap.add_argument("--no-blend-longterm", action="store_true", help="Disable blending near-term growth with 5y long-term growth (if available)")
    ap.add_argument("--weights", type=str, default="momentum=0.5,forward=0.4,revisions=0.1", help='Comma list like momentum=0.5,forward=0.4,revisions=0.1')

    args = ap.parse_args()
    res = compute_metrics(
        args.symbol,
        momentum_mode=args.momentum_mode,
        winsor=args.winsor,
        no_blend_longterm=args.no_blend_longterm,
        weights=args.weights,
        force_years=args.force_years,
        debug=args.debug,
    )

    if args.json:
        print(json.dumps(res, default=str))
        return

    # Pretty print like original
    symbol = res["symbol"]
    momentum = res["momentum_percent"]
    forward = res["forward_yoy_avg_percent"]
    rev_adj = res["revision_breadth_adj_percent"]
    final = res["final_metric_percent"]
    fwd_pe = res["fwd_pe"]
    momentum_label = res["momentum_label"]
    force_range = parse_force_years(args.force_years)

    print(f"\nSymbol: {symbol}")
    if force_range:
        print(f"Forced forward year range: {force_range[0]}-{force_range[1]}")
    print("-" * 84)
    print(f"{'Component':26} {'Value'}")
    print("-" * 84)
    print(f"{momentum_label:26} {('%.2f%%' % momentum) if momentum is not None else 'N/A'}")
    print(f"{'Forward YoY avg% (Yahoo)':26} {('%.2f%%' % forward) if forward is not None else 'N/A'}")
    print(f"{'Revision breadth adj%':26} {('%+.2f%%' % rev_adj)}")
    if fwd_pe is not None:
        print(f"{'Price / fwd EPS (P/E)':26} {('%.2f' % fwd_pe)}")
    print(f"{'Final metric%':26} {('%.2f%%' % final) if final is not None else 'N/A'}")
    print("-" * 84)

    if args.baseline is not None and final is not None:
        diff = abs(final - args.baseline)
        print(f"Baseline: {args.baseline:.2f}%  |  Δ = {diff:.2f}%")

    if args.debug:
        print("\n--- Debug: Alpha Vantage ---")
        print(json.dumps(res.get("debug_alpha"), default=str))
        print("\n--- Debug: Yahoo EPS path ---")
        print("pairs_used(year,eps):", res.get("pairs_used"))
        print("yoy_list%:", res.get("yoy_list_percent"))
        print(json.dumps(res.get("debug_yahoo"), default=str))


if __name__ == "__main__":  # pragma: no cover - CLI entry
    cli_main()
