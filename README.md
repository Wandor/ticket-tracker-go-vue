# Mini Ticket Tracker

A small support-ticket tracker: a REST API in Go (standard library `net/http` + pgx), PostgreSQL
for storage and a single-page Vue 3 UI.

## Stack

| Layer    | Choice                                              |
|----------|-----------------------------------------------------|
| API      | Go 1.25+, `net/http` method-aware routing, pgx v5   |
| Database | PostgreSQL 16, plain SQL migrations embedded in the binary |
| UI       | Vue 3 (`<script setup>`, TypeScript) + Vite         |
| Tests    | `go test` + `httptest` against a real Postgres      |

## Running it

### 1. Start PostgreSQL

The quickest route is the bundled compose file, which runs Postgres only and maps it to port
`55432` to avoid clashing with a local install:

```bash
docker compose up -d db
```

Using your own Postgres instead is fine: create a database and point `DATABASE_URL` at it.

### 2. Start the API

```bash
cd api
cp .env.example .env
set -a; source .env; set +a      # or export DATABASE_URL / PORT yourself
go run ./cmd/server              # applies migrations on boot, listens on :8080
```

### 3. Start the UI

```bash
cd web
npm install
npm run dev                      # http://localhost:5173
```

The UI calls `/api/*`, and the Vite dev server proxies that to the API with the prefix stripped.
If the API is not on `localhost:8080`, set `API_URL`, e.g. `API_URL=http://localhost:9000 npm run dev`.

### Running the tests

```bash
cd api
go test ./...
```

The integration tests in `internal/server` use `TEST_DATABASE_URL` (default
`postgres://tickets:tickets@localhost:55432/tickets_test?sslmode=disable`). `TestMain` creates
that database if it is missing and migrates it; each test truncates `tickets` first.

- `internal/server/server_test.go`: create (defaults, trimming, validation), list (status
  filter, pagination, ordering, bad filter), update (partial, 404, non-updatable field) and stats.
- `internal/tickets/ticket_test.go`: table-driven unit tests for request validation.

## API

All responses are JSON. Errors share one shape:

```json
{ "error": { "code": "validation_error", "message": "Request validation failed",
             "fields": { "title": "is required" } } }
```

| Method  | Path             | Notes |
|---------|------------------|-------|
| `POST`  | `/tickets`       | Body: `title` (required, 1–200 characters after trimming), `description`, `priority` (default `medium`), `status` (default `open`). Returns `201`. |
| `GET`   | `/tickets`       | Query: `status`, `page` (default 1), `limit` (default 20, max 100). Returns `{ data, pagination: { page, limit, total, total_pages } }`, newest first. |
| `PATCH` | `/tickets/{id}`  | Body: `status` and/or `priority`. Unknown fields are rejected with `400`; unknown id is `404`. |
| `GET`   | `/tickets/stats` | `{ "open": n, "in_progress": n, "closed": n }`, always all three keys. |
| `GET`   | `/health`        | Liveness check. |

```bash
curl -X POST localhost:8080/tickets -H 'Content-Type: application/json' \
  -d '{"title": "Printer on fire", "priority": "high"}'
curl 'localhost:8080/tickets?status=open&page=1&limit=10'
curl -X PATCH localhost:8080/tickets/1 -H 'Content-Type: application/json' -d '{"status": "closed"}'
curl localhost:8080/tickets/stats
```

## Project layout

```
api/
  cmd/server/          entry point: config, migrations, graceful shutdown
  internal/
    database/          pgx pool + embedded forward-only migration runner
    httpx/             JSON responses, error shape, request decoding
    server/            builds the http.Handler (shared by main and the tests)
    tickets/
      ticket.go        types, enums, request validation
      store.go         SQL
      handler.go       HTTP layer
web/
  src/
    api.ts             typed fetch wrapper
    App.vue            page state: filter, pagination, stats, reloads
    components/        TicketForm, TicketTable, StatsBar
```

## Decisions & Tradeoffs

I stayed on the standard library for HTTP, since Go 1.22's method and path-parameter routing
covers these four endpoints without a framework, and used pgx with hand-written SQL instead of an
ORM or code generator. I started with goose for migrations, but it pulled in a large dependency
tree for one file, so I replaced it with a ~50-line runner over `embed` that records applied files
in `schema_migrations`. With more time, or once rollbacks matter, I'd go back to a real migration
tool run as a separate deploy step. Status and priority are `VARCHAR` columns with `CHECK`
constraints rather than native Postgres enums: the database still rejects bad values, but adding a
status is a one-line change. Request bodies decode into pointer fields so "omitted" and "empty" are
distinguishable, and `PATCH` uses `DisallowUnknownFields` so a request that tries to edit the
title fails loudly instead of being silently half-applied. Pagination is offset-based on
`created_at DESC, id DESC` with matching indexes, which is fine at this scale but would move to
keyset pagination for large or busy tables. The integration tests hit a real Postgres instead of a
mocked store, because the constraints, ordering and `GROUP BY` are part of what is being tested.
Shortcuts I took: no authentication, no structured request logging or metrics, no frontend tests,
and the UI re-fetches after each change rather than caching. With more time I'd add those, plus an
OpenAPI spec, CI running the tests against a Postgres service container, and container images for
the API and UI.

### Assumptions

- `id` is an auto-incrementing integer; UUIDs weren't needed for a single-database app.
- A new ticket may be created with any status, but defaults to `open` / `medium`.
- A blank description is stored as `NULL`.
- `limit` above 100 is rejected rather than silently clamped.
- Stats count all tickets and ignore the list's filter and pagination.
