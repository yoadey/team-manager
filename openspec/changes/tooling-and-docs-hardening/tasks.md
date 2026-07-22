## 1. Enforce lint warnings
- [ ] 1.1 `eslint . --max-warnings 0` — the tree currently has 27 warnings:
      `no-console`/an unused var in Node build scripts (legitimate there;
      scope an eslint override rather than removing them), and
      `complexity`/`max-params` in real components (`EventCalendar`,
      `EventDetailSheet`, `EventFormSheet`, `eventFormSchema.ts`,
      `AppShell.tsx`'s `Shell`, `mocks/db.ts`) that need genuine refactors.
      No longer deferred — see `openspec/changes/alpha-initial-setup` for the
      unrelated alpha-release cleanup this rides alongside.

## 2. Pre-commit backend
- [x] 2.1 Extended `.husky/pre-commit` to `gofmt -l` staged `backend/**/*.go` (fails on unformatted) and run `golangci-lint run ./...` when the tool is installed — Go quality now enforced locally, not only in CI

## 3. TypeScript strictness
- [ ] 3.1 `noUncheckedIndexedAccess` — enabling both this and 3.2 together
      surfaces 251 errors across ~50 files (`src/mocks/handlers.ts`,
      `src/services/serviceLayerReal.ts`, `src/api/map.ts`, etc.); fix with
      guards/`?.`/narrowing, not blanket non-null assertions. No longer
      deferred.
- [ ] 3.2 `exactOptionalPropertyTypes` — enabled together with 3.1 (fixing
      them separately would require re-deriving which errors belong to
      which flag; both land in the same pass)

## 4. License
- [x] 4.1 Filled `LICENSE` copyright (`Copyright 2026 yoadey`); added `"license": "Apache-2.0"` to `frontend/package.json`

## 5. Docs
- [x] 5.1 Added a "Full-Stack lokal" section to `README.md` (`cp .env.example .env` → `make install` → `make dev`, plus the backend-only path and ports)
- [x] 5.2 Synced `.env.example` with CLAUDE.md's env table (`COOKIE_ENCRYPTION_KEYS`, `PAGINATION_HMAC_KEY`, `LOG_LEVEL`, `METRICS_ALLOW_OPEN`, `RETENTION_*`, `API_DEPRECATION_DATE`, `S3_*`); added the `GOMEMLIMIT` row to the CLAUDE.md table

## 6. Verification
- [x] 6.1 `frontend/package.json` remains valid JSON; typecheck unaffected by the metadata-only `license` field
- [x] 6.2 Pre-commit hook is shell-lint clean and no-ops when no Go files are staged (frontend-only commits unaffected)
