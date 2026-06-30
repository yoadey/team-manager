# Teamverwaltung — Developer Guide (Monorepo)

This is a monorepo containing the React frontend and Go backend for the Teamverwaltung sports-club management application.

## Quick Start

```bash
# Frontend (in /frontend)
cd frontend && npm install
npm run dev          # http://localhost:5173
npm test             # run all tests once
npm run typecheck    # TypeScript check
npm run lint         # ESLint

# Backend (in /backend)
cd backend && make tools   # install go tools (once)
make generate              # regenerate from openapi.yaml
make build                 # compile ./cmd/server
make test                  # go test ./...
make lint                  # golangci-lint

# Full stack (Docker Compose)
docker compose up          # Postgres + Backend + Frontend
```

## Repository Structure

```
team-manager/
├── frontend/              React 18 + TypeScript SPA
│   ├── src/               Application source
│   │   ├── services/serviceLayer.ts   Mock backend (replace with real API)
│   │   └── ...
│   ├── package.json
│   └── vite.config.ts
├── backend/               Go REST API
│   ├── cmd/server/main.go Entry point
│   ├── internal/
│   │   ├── auth/          Auth module (password login, JWT, OIDC-ready)
│   │   ├── teams/         Teams, invites
│   │   ├── members/       Team members
│   │   ├── roles/         RBAC roles and permissions
│   │   ├── events/        Events, series, attendance, comments
│   │   ├── absences/      Planned absences
│   │   ├── news/          Team news
│   │   ├── polls/         Polls and voting
│   │   ├── notifications/ Activity feed
│   │   ├── finances/      Transactions, penalties, contributions
│   │   ├── stats/         Attendance statistics
│   │   ├── server/        Aggregator (implements StrictServerInterface)
│   │   ├── gen/           oapi-codegen generated types (DO NOT EDIT)
│   │   ├── db/            DB pool + migration runner
│   │   ├── middleware/    HTTP middleware (auth, logging, CORS, rate-limit)
│   │   ├── apierror/      RFC 9457 Problem Details
│   │   ├── config/        Environment config
│   │   └── testutil/      Test helpers (testcontainers)
│   ├── openapi/openapi.yaml  Source of truth for API contract
│   ├── go.mod
│   └── Makefile
├── docker-compose.yml     Local dev: Postgres + Backend + Frontend
├── .github/workflows/ci.yml  CI: Frontend + Backend jobs
└── CLAUDE.md
```

## Architecture

### Frontend

- **React 18** + **TypeScript 5** (strict mode)
- **Material UI v6** for components, **Emotion** for styling
- **Vite 6** for bundling, **Vitest 2** for tests
- **State-based routing** (no router dependency; navigation driven by `state.route`)
- **i18n** via lightweight in-house layer (`src/i18n`)
- All state in `src/context/AppContext.tsx`; access via `useApp()`
- Mock backend at `src/services/serviceLayer.ts` — replace bodies with `fetch()` to connect real API

### Backend

- **Go 1.24+** with **Chi v5** router
- **PostgreSQL 17** via **pgx/v5**; migrations via **goose**
- **Spec-first**: `openapi/openapi.yaml` → `oapi-codegen` → `internal/gen/api.gen.go`; never edit gen manually
- **JWT (RS256)** session management; keys configurable via env; auto-generates dev keys when empty
- **Layered architecture** per feature: `handler.go` → `service.go` → `repository.go`
- TDD: tests live alongside source (`*_test.go`)

### RBAC

Each team member has roles; each role has per-module permission levels (`none | read | write`). Modules: `events`, `members`, `finances`, `news`, `polls`, `settings`. Permissions are stored as JSONB in Postgres.

## OpenAPI Contract

`backend/openapi/openapi.yaml` is the source of truth. After editing it:

```bash
cd backend && make generate  # regenerates internal/gen/api.gen.go
```

The TypeScript client is also generated from this spec (future: `openapi-typescript` + `openapi-fetch`).

## Environment Variables

### Frontend (`frontend/.env`)

| Variable                  | Default          | Purpose                          |
|---------------------------|------------------|----------------------------------|
| `VITE_APP_NAME`           | `Teamverwaltung` | Browser title                    |
| `VITE_STORAGE_KEY_PREFIX` | `tv_db_`         | localStorage prefix (mock DB)    |
| `VITE_MOCK_DELAY_MIN/MAX` | `120` / `320`    | Simulated latency (ms)           |
| `VITE_SENTRY_DSN`         | _(empty)_        | Sentry; disabled when empty      |
| `VITE_API_BASE_URL`       | _(empty)_        | Real backend URL                 |

### Backend

| Variable          | Default                     | Purpose                        |
|-------------------|-----------------------------|--------------------------------|
| `DATABASE_URL`    | _(required)_                | PostgreSQL DSN                 |
| `PORT`            | `8080`                      | HTTP port                      |
| `ALLOWED_ORIGINS` | `http://localhost:5173`     | CORS whitelist                 |
| `PUBLIC_BASE_URL` | _(first `ALLOWED_ORIGINS` entry)_ | Public frontend origin for shareable invite links (e.g. `https://app.example.com`); trailing slash trimmed |
| `JWT_PRIVATE_KEY` | _(auto-generated in dev)_   | RSA-2048 private key PEM       |
| `JWT_PUBLIC_KEY`  | _(auto-generated in dev)_   | RSA-2048 public key PEM        |
| `SESSION_TTL_HOURS`| `720`                      | Session lifetime (30 days)     |
| `MIGRATIONS_DIR`  | `internal/db/migrations`    | Goose migrations directory     |
| `COOKIE_ENCRYPTION_KEYS`| _(empty)_ | Comma-separated list of AES-256 keys (newest first) for zero-downtime rotation. Takes precedence over `COOKIE_ENCRYPTION_KEY`. Each key: 32 bytes, hex or base64. |
| `COOKIE_ENCRYPTION_KEY`| _(auto-generated in dev)_ | Single AES-256 key (32 bytes, hex or base64). Used when `COOKIE_ENCRYPTION_KEYS` is unset. **Required when `COOKIE_SECURE=true`** — startup fails without it. Generate with `openssl rand -base64 32`. |
| `COOKIE_SECURE`   | `true`                      | Cookie `Secure` flag; set `false` for local http |
| `COOKIE_NAME`     | `tv_session`                | Session cookie name (override only if needed) |
| `METRICS_TOKEN`   | _(empty)_                   | Bearer token guarding `/metrics`; open when unset. **Recommended in production** — a warning is logged at startup when unset with `COOKIE_SECURE=true`. |
| `RATE_LIMIT_RPS`  | `100`                       | Global per-IP request rate limit (requests per second). |
| `LOGIN_RATE_LIMIT_PER_MIN` | `5`              | Per-IP login attempt limit per minute (brute-force protection). |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(empty)_       | OTLP/HTTP collector URL; enables OpenTelemetry tracing when set (other `OTEL_*` vars honored by the SDK) |
| `OTEL_SERVICE_NAME` | `team-manager-backend`    | Service name reported in traces |
| `SENTRY_DSN`      | _(empty)_                   | Sentry DSN for backend error tracking; disabled when empty |
| `ENVIRONMENT`     | _(empty)_                   | Environment label attached to Sentry events |

> **Key rotation:** Use `COOKIE_ENCRYPTION_KEYS` (plural) for zero-downtime rotation. Set
> the new key first, then append the old key(s): `COOKIE_ENCRYPTION_KEYS=<new>,<old>`.
> Encryption always uses the first key; decryption tries all keys in order. Old keys can
> be removed once all sessions using them have expired (after `SESSION_TTL_HOURS`).
> Generate a new key with `openssl rand -base64 32`.

## Testing

### Frontend

```bash
cd frontend
npm test                  # single run
npm run test:watch        # watch mode
npm run test:coverage     # coverage report
```

### Backend

```bash
cd backend
make test                 # all tests (integration tests skip if no Docker)
make test-unit            # unit tests only (-short flag)
make test-integration     # requires Docker for testcontainers
```

Integration tests use `testutil.NewTestDB(t)` which spins up a `postgres:17` testcontainer and runs migrations. Tests are automatically skipped when Docker is not available.

## Code Quality

- **Frontend**: lint-staged via Husky (ESLint + Prettier); CI runs lint → typecheck → test → build
- **Backend**: `golangci-lint`; CI runs lint → test → build + `govulncheck`
- **Commits** enforce quality via pre-commit hooks

## Connecting the Real Backend

When replacing the mock frontend with real API calls:

1. Replace method bodies in `frontend/src/services/serviceLayer.ts` with `fetch()` calls
2. The exported `api` object shape must stay unchanged — no other frontend code changes
3. Set `VITE_API_BASE_URL` in `frontend/.env`
4. A generated TypeScript client from the OpenAPI spec is the recommended approach (see plan)
