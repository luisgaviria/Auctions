---
name: backend-unit-tests
description: 'Use when writing, debugging, reviewing, or running unit tests for this repository’s Go backend, including controllers, services, middleware, models, utilities, HTTP handlers, SQL behavior, authentication, caching, geocoding, and scraper helpers.'
---

# Backend Unit Tests

Use this skill for isolated unit tests under `backend/`. Keep tests deterministic, fast, and independent of a live PostgreSQL database, network service, scraper site, or local environment file.

## Test Selection

1. Start with the package owning the behavior: `controllers`, `services`, `middleware`, `models`, `utils`, `utils/sites`, `cmd`, or the root router package.
2. Run the narrowest relevant test first, then the package, then the full backend suite when the change crosses package boundaries.
3. Preserve the package name and existing test organization. Add tests beside the implementation in a `*_test.go` file.

From `backend/`:

```sh
go test ./path/to/package -run 'TestName'
go test ./path/to/package
go test ./...
go vet ./...
```

## Test Patterns

- Use the standard `testing` package, table-driven cases, and `t.Run` for related behavior.
- Use `httptest.NewRequest` and `httptest.NewRecorder` for controller, router, and middleware tests. Assert status, relevant headers, cookies, and response bodies or decoded JSON.
- Use the repository’s SQL mock helpers and `github.com/DATA-DOG/go-sqlmock` for service/controller database behavior. Set exact query and argument expectations, return realistic rows including nullable values, and finish with `mock.ExpectationsWereMet()`.
- Use `t.Setenv` for configuration and JWT secrets, and `t.Cleanup` for cache entries, global state, temporary files, or restored process resources.
- Test both successful behavior and meaningful failure paths: malformed input, empty results, `sql.ErrNoRows`, database errors, invalid credentials, missing configuration, and cache behavior where applicable.
- Decode JSON responses into the package’s response types or maps instead of asserting only formatting. Preserve API status/error contracts.
- Keep time, randomness, network, and filesystem behavior controllable. Prefer injected clients or the existing HTTP transport test helpers; never make real upstream requests in unit tests.

## Database Mocking

Keep SQL column order and scan values aligned with the implementation. Reuse local fixture helpers such as auction rows when they exist. Include `rows.Err()` and query/exec failure cases when the code owns those paths. Do not run migrations or connect to a real database as a unit-test shortcut.

## Completion Check

After editing tests or implementation, run the focused test command first. Then run `go test ./...` for shared behavior and `go vet ./...` when practical. Report unavailable checks separately, especially when they require `DATABASE_URL`, external APIs, Chrome, or other services.
