## 1. Enforce lint warnings
- [ ] 1.1 `eslint . --max-warnings 0` — **deferred to a follow-up.** The current tree has 19 warnings: ~11 legitimate `no-console` in build scripts / MSW mocks (fixable via eslint overrides) and ~8 `complexity`/`max-params` in real components (`EventDetailSheet`, `EventCalendar`, `Shell`, …) that need genuine refactors or justified per-site disables. Enforcing 0 without triaging these would either break CI or bury unjustified disables; split out so it doesn't block the safe doc/config wins here (design.md pre-authorizes this).

## 2. Pre-commit backend
- [x] 2.1 Extended `.husky/pre-commit` to `gofmt -l` staged `backend/**/*.go` (fails on unformatted) and run `golangci-lint run ./...` when the tool is installed — Go quality now enforced locally, not only in CI

## 3. TypeScript strictness
- [ ] 3.1 `noUncheckedIndexedAccess` — **deferred to a follow-up.** A trial enable surfaces **136** type errors across the state/`Record`-access layer; too broad to fold in safely here without risking regressions. Tracked as its own change per design.md ("if too broad, split into its own follow-up rather than blocking the rest").
- [ ] 3.2 `exactOptionalPropertyTypes` — deferred with 3.1 (same triage)

## 4. License
- [x] 4.1 Filled `LICENSE` copyright (`Copyright 2026 yoadey`); added `"license": "Apache-2.0"` to `frontend/package.json`

## 5. Docs
- [x] 5.1 Added a "Full-Stack lokal" section to `README.md` (`cp .env.example .env` → `make install` → `make dev`, plus the backend-only path and ports)
- [x] 5.2 Synced `.env.example` with CLAUDE.md's env table (`COOKIE_ENCRYPTION_KEYS`, `PAGINATION_HMAC_KEY`, `LOG_LEVEL`, `METRICS_ALLOW_OPEN`, `RETENTION_*`, `API_DEPRECATION_DATE`, `S3_*`); added the `GOMEMLIMIT` row to the CLAUDE.md table

## 6. Verification
- [x] 6.1 `frontend/package.json` remains valid JSON; typecheck unaffected by the metadata-only `license` field
- [x] 6.2 Pre-commit hook is shell-lint clean and no-ops when no Go files are staged (frontend-only commits unaffected)
