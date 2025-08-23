Challenge summary and approach

Overview
- Goal: Ingest stock rating/target updates from a provided API, persist in CockroachDB, expose a REST API in Go, and present a Vue 3 + TypeScript + Pinia + Tailwind UI with search, sorting, pagination, detail view, and recommendations.

Backend (Go)
- Stack: Gin, pgx, zap. CockroachDB schema includes ticker, company, brokerage, action, rating_from/to, target_from/to, last_rating_change_at, price_target_delta.
- Ingestion: Streams pages from API using Bearer auth and next_page, idempotently upserts by ticker, and parses prices and timestamps robustly.
- API: Routes for health, list, search, sort, detail, recommendations, and manual ingest trigger. Sorting supports whitelisted fields including updated_at and price_target_delta.
- Recommendations: Scoring blends rating transitions, price-target deltas, recency, and brokerage trust weights.

Frontend (Vue 3 + TS + Pinia + Tailwind)
- Stock list with search, sorting, pagination, and navigation to detail view.
- Recommendation banner fetches server-side recommendations.
- Vite dev proxy to backend; Docker Compose wires services.

Testing
- Backend unit tests cover ingestion parsing and error cases, router handlers, and recommendation scoring.
- Run: `cd backend && go test ./...`
  - CI also runs `go vet ./...`.

Setup
1) cp .env.example .env and set API_TOKEN (raw token, no "Bearer").
2) docker compose up --build
3) Open http://localhost:5173 (frontend), http://localhost:8080 (backend), and Cockroach UI on http://localhost:8082.

Notes and learnings
- CockroachDB SQL port inside container is set to 26259 to avoid conflicts; DB_URL defaults accordingly.
- Be careful not to commit .env or credentials; use .env.example for placeholders.

What I learned
- Designing resilient ingestion: tolerant parsing for prices and strict RFC3339 timestamps to avoid bad data.
- Table-driven tests and pgxmock for DB interactions keep tests fast and deterministic.
- Dockerized local CockroachDB with non-default port inside container avoids host collisions.

Approach
- Start with schema and ingestion client; ensure idempotent upsert by ticker.
- Implement REST endpoints next; keep handlers thin and business logic in packages.
- Add recommendation scoring with clear, tunable weights and recency/transition bonuses.
- Wire Compose for a reproducible dev environment; add CI for push/PR to main.

Thoughts on the challenge
- Clear, open-ended brief that allows room for reasonable tradeoffs.
- Extra credit ideas: add frontend tests and a production Docker image that serves built assets.
- The API is simple and clear; providing a small JSON schema sample in the prompt helped implement parsing logic defensively.
