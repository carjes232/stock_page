#!/usr/bin/env python3
import os
import time
from typing import List

import dotenv
import psycopg
import requests

dotenv.load_dotenv()


def getenv(name: str, default: str) -> str:
    v = os.getenv(name)
    return v if v is not None and v != "" else default


def parse_duration(s: str) -> int:
    s = s.strip().lower()
    units = {"s": 1, "m": 60, "h": 3600, "d": 86400}
    if s and s[-1] in units:
        return int(float(s[:-1]) * units[s[-1]])
    return int(float(s))


def get_watchlist_symbols(db_url: str) -> List[str]:
    out: List[str] = []
    with psycopg.connect(db_url) as conn, conn.cursor() as cur:
        cur.execute("SELECT ticker FROM watchlist ORDER BY added_at DESC")
        out = [t.strip().upper() for (t,) in cur.fetchall() if t and t.strip()]
    return out


def get_portfolio_symbols(db_url: str) -> List[str]:
    out: List[str] = []
    with psycopg.connect(db_url) as conn, conn.cursor() as cur:
        cur.execute("SELECT DISTINCT ticker FROM portfolio")
        out = [t.strip().upper() for (t,) in cur.fetchall() if t and t.strip()]
    return out


def main():
    api = getenv("FUNDAMENTALS_API_BASE", "http://fundamentals-api:9000")
    db_url = os.getenv("DB_URL")
    if not db_url:
        raise SystemExit("DB_URL not set")
    syms_env = os.getenv("FUNDAMENTALS_SYMBOLS")
    symbols = [s.strip().upper() for s in syms_env.split(",") if s.strip()] if syms_env else None
    # Always use the final metric as it's more comprehensive
    use_final = True
    price_every = parse_duration(getenv("PRICE_UPDATE_INTERVAL", "24h"))
    # Only refresh quotes that are older than this age
    min_quote_age = parse_duration(getenv("QUOTES_MIN_REFRESH_AGE", "6h"))
    fund_every = parse_duration(getenv("FUNDAMENTALS_UPDATE_INTERVAL", "720h"))
    # recentN is ignored now; we refresh only watchlist for scheduled jobs
    _recentN = int(getenv("TOP_RECENT_COUNT", "50"))

    next_price = time.time() + 3
    next_fund = time.time() + 10
    while True:
        now = time.time()
        # Compose symbol list
        if symbols:
            target = symbols
        else:
            # Get both watchlist and portfolio symbols
            watchlist_symbols = get_watchlist_symbols(db_url)
            portfolio_symbols = get_portfolio_symbols(db_url)
            # Combine and deduplicate symbols
            target = list(set(watchlist_symbols + portfolio_symbols))
        if target:
            print(f"[scheduler] target symbols: {len(target)}", flush=True)
        else:
            print(f"[scheduler] no symbols found; set FUNDAMENTALS_SYMBOLS or add to watchlist", flush=True)
        # Update quotes daily on weekdays only
        if now >= next_price:
            # 0=Mon .. 6=Sun; skip Sat/Sun
            weekday = time.gmtime(now).tm_wday
            if weekday >= 5:
                next_price = now + price_every
            else:
                try:
                    # Filter to only stale symbols by checking quotes_cache.as_of
                    stale: List[str] = []
                    try:
                        with psycopg.connect(db_url) as conn, conn.cursor() as cur:
                            now = time.time()
                            for t in target:
                                cur.execute("SELECT as_of FROM quotes_cache WHERE symbol = %s", (t,))
                                row = cur.fetchone()
                                if not row or not row[0]:
                                    stale.append(t)
                                else:
                                    as_of = row[0]
                                    age = now - as_of.timestamp()
                                    if age >= min_quote_age:
                                        stale.append(t)
                    except Exception:
                        # If any issue checking freshness, fall back to all symbols
                        stale = list(target)

                    if not stale:
                        print(f"[scheduler] quotes: all up-to-date (< {min_quote_age}s)", flush=True)
                    else:
                        print(f"[scheduler] updating quotes for {len(stale)} symbols", flush=True)
                        r = requests.post(api + "/api/update/quotes", json={"symbols": stale}, timeout=45)
                    print(f"[scheduler] quotes status={r.status_code} body={r.text[:200]}", flush=True)
                    # Log failed symbols if available in response
                    try:
                        response_data = r.json()
                        failed_symbols = response_data.get("failed_symbols", [])
                        if failed_symbols:
                            print(f"[scheduler] failed to update quotes for symbols: {failed_symbols}", flush=True)
                    except Exception:
                        pass  # Ignore JSON parsing errors
                except Exception as e:
                    print(f"[scheduler] quotes error: {e}", flush=True)
                next_price = now + price_every
        if now >= next_fund:
            try:
                print(f"[scheduler] updating fundamentals for {len(target)} symbols (use_final={use_final})", flush=True)
                r = requests.post(api + "/api/update/fundamentals", json={"symbols": target, "use_final_metric": use_final}, timeout=180)
                print(f"[scheduler] fundamentals status={r.status_code} body={r.text[:200]}", flush=True)
            except Exception as e:
                print(f"[scheduler] fundamentals error: {e}", flush=True)
            next_fund = now + fund_every
        time.sleep(5)


if __name__ == "__main__":
    main()
