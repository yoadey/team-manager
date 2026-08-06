## Why

Two independent per-process resource limits don't account for
`values-prod.yaml`'s autoscaling range (up to 20 backend replicas):

1. **DB connection pool.** `backend/internal/db/db.go:47-48` hardcodes
   `MaxConns = 25`, `MinConns = 2` at compile time — not env-configurable.
   At max scale that's up to 20 × 25 = 500 backend connections
   (transiently more during a rolling deploy, since old and new pods
   coexist), against Postgres's own default `max_connections=100`.
   Nothing in CLAUDE.md's env-var table, `docs/operations.md`, or the
   Helm values documents a required `max_connections` sizing or mandates
   a connection pooler — the chart only has an offhand comment
   (`values.yaml:441`) that a pooler is *possible*, not required.
2. **Per-IP rate limiting.** `middleware.go:293-313`'s `RateLimit`/
   `PerIPRateLimit` use `httprate`'s in-memory, per-process counter (a
   deliberate tradeoff per its own comment, for the general
   `RATE_LIMIT_RPS` limit). The same mechanism also backs
   `LOGIN_RATE_LIMIT_PER_MIN`/`REGISTER_RATE_LIMIT_PER_MIN`/
   `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN` — the brute-force defenses. At
   3-20 replica autoscaling, an attacker whose requests land across pods
   effectively gets up to `limit × replicas-hit` attempts per minute
   instead of the documented `limit`, and this degradation isn't
   mentioned anywhere the documented values are.

## What Changes

- Make the DB pool size configurable via `DB_MAX_CONNS`/`DB_MIN_CONNS`
  env vars (defaulting to today's `25`/`2`), and document the required
  Postgres `max_connections` (or mandatory pooler) as a function of
  `autoscaling.maxReplicas` in CLAUDE.md and `docs/operations.md`.
- Document, next to `LOGIN_RATE_LIMIT_PER_MIN` and friends in CLAUDE.md's
  env-var table, that these limits are per-process/per-pod and therefore
  scale with replica count — so operators don't over-trust the
  documented number as a hard global ceiling. (Moving brute-force rate
  limiting to a shared store, e.g. Redis, is a larger architectural
  change that conflicts with the project's stated "deliberately
  dependency-light" convention — out of scope here; this change makes
  the current, real behavior explicit rather than implementing a fix.)

## Capabilities

### Modified Capabilities
- `helm-deployment`: the backend's DB connection pool size is
  configurable and documented relative to `autoscaling.maxReplicas` and
  Postgres's `max_connections`.

## Impact

- `backend/internal/db/db.go`, `backend/internal/config/config.go`
  (+ tests).
- `CLAUDE.md` (env-var table), `docs/operations.md`.
- `helm/team-manager/values-prod.yaml` / `values.yaml` — no schema
  change required unless the new env vars should be chart-configurable
  too (decide during implementation; at minimum document the sizing
  relationship).
