# Repository Guidelines

Note: This file contains general contributor guidance for this repository. It is safe to include in a public submission; it contains no secrets or company-specific names.

## Contributor Checklist
- Verify tooling: `cd frontend && npm ci`; `cd backend && go mod download`.
- Add or update tests for backend logic you change.
- Follow Go and Vue naming/style rules outlined below.
- Keep edits scoped; update docs/scripts if commands change.
- Use Conventional Commits and include clear PR context.

## Project Structure & Module Organization
- `frontend/`: Vue 3 + Vite + TypeScript app (Pinia, Tailwind). Key dirs: `src/components/`, `src/pages/`, `src/stores/`, `src/router/`, `src/assets/`.
- `backend/`: Go service. Entry: `cmd/api`. Internal packages: `internal/api`, `internal/config`, `internal/db` (incl. `migrations/`), `internal/ingest`, `internal/rec`.
- Root: `docker-compose.yml`, `.env.example`, `README.md`.

## Build, Test, and Development Commands
- Frontend (dev): `cd frontend && npm ci && npm run dev` (serves on `:5173`).
- Frontend (build/preview): `npm run build` then `npm run preview`.
- Backend (deps): `cd backend && go mod download`.
- Backend (test): `go test ./...`.
- Backend (run): `go run ./cmd/api` (listens on `:${BACKEND_PORT:-8080}`).
- Full stack (Docker): `docker compose up --build`.

## Coding Style & Naming Conventions
- Go: run `gofmt` (default formatting) and prefer `go vet` locally; packages lowercased; table-driven tests when useful.
- Vue/TS: 2-space indent; component files PascalCase (e.g., `RecommendationBanner.vue`); variables/functions camelCase; keep presentational styles in Tailwind utilities or `src/assets/tailwind.css`.

## Testing Guidelines
- Backend uses Go’s `testing` with `testify` and `pgxmock` where DB is involved.
- Place tests alongside code as `*_test.go` with `func TestXxx(t *testing.T)`.
- Run all tests with `cd backend && go test ./...` before pushing.
- Frontend has no test runner configured yet; if adding one, keep tests fast and colocated.

## Commit & Pull Request Guidelines
- Commits: Conventional Commits (e.g., `feat:`, `fix:`, `chore:`) with a brief rationale.
- PRs: include purpose, linked issues, test plan, and screenshots for UI changes. Ensure `go test` passes and the app boots locally or via Compose.

## Security & Configuration Tips
- Copy `.env.example` to `.env` and set `API_TOKEN`, DB settings, and ports. Never commit secrets or tokens. `.env` is git-ignored.
- To trigger a one-off ingestion in dev, POST to `/api/admin/ingest` on the backend.

## CI
- GitHub Actions workflow at `.github/workflows/ci.yml` runs on push/PR to `main` and verifies:
  - Backend: `go test ./...`, `go vet ./...`
  - Frontend: `npm ci`, `npm run build`
