#!/usr/bin/env python3
import os
from typing import List, Optional

import dotenv
import psycopg
import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from eps_metric_free import compute_metrics, yahoo_price

dotenv.load_dotenv()

app = FastAPI(title="Fundamentals API")


def get_db() -> psycopg.Connection:
    db_url = os.getenv("DB_URL")
    if not db_url:
        raise RuntimeError("DB_URL not set")
    return psycopg.connect(db_url)


class UpdateFundamentalsRequest(BaseModel):
    symbols: List[str]
    use_final_metric: bool = False


class UpdateQuotesRequest(BaseModel):
    symbols: Optional[List[str]] = None


@app.post("/api/update/fundamentals")
def update_fundamentals(req: UpdateFundamentalsRequest):
    if not req.symbols:
        # default to all distinct tickers from stocks table
        with get_db() as conn, conn.cursor() as cur:
            cur.execute("SELECT DISTINCT ticker FROM stocks")
            req.symbols = [r[0] for r in cur.fetchall()]
        if not req.symbols:
            raise HTTPException(status_code=400, detail="symbols required")
    updated = 0
    errors = 0
    with get_db() as conn:
        for sym in req.symbols:
            sym = sym.strip().upper()
            try:
                res = compute_metrics(sym)
                # EPS selection: prefer TTM (sum of last 4 quarters),
                # then fall back to next-year EPS estimate, then current-year.
                eps_ttm = res.get("eps_ttm")
                eps_next = res.get("eps_next")
                eps_cur = res.get("eps_current")
                eps = None
                if isinstance(eps_ttm, (int, float)):
                    eps = float(eps_ttm)
                elif isinstance(eps_next, (int, float)):
                    eps = float(eps_next)
                elif isinstance(eps_cur, (int, float)):
                    eps = float(eps_cur)
                if eps is None or eps == 0:
                    continue
                # Always use the final metric as it's more comprehensive
                gm = res.get("final_metric_percent")
                if not isinstance(gm, (int, float)):
                    # Fallback to forward_yoy_avg_percent if final_metric_percent is not available
                    gm = res.get("forward_yoy_avg_percent")
                if not isinstance(gm, (int, float)):
                    continue
                growth_decimal = float(gm) / 100.0
                mom = res.get("momentum_percent")
                surprise_sum = float(mom) if isinstance(mom, (int, float)) else None
                with conn.cursor() as cur:
                    cur.execute(
                        """
UPSERT INTO fundamentals (ticker, eps_avg, growth_estimate, surprise_sum, updated_at)
VALUES (%s, %s, %s, %s, now())
""",
                        (sym, eps, growth_decimal, surprise_sum),
                    )
                updated += 1
            except Exception as e:
                errors += 1
                print(f"[fundamentals] {sym} error: {e}", flush=True)
    print(f"[fundamentals] updated={updated} errors={errors} symbols={len(req.symbols)}", flush=True)
    return {"updated": updated, "errors": errors, "symbols": req.symbols}


@app.post("/api/update/quotes")
def update_quotes(req: UpdateQuotesRequest):
    syms = req.symbols
    if not syms:
        # fetch distinct tickers from stocks if not provided
        with get_db() as conn, conn.cursor() as cur:
            cur.execute("SELECT DISTINCT ticker FROM stocks")
            syms = [r[0] for r in cur.fetchall()]
    if not syms:
        raise HTTPException(status_code=400, detail="no symbols")
    updated = 0
    errors = 0
    with get_db() as conn:
        for sym in syms:
            sym = sym.strip().upper()
            try:
                p = yahoo_price(sym)
                if not isinstance(p, (int, float)) or p <= 0:
                    continue
                with conn.cursor() as cur:
                    cur.execute(
                        """
INSERT INTO quotes_cache(symbol, price, as_of, updated_at)
VALUES (%s,%s, now(), now())
ON CONFLICT (symbol) DO UPDATE SET price=EXCLUDED.price, as_of=EXCLUDED.as_of, updated_at=EXCLUDED.updated_at
""",
                        (sym, float(p)),
                    )
                updated += 1
            except Exception as e:
                errors += 1
                print(f"[quotes] {sym} error: {e}", flush=True)
    print(f"[quotes] updated={updated} errors={errors} symbols={len(syms)}", flush=True)
    return {"updated": updated, "errors": errors, "symbols": syms}

@app.get("/healthz")
def healthz():
    # Basic readiness; DB connection tested lazily in endpoints
    return {"ok": True}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", "9000")))
