#!/usr/bin/env python3
"""
Compute EPS + forward growth via free sources (Alpha Vantage + Yahoo) and upsert
into the backend fundamentals table so the Go service can use it without relying
on FMP's growth metric.

Usage:
  # 1) Install deps
  pip install -r backend/tools/requirements.txt

  # 2) Ensure env has DB_URL and ALPHAVANTAGE_KEY
  export DB_URL=postgresql://root@localhost:26259/stocks?sslmode=disable
  export ALPHAVANTAGE_KEY=YOUR_KEY

  # 3) Update one or more tickers
  python3 backend/tools/upsert_fundamentals.py --symbols NVDA,AAPL,MSFT

By default, growth_estimate is the forward YoY average (decimal).
You can instead store the blended final metric as growth with --use-final-metric.
"""
import argparse
import os
import sys
from typing import List

import dotenv
import psycopg

from eps_metric_free import compute_metrics

dotenv.load_dotenv()


def upsert_one(conn: psycopg.Connection, ticker: str, eps: float, growth: float, surprise_sum: float | None):
    with conn.cursor() as cur:
        cur.execute(
            """
UPSERT INTO fundamentals (ticker, eps_avg, growth_estimate, surprise_sum, updated_at)
VALUES (%s, %s, %s, %s, now())
""",
            (ticker, eps, growth, surprise_sum),
        )


def parse_symbols(s: str) -> List[str]:
    out = []
    for part in s.split(','):
        p = part.strip().upper()
        if p:
            out.append(p)
    return out


def main():
    ap = argparse.ArgumentParser(description="Upsert fundamentals using free EPS-growth metric")
    ap.add_argument("--symbols", required=True, help="Comma-separated tickers, e.g., NVDA,AAPL")
    ap.add_argument("--use-final-metric", action="store_true", help="Store the blended final metric as growth")
    ap.add_argument("--momentum-mode", choices=["sum", "ewma_avg"], default="sum")
    ap.add_argument("--winsor", type=float, default=60.0)
    ap.add_argument("--no-blend-longterm", action="store_true")
    ap.add_argument("--weights", type=str, default="momentum=0.5,forward=0.4,revisions=0.1")
    args = ap.parse_args()

    db_url = os.getenv("DB_URL")
    if not db_url:
        print("DB_URL not set in env")
        sys.exit(1)
    if not os.getenv("ALPHAVANTAGE_KEY"):
        print("ALPHAVANTAGE_KEY not set in env")
        sys.exit(1)

    symbols = parse_symbols(args.symbols)
    if not symbols:
        print("No symbols provided")
        sys.exit(1)

    ok = 0
    with psycopg.connect(db_url) as conn:
        for sym in symbols:
            try:
                res = compute_metrics(
                    sym,
                    momentum_mode=args.momentum_mode,
                    winsor=args.winsor,
                    no_blend_longterm=args.no_blend_longterm,
                    weights=args.weights,
                )
                # Prefer next year EPS; fall back to current EPS
                eps_next = res.get("eps_next")
                eps_cur = res.get("eps_current")
                eps = None
                if isinstance(eps_next, (int, float)):
                    eps = float(eps_next)
                elif isinstance(eps_cur, (int, float)):
                    eps = float(eps_cur)
                if eps is None or eps == 0:
                    print(f"[{sym}] skip: missing EPS")
                    continue

                if args.use_final_metric:
                    gm = res.get("final_metric_percent")
                else:
                    gm = res.get("forward_yoy_avg_percent")
                if not isinstance(gm, (int, float)):
                    print(f"[{sym}] skip: missing growth metric")
                    continue
                growth_decimal = float(gm) / 100.0

                mom = res.get("momentum_percent")
                surprise_sum = float(mom) if isinstance(mom, (int, float)) else None

                upsert_one(conn, sym, eps, growth_decimal, surprise_sum)
                print(f"[{sym}] upserted fundamentals: eps={eps:.3f}, growth={growth_decimal:.4f}, surprise_sum={surprise_sum}")
                ok += 1
            except Exception as e:
                print(f"[{sym}] error: {e}")

    print(f"Done. Updated {ok}/{len(symbols)} tickers.")


if __name__ == "__main__":
    main()

