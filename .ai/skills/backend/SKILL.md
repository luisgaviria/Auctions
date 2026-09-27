---
name: backend
description: 'Use when implementing, debugging, reviewing, or testing this repository’s Go backend: HTTP API routes, PostgreSQL queries, migrations, authentication, caching, auction scraping, geocoding, and registry enrichment.'
---

# Backend Development

Use this skill for changes under `backend/` and for work that changes the contract between the Go API and the frontend.

## Repository Map

- `main.go` wires Gorilla Mux routes, middleware, database initialization, and the HTTP server.
- `controllers/` handles HTTP input/output. `services/` owns query and business logic.
- `models/` defines database scan types and API response shapes; `AuctionModel.ToJSON()` converts nullable SQL values.
- `utils/sites/` contains source-specific auction scrapers. Shared normalization, upsert, and scrape orchestration live in `utils/scrap.go`.
- `middleware/` contains authentication and response caching; `config/` reads environment variables.
- `migrations/` contains embedded Goose PostgreSQL migrations (`migrations/embed.go`).

## Working Procedure

1. Trace the request or data flow through its owner: route in `main.go`, controller, service, model, and any relevant SQL migration or frontend API consumer. For scraping, follow the site collector into the shared normalization and persistence path.
2. Keep HTTP parsing and response writing in controllers; put reusable query/business logic in services. Use parameterized SQL arguments, propagate database errors, close result rows, and check `rows.Err()`.
3. Keep SQL select columns and `rows.Scan` arguments in the same order. Auction queries use the explicit `auctionCols` list; avoid `SELECT *` where scan shape matters. Preserve nullable database values in the model and convert them deliberately at the API boundary.
4. Add schema changes as a new, sequential Goose migration. Do not rewrite an already-applied migration. Check embedded migration behavior and compatibility with existing rows before changing constraints, indexes, or data cleanup.
5. For scraper changes, prefer source-specific parsing in that source’s file and shared data cleanup in the common pipeline. Preserve deduplication/upsert behavior and `last_seen` semantics. Respect the supplied context and avoid destructive cleanup based on a failed or partial scrape.
6. Treat auth, CORS, cache invalidation, date filtering, and slug generation as API behavior. Keep secrets out of logs and responses; do not expose credentials in frontend `VITE_*` variables.
7. Run the narrowest meaningful check, then broaden only as needed. Inspect the diff for API/schema contract changes and report any check that could not be run.

## Validation

From `backend/`:

```sh
go test ./...
go vet ./...
```

There may be no tests for the changed package; `go test ./...` still checks package compilation. Use focused tests for parsing, normalization, and API/service behavior where available. `go run .` starts the API but requires a working database configuration and database; do not invoke live scraping or migration commands against a real database merely as a validation shortcut.

## Configuration Notes

- The backend reads `DATABASE_URL`, `JWT_SECRET`, `ALLOWED_ORIGINS`, `FRONTEND_URL`, and `PORT` as applicable. `GetPort()` defaults to `8000`.
- The frontend currently falls back to `http://localhost:8080`; confirm `PORT`/`VITE_API_URL` when diagnosing local connectivity instead of assuming the documented port matches runtime configuration.
- `backend/Makefile` includes `.env` at parse time. Its Make targets may fail when that file is absent; do not create or overwrite local environment files as part of unrelated work.
- Database migration targets need a valid `DATABASE_URL` and Goose. Review their effects before running them.
