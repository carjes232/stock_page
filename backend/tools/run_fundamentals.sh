#!/bin/sh
set -eu

# Install Python deps (cached in the container layer only during this run)
python -m pip install --no-cache-dir -r backend/tools/requirements.txt

# Wait for backend to be up (so migrations are applied)
BACKEND_URL="http://backend:8080/healthz"
echo "Waiting for backend at ${BACKEND_URL}..."
python - <<'PY'
import os, sys, time, urllib.request
url = os.environ.get('BACKEND_URL', 'http://backend:8080/healthz')
for i in range(180):
    try:
        with urllib.request.urlopen(url, timeout=2) as r:
            if r.status == 200:
                print('backend ready')
                sys.exit(0)
    except Exception:
        pass
    time.sleep(1)
print('backend not ready after timeout', file=sys.stderr)
sys.exit(0)
PY

# Default symbols if not provided
SYMS=${FUNDAMENTALS_SYMBOLS:-NVDA}

echo "Upserting fundamentals for: ${SYMS}"
python backend/tools/upsert_fundamentals.py --symbols "${SYMS}" ${FUNDAMENTALS_USE_FINAL_METRIC:+--use-final-metric}

echo "Fundamentals updater finished."
