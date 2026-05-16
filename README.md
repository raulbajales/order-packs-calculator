# Order Packs Calculator

Given a set of pack sizes and an order quantity, the app calculates the optimal pack distribution that:

1. Ships at least the requested quantity
2. Minimises excess items first
3. For equal excess, minimises the number of packs

---

## TOC

- [Diagrams](#diagrams)
- [Quick start](#quick-start-docker)
- [Local setup](#local-setup)
- [Make targets](#make-targets)
- [Project folder structure](#project-folder-structure)
- [API](#api)
- [Algorithm](#algorithm)
- [Env vars](#env-vars)

---

## Diagrams

### Components

```mermaid
graph LR
    FE["Frontend"]
    API["REST API"]
    DB["PostgreSQL"]

    FE -->|HTTP Requests| API
    API -->|SQL Queries| DB
    DB -->|Query Results| API
    API -->|JSON Response| FE
```

### Workflows

#### 1. Read Packs (on page load)

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant API as REST API
    participant DB as PostgreSQL

    FE->>API: GET /api/packs
    API->>DB: SELECT sizes FROM packs
    DB-->>API: [5000, 2000, 1000, 500, 250]
    API-->>FE: {"sizes": [...]}
    FE->>FE: Render pack list
```

#### 2. Store/Update Packs

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant API as REST API
    participant DB as PostgreSQL

    FE->>API: POST /api/packs
    API->>API: Validate input
    API->>DB: Uopdate packs
    DB-->>API: OK
    API-->>FE: {"ok": true}
    FE->>FE: Update UI
```

#### 3. Calculate Optimal Distribution

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant API as REST API
    participant DB as PostgreSQL

    FE->>API: GET /api/calculate?amount=1500
    API->>DB: SELECT sizes FROM packs
    DB-->>API: [5000, 2000, 1000, 500, 250]
    API->>API: Algorithm to find distribution
    API-->>FE: {"5000": 0, "2000": 1, "1000": 0, ...}
    FE->>FE: Render result
```

---

## Quick start (Docker)

```bash
make docker-up
```

Then open <http://localhost:8080>.

(Migrations run automatically at startup)

---

## Local setup

### Requisites

- Go 1.25+
- PostgreSQL 14+

Open <http://localhost:8080>.

## Make targets

- `make run` — start the development server (uses `DB_URL`)
- `make build` — compile the binary to `./bin/web`
- `make test` — run unit tests
- `make integration-test` — run all tests including handler integration tests
  - requires Postgres, you can run `make docker-up`
- `make docker-up` — build and start the app + Postgres via docker-compose (detached)
- `make docker-down` — stop and remove all containers

---

## Project folder structure

- `cmd/web/main.go` — entry point
- `internal/`
  - `app/app.go` — app init, DB wiring, graceful shutdown
  - `config/config.go` — environment-variable config
  - `db/`
    - `db.go` — embeds migration files into the binary
    - `migrations/` — Goose SQL migration files
  - `http/`
    - `router.go` — chi router setup
    - `handlers.go` — HTTP handlers
    - `handlers_test.go` — integration tests via httptest
  - `models/pack.go` — Pack domain model
  - `repository/pack_repository.go` — Postgres queries
  - `service/`
    - `calculator.go` — core algorithm (dynamic programming)
    - `calculator_test.go` — unit tests
- `web/`
  - `index.html` — main page, static HTML served from disk
  - `app.css` — stylesheet

### Libraries used

- **Standard library**
  - `database/sql` — database interface
  - `net/http` — HTTP server
  - `log/slog` — structured logging
  - `embed` — embeds migration files into the binary at build time
  - `os` — filesystem access for web assets
- **Third-party**
  - [`chi`](https://github.com/go-chi/chi) — HTTP router and middleware
  - [`pgx`](https://github.com/jackc/pgx) — PostgreSQL driver
  - [`goose`](https://github.com/pressly/goose) — DB migrations

---

## API

All endpoints return `application/json`.

---

### `GET /api/health`

Liveness check.

- **Response** `200` — `{"status":"ok"}`

---

### `GET /api/packs`

Returns all configured pack sizes in descending order.

- **Response** `200` — `{"sizes":[5000,2000,1000,500,250]}`
- **Response** `500` — `{"error":"..."}` — database error

---

### `POST /api/packs`

Replaces all pack sizes atomically.

- **Request body** `application/json`
  - `sizes` _(string, required)_ — comma-separated positive integers, e.g. `"250,500,1000"`
- **Response** `200` — `{"ok":true}`
- **Response** `400` — `{"error":"..."}` — invalid JSON or validation failure
  - non-numeric token
  - non-positive number
  - empty input
- **Response** `500` — `{"error":"..."}` — database error

---

### `GET /api/calculate`

Returns the optimal pack distribution for a given order quantity.

- **Query params**
  - `amount` _(integer, required)_ — order quantity, must be a positive integer ≤ 1,000,000
- **Response** `200` — `{"<packSize>": <quantity>, …}` — keys are pack sizes as strings
- **Response** `400` — `{"error":"..."}` — missing or invalid `amount`
- **Response** `422` — `{"error":"..."}` — no valid pack combination found
- **Response** `500` — `{"error":"..."}` — database error

---

## Algorithm

The calculator uses dynamic programming:

- `dp[i]` = minimum number of packs to ship exactly `i` items
- Scans targets from `order` to `order + minPackSize`
- Returns the first reachable target (= minimum excess)
- At that target `dp[T]` is already the minimum pack count

Complexity: O((order + minPackSize) × numSizes) time

---

## Env vars

- `PORT` — HTTP server port _(default: `8080`)_
- `DATABASE_URL` — PostgreSQL connection string _(default: `postgres://postgres:postgres@localhost:5432/challenge?sslmode=disable`)_
- `WEB_DIR` — path to the directory containing `index.html` and `app.css` _(default: `./web`)_
