# Stock Info Challenge - Dockerized Monorepo

Fully containerized solution scaffold for the SWE challenge.

Stack
- Backend: Go (Gin, pgx), CockroachDB, migrations, ingestion cron + manual trigger, REST API
- Frontend: Vue 3, Vite, TypeScript, Pinia, Tailwind
- Database: CockroachDB (single-node for local)
- Orchestration: Docker Compose
- Dev: Vite dev server

Quick start
1) Copy env
   cp .env.example .env
   # Fill API token (raw token only, no "Bearer ")

2) Build & run (dev)
   docker compose up --build

3) Open apps
   - Backend: http://localhost:8080
   - Frontend (dev): http://localhost:5173
   - Cockroach UI: http://localhost:8082

Default ports
- Backend: 8080
- Frontend (dev): 5173
- CockroachDB: host 26258 -> container 26259 (SQL), host 8082 -> container 8081 (Admin)

Environment variables (.env)
- API_BASE=
- API_TOKEN=<paste raw token here>  # the app adds the Bearer prefix
- PRICE_TOPK=20        # top-K enrichment in backend scoring (uses cached quotes)
- QUOTES_TTL=24h       # cache quote snapshots; Python updater refreshes these
- DB_URL=postgresql://root@db:26259/stocks?sslmode=disable
- INGEST_INTERVAL=15m
- BACKEND_PORT=8080
- FRONTEND_PORT=5173
- DISABLE_GRAHAM_PROVIDER=true  # use fundamentals table populated by Python
- FUNDAMENTALS_API_BASE=http://fundamentals-api:9000
- ALPHAVANTAGE_KEY=...  # used by Python fundamentals service
- FUNDAMENTALS_SYMBOLS=NVDA,AAPL,MSFT  # or leave empty to use watchlist+recent
- FUNDAMENTALS_USE_FINAL_METRIC=false
- PRICE_UPDATE_INTERVAL=24h
- FUNDAMENTALS_UPDATE_INTERVAL=720h
- TOP_RECENT_COUNT=50

Services (compose)
- db: CockroachDB single-node, volume persisted
- backend: Go app with migrations + REST + ingestion
- frontend: Vite dev server with proxy to backend

Development workflow
- The backend will run migrations on start and expose:
  - GET /healthz
  - GET /api/stocks
  - GET /api/stocks/:ticker
  - GET /api/stocks/search?q=<query>&page=<n>&limit=<n>
  - GET /api/stocks/sort?field=<whitelisted>&order=ASC|DESC&page=<n>&limit=<n>
  - POST /api/admin/ingest  (manual ingestion)
  - POST /api/admin/fundamentals/refresh  (proxy to Python fundamentals API)
  - GET /api/recommendations
    - Recommendations include `current_price` and `percent_upside` when a cached quote exists (Python updater writes quotes to quotes_cache).
    - If fundamentals are present in the DB, recommendations include `eps` and `intrinsic_value`.
- The frontend calls the backend via /api (proxy configured by Vite dev server to http://backend:8080 inside compose or localhost:8080 on host)

Project structure
- backend/
  - cmd/api/main.go
  - internal/
    - api/ (handlers, router)
    - db/ (pool + migrations runner, migrations embedded via go:embed at internal/db/migrations)
    - ingest/ (external API client + ingestion job)
    - models/ (domain structs)
    - rec/ (recommendation scoring)
    - config/ (env config)
  - Dockerfile
- frontend/
  - Dockerfile (dev)
  - package.json, vite.config.ts, tailwind.config.js, postcss.config.js
  - src/
    - main.ts, App.vue
    - stores/
    - components/
    - pages/

Build commands
- docker compose build
- docker compose up

Testing
- Backend: `cd backend && go test ./...`
- Frontend: (not configured yet)

Free fundamentals (EPS + growth)
- Recommended (Compose auto-run): set in `.env`:
  - `DISABLE_GRAHAM_PROVIDER=true`
  - `ALPHAVANTAGE_KEY=...`
  - `FUNDAMENTALS_SYMBOLS=NVDA,AAPL,MSFT` (your universe)
  Then run `docker compose up` — services include:
  - `fundamentals-api`: FastAPI service the backend (and you) can call on-demand
  - `fundamentals-scheduler`: periodic updater (prices daily by default, fundamentals monthly)
- Manual (host-run):
  - `pip install -r backend/tools/requirements.txt`
  - `export DB_URL=postgresql://root@localhost:26258/stocks?sslmode=disable`
  - `export ALPHAVANTAGE_KEY=...`
  - `python3 backend/tools/upsert_fundamentals.py --symbols NVDA,AAPL,MSFT`
- Inspect metrics for a single symbol:
  - `python3 backend/tools/eps_metric_free.py --symbol NVDA --json`

CI
- GitHub Actions runs on pushes and pull requests to `main`:
  - Backend: Go 1.23.x, `go mod download`, `go test ./...`, `go vet ./...`
  - Frontend: Node 20.x, `npm ci`, `npm run build`

Deterministic installs
- Frontend Dockerfile enforces npm ci. Commit package-lock.json and avoid npm install in CI/builds.

Notes
- Do not commit real API keys. Only keep `.env.example` with placeholders.
- Public repositories: avoid including company names or confidential info in code or docs.
- For prod mode, we’ll add a stage that builds frontend and serves via nginx.
- On-demand fundamentals refresh
  - Backend proxies to Python service: `POST /api/admin/fundamentals/refresh`
  - JSON body: `{ "symbols": ["NVDA","AAPL"], "use_final_metric": false }`
  - Or query: `/api/admin/fundamentals/refresh?symbols=NVDA,AAPL&use_final_metric=false`

Watchlist
- Manage your tickers for guaranteed coverage:
  - GET `/api/watchlist`
  - POST `/api/watchlist` with `{ "ticker": "NVDA", "notes": "high conviction" }`
  - DELETE `/api/watchlist/NVDA`
- Scheduler updates quotes daily and fundamentals monthly for union of Watchlist + top recent tickers from `stocks`.
