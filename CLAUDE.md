# Teamverwaltung — Developer Guide (Monorepo)

This is a monorepo containing the React frontend and Go backend for the Teamverwaltung sports-club management application.

## Spec-Driven Development with OpenSpec (MANDATORY)

**This repository uses [OpenSpec](https://github.com/Fission-AI/OpenSpec) for all work. Every non-trivial change MUST start as an OpenSpec change proposal — no exceptions.** Do not begin implementing a feature, refactor, bugfix, or dependency change by editing source directly; first capture it as a change under `openspec/changes/`, then implement against its tasks.

The `openspec/` directory is the source of truth for *planned* work; the OpenSpec `specs/` capture *built* capabilities. Convention lives in `openspec/config.yaml`.

Directory layout:

```
openspec/
├── config.yaml                     Project context + per-artifact rules (spec-driven schema)
├── specs/<capability>/spec.md      Current built capabilities (populated on archive)
└── changes/<change-name>/          One folder per proposed change
    ├── proposal.md                 Why / What Changes / Capabilities / Impact
    ├── design.md                   Context / Goals / Decisions / Risks
    ├── tasks.md                    Numbered, checkboxed implementation steps
    └── specs/<capability>/spec.md  Delta: ## ADDED|MODIFIED|REMOVED Requirements
```

Workflow (CLI: `npx @fission-ai/openspec@latest <cmd>`; Claude Code slash commands: `/opsx:propose`, `/opsx:apply`, `/opsx:archive`):

1. **Propose** — `openspec new change "<kebab-name>"`, then author `proposal.md`, `design.md`, delta `specs/`, and `tasks.md`. Every requirement needs at least one `#### Scenario:` (WHEN/THEN). Keep tasks' final group a **Verification** checklist of the CI gates it must keep green.
2. **Validate** — `openspec validate <name> --strict` (or `--all`) MUST pass before implementation.
3. **Apply** — implement the change, ticking `tasks.md` checkboxes as you go.
4. **Archive** — `openspec archive <name>` once merged, folding the deltas into `openspec/specs/` and moving the change to `changes/archive/`.

Notes:
- The `.claude/` OpenSpec skills/commands are git-ignored (regenerate with `openspec init --tools claude`); the versioned artifacts live under `openspec/`.
- Open change proposals seeded from the architecture audit live in `openspec/changes/` (MSW, TanStack Query, object storage, sqlc, spec-generated RBAC, React Hook Form). See each change's `proposal.md`/`tasks.md`.
- OpenSpec sits *above* the existing spec-first OpenAPI workflow — it does not replace it. `backend/openapi/openapi.yaml` remains the API contract; a change that touches the API still runs `make generate` / `make generate-ts`.

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
├── frontend/              React 19 + TypeScript SPA
│   ├── src/               Application source
│   │   ├── services/serviceLayer.ts   Mock/real backend switch (see "Connecting the Real Backend")
│   │   └── ...
│   ├── package.json
│   └── vite.config.ts
├── backend/               Go REST API
│   ├── cmd/server/main.go Entry point
│   ├── cmd/healthcheck/   Docker HEALTHCHECK binary (no HTTP client at runtime)
│   ├── cmd/genrbac/       Generates internal/middleware/rbac_table.gen.go from openapi.yaml's x-rbac-* extensions
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
│   │   ├── audit/         Audit log
│   │   ├── jobs/          River-based background workers (retention, notifications)
│   │   ├── storage/       S3-compatible object store (team/user photos, team logos) — ObjectStore interface + S3/fake impls
│   │   ├── metrics/       Prometheus metrics (business + retention job)
│   │   ├── observability/ OpenTelemetry tracing + Sentry wiring
│   │   ├── pagination/    Keyset pagination + HMAC-signed cursors
│   │   ├── validate/      Shared input-validation helpers
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

- **React 19** + **TypeScript 5** (strict mode)
- **Material UI v9** for components, **Emotion** for styling
- **Vite 8** for bundling, **Vitest 4** for tests
- **State-based routing** (no router dependency; navigation driven by `state.route`)
- **i18n** via lightweight in-house layer (`src/i18n`)
- All state in `src/context/AppContext.tsx`; access via `useApp()`
- `src/services/serviceLayer.ts` exports `api`, switching between the in-memory mock and the real
  backend based on `VITE_API_BASE_URL` (see "Connecting the Real Backend" below)

### Backend

- **Go 1.25+** with **Chi v5** router
- **PostgreSQL 17** via **pgx/v5**; migrations via **goose**
- **Spec-first**: `openapi/openapi.yaml` → `oapi-codegen` → `internal/gen/api.gen.go`; never edit gen manually
- **JWT (RS256)** session management; keys configurable via env; auto-generates dev keys when empty
- **Layered architecture** per feature: `handler.go` → `service.go` → `repository.go`
- TDD: tests live alongside source (`*_test.go`)

### RBAC

Each team member has roles; each role has per-module permission levels (`none | read | write`). Modules: `events`, `members`, `finances`, `news`, `polls`, `settings`. Permissions are stored as JSONB in Postgres.

**Route-to-module mapping is generated from the spec, not hand-maintained.** Every team-scoped operation in `openapi.yaml` carries an `x-rbac-module` extension (`events | members | finances | news | polls | settings | public`) and, where applicable, `x-rbac-self-service: true`. `cmd/genrbac` (wired into `make generate`, after oapi-codegen) parses these into `internal/middleware/rbac_table.gen.go` — DO NOT EDIT; edit the extensions in `openapi.yaml` and re-run `make generate`. `genrbac` fails the build if any team-scoped operation is missing `x-rbac-module`, so a newly added route can't silently end up unclassified.

Enforcement (`internal/middleware/authz.go`, `RequirePermission`): looks up the request's method+path in the generated table via `matchRBACRoute`. **A request whose method+path matches no table entry is rejected with 404 for every HTTP method, including GET** — there is no fallback to "unrestricted". `module: public` routes (team info itself, photo, logo, absences, notifications) require nothing beyond membership, for any method. For module-gated routes, mutating requests (POST/PUT/PATCH/DELETE) require `write`, as expected — but GET requests are *also* gated, requiring at least `read`; a module set to `none` hides read access too, not just writes. Self-service routes (an event's own attendance/comments, a poll vote — `x-rbac-self-service: true`) never require `write` on their module regardless of method, but still require at least `read` where the route reads back module data (e.g. `polls/vote` returns the assembled poll) — self-service exempts a caller from `write`, not from `none`. `stats` isn't a module of its own: its GET operations carry `x-rbac-module: events`, since its data is event/attendance-derived. `notifications` has no route-level gate at all (`x-rbac-module: public`) since it aggregates across modules — instead, `notifications.Service.List` filters each returned item server-side by the caller's permission on that item's originating module.

## OpenAPI Contract

`backend/openapi/openapi.yaml` is the source of truth. After editing it:

```bash
cd backend && make generate  # regenerates internal/gen/api.gen.go
```

The TypeScript client is also generated from this spec via `openapi-typescript` + `openapi-fetch`
(`make generate-ts` at the repo root; see "Connecting the Real Backend" below).

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
| `JWT_PRIVATE_KEY` | _(auto-generated in dev)_   | RSA-2048 private key PEM. **Required when `COOKIE_SECURE=true`** (with `JWT_PUBLIC_KEY`) — startup fails without it, since an ephemeral per-process key invalidates sessions on restart and fails verification across replicas. |
| `JWT_PUBLIC_KEY`  | _(auto-generated in dev)_   | RSA-2048 public key PEM. Same requirement as `JWT_PRIVATE_KEY`. |
| `SESSION_TTL_HOURS`| `720`                      | Session lifetime (30 days)     |
| `MIGRATIONS_DIR`  | `internal/db/migrations`    | Goose migrations directory     |
| `COOKIE_ENCRYPTION_KEYS`| _(empty)_ | Comma-separated list of AES-256 keys (newest first) for zero-downtime rotation. Takes precedence over `COOKIE_ENCRYPTION_KEY`. Each key: 32 bytes, hex or base64. |
| `COOKIE_ENCRYPTION_KEY`| _(auto-generated in dev)_ | Single AES-256 key (32 bytes, hex or base64). Used when `COOKIE_ENCRYPTION_KEYS` is unset. **Required when `COOKIE_SECURE=true`** — startup fails without it. Generate with `openssl rand -base64 32`. |
| `COOKIE_SECURE`   | `true`                      | Cookie `Secure` flag; set `false` for local http |
| `COOKIE_NAME`     | `tv_session`                | Session cookie name (override only if needed) |
| `METRICS_TOKEN`   | _(empty)_                   | Bearer token guarding `/metrics`; open when unset. **Recommended in production** — a warning is logged at startup when unset with `COOKIE_SECURE=true`. |
| `METRICS_ALLOW_OPEN` | `false`                   | Set `true` to allow startup with an open, unauthenticated `/metrics` when `COOKIE_SECURE=true` and `METRICS_TOKEN` is unset (otherwise startup fails). Use only when `/metrics` is restricted at the network layer instead. |
| `RATE_LIMIT_RPS`  | `100`                       | Global per-IP request rate limit (requests per second). |
| `LOGIN_RATE_LIMIT_PER_MIN` | `5`              | Per-IP login attempt limit per minute (brute-force protection). |
| `TRUSTED_PROXY_CIDRS` | _(empty)_                | Comma-separated CIDRs of reverse proxies/load balancers allowed to set `X-Forwarded-For`/`X-Real-IP`/`True-Client-IP` for rate limiting. Empty (default) trusts nothing — rate limiting keys on the raw TCP peer address, so header spoofing cannot bypass it. **Set this when deploying behind a reverse proxy/LB**, or the real clients behind it will all share one rate-limit bucket (the proxy's IP). |
| `PAGINATION_HMAC_KEY` | _(empty)_               | AES-256-equivalent key (32 bytes, hex or base64) that HMAC-signs keyset pagination cursors so clients can't craft arbitrary ones. Optional — unsigned (plain base64) cursors are used when unset; a warning is logged at startup when unset with `COOKIE_SECURE=true`. |
| `RETENTION_NOTIFICATIONS_DAYS` | `90`         | How many days to keep notification rows before the daily retention job deletes them. |
| `RETENTION_SESSIONS_DAYS` | `30`               | How many days past expiry to keep session rows before the daily retention job deletes them. |
| `RETENTION_AUDIT_LOG_DAYS` | `365`             | How many days to keep `audit_log` rows before the daily retention job deletes them. Compliance-relevant — raise if your retention policy requires longer. |
| `S3_ENDPOINT`     | _(empty)_                   | S3-compatible host for image object storage (team/user photos, team logos), e.g. `s3.eu-central-1.amazonaws.com` or `minio:9000`; optionally prefixed `http://`/`https://` (defaults to secure). **Required when `COOKIE_SECURE=true`** (with `S3_BUCKET`/`S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY`) — startup fails without it. Unset in dev falls back to an in-memory fake store (images don't survive a restart). |
| `S3_REGION`       | _(empty)_                   | Object store region, e.g. `eu-central-1`. May be blank for MinIO/region-less endpoints. |
| `S3_BUCKET`       | _(empty)_                   | Bucket image objects are stored in. Same `COOKIE_SECURE=true` requirement as `S3_ENDPOINT`. |
| `S3_ACCESS_KEY_ID` / `S3_SECRET_ACCESS_KEY` | _(empty)_ | Static credentials for the object store. Same `COOKIE_SECURE=true` requirement as `S3_ENDPOINT`. |
| `S3_USE_PATH_STYLE` | `false`                    | Force path-style bucket addressing (`https://host/bucket/key`) instead of virtual-hosted-style. Set `true` for most self-hosted S3-compatible stores (MinIO); leave `false` for real AWS S3. |
| `S3_PUBLIC_BASE_URL` | _(empty)_                | Overrides the scheme+host of presigned image URLs after signing. Needed when the backend's S3 endpoint (e.g. in-cluster/Compose service DNS) differs from the endpoint a browser can reach. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(empty)_       | OTLP/HTTP collector URL; enables OpenTelemetry tracing when set (other `OTEL_*` vars honored by the SDK) |
| `OTEL_SERVICE_NAME` | `team-manager-backend`    | Service name reported in traces |
| `SENTRY_DSN`      | _(empty)_                   | Sentry DSN for backend error tracking; disabled when empty |
| `ENVIRONMENT`     | _(empty)_                   | Environment label attached to Sentry events |
| `ERROR_TYPE_BASE_URI` | _(empty)_               | Base URI prefix for the `type` field of RFC 9457 problem+json error responses (e.g. `https://docs.example.com/errors`); left as relative paths when unset. |
| `LOG_LEVEL`       | `info`                      | Minimum level the JSON structured logger emits (`debug`\|`info`\|`warn`\|`error`, case-insensitive). An unrecognized value falls back to `info` rather than failing startup. |
| `API_DEPRECATION_DATE` | _(empty)_              | When set, emitted as both the RFC 8594 `Deprecation` and `Sunset` response headers on every request, so API clients can programmatically detect a pending deprecation window. Any string is passed through verbatim (e.g. `@1735689600` or an HTTP-date) — no format validation. |
| `GOMEMLIMIT`      | _(container limit)_         | Go soft memory limit. In Kubernetes the Helm chart derives it from the container memory limit via the Downward API and applies a headroom factor in `cmd/server/main.go` (`applyMemoryLimitHeadroom`). **Only the raw-byte form is honored by the headroom override** — a suffixed value like `256MiB` set by hand bypasses it. |

> **Key rotation:** Use `COOKIE_ENCRYPTION_KEYS` (plural) for zero-downtime rotation.
> Encryption always uses the *first* key; decryption tries all keys in order. Like
> `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` (see docs/operations.md's JWT key rotation runbook),
> this env var is only read once at process start — updating the Secret alone does
> nothing until pods are restarted, and a *single* rolling restart straight to
> `<new>,<old>` is not actually zero-downtime: mid-rollout, already-restarted pods
> encrypt new cookies with `<new>` while not-yet-restarted pods have never loaded
> `<new>` and reject them, forcing re-login for any user whose requests land on both
> pod generations. The genuinely zero-downtime sequence is **two** rolling restarts:
> 1. Set `COOKIE_ENCRYPTION_KEYS=<old>,<new>` (new key appended, not prepended) and
>    restart every pod — every replica can now decrypt both, but all still encrypt
>    with `<old>`, so nothing changes for existing/new cookies yet.
> 2. Once step 1 has fully rolled out, flip to `COOKIE_ENCRYPTION_KEYS=<new>,<old>`
>    and restart every pod again — this is the actual cutover to encrypting with
>    `<new>`; every replica already knows `<new>` from step 1, so no pod ever rejects
>    a cookie encrypted by another.
> Old keys can be removed once all sessions using them have expired (after
> `SESSION_TTL_HOURS`). Generate a new key with `openssl rand -base64 32`.

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

The real backend integration is already implemented, not a future step:

- `frontend/src/services/serviceLayerReal.ts` implements the full API contract against
  the Go backend, using a generated TypeScript client (`frontend/src/api/`) from
  `backend/openapi/openapi.yaml`.
- `frontend/src/services/serviceLayer.ts` exports `api`, which resolves to `realApi`
  when `VITE_API_BASE_URL` is set (see `frontend/.env`) and falls back to the in-memory
  mock (`localStorage`-backed) otherwise — no other frontend code needs to change either
  way, since both implementations satisfy the same `api` shape.
- `frontend/src/services/serviceContract.test.ts` cross-tests both implementations
  against the same contract to keep them from drifting apart.

When the OpenAPI spec changes, regenerate both clients from the repo root:

```bash
make generate     # internal/gen/api.gen.go
make generate-ts  # frontend/src/api/types.gen.ts (via openapi-typescript)
```

`frontend/src/api/types.gen.ts` is consumed by the `openapi-fetch` client in
`frontend/src/api/client.ts`. CI's `backend-openapi-drift` job runs both generators and fails the
build if the checked-in output doesn't match `backend/openapi/openapi.yaml`.
