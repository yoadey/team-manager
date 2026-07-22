## 1. Enforce lint warnings
- [x] 1.1 `eslint . --max-warnings 0` — implemented. Of the tree's 19 warnings: the ~10 `no-console` warnings in `scripts/**/*.mjs` got a targeted `rules: { 'no-console': 'off' }` override (legitimate CLI output, not app logging) plus one now-unused `eslint-disable` comment removed; the unused `statSync` import in `check-bundle-size.mjs` was dropped. The remaining 8 `complexity`/`max-params` violations (`EventCalendar`, `EventDetailSheet`, `EventFormSheet`, `eventFormSchema.ts`, `AppShell`'s `Shell`, `mocks/db.ts`'s `perms`/attendance-seeding helper) were fixed with genuine refactors -- extracted sub-components/helper functions and options-object parameters -- not threshold changes or blanket disables. `frontend/package.json`'s `"lint"` script now runs `eslint . --max-warnings 0`.

## 2. Pre-commit backend
- [x] 2.1 Extended `.husky/pre-commit` to `gofmt -l` staged `backend/**/*.go` (fails on unformatted) and run `golangci-lint run ./...` when the tool is installed — Go quality now enforced locally, not only in CI

## 3. TypeScript strictness
- [x] 3.1 `noUncheckedIndexedAccess` — implemented, added to `frontend/tsconfig.app.json`. Fixed alongside 3.2 below (enabling both together surfaced 251 errors across ~50 files); array/`Record` accesses now `T | undefined` were fixed with narrowing guards, `??` defaults, or by restructuring the lookup so a fallback is bound to a real object/tuple literal instead of re-indexed (`THEME_PRESETS`/`typeMeta`/`statusMeta`/`pageMeta`'s per-route defs, `mocks/db.ts`'s seeded roles). A handful of `!` non-null assertions were kept, each immediately after a `.length`/emptiness check on the same array (mocks/tests only).
- [x] 3.2 `exactOptionalPropertyTypes` — implemented alongside 3.1 (evaluated together since the fix cost turned out bounded, not deferred separately). Object literals assigning `undefined` to an optional property were fixed either by omitting the key when absent (a small `opt(key, value)` helper spreads `{ [key]: value }` only when defined, added to `mocks/handlers.ts`/`api/map.ts`/`services/serviceLayerReal.ts`) or, where the call site's existing intent was a genuinely meaningful `undefined` (not "omit this key") -- e.g. `SaveTxInput.id`/`SaveMemberInput.photo` picking create-vs-edit, `SheetState.eventId`/`member`, `Field.errorText`, `Av`'s `name`/`photo`/`color`, `AppNotification.eventDate` -- by widening the type to `field?: T | undefined` instead.

## 4. License
- [x] 4.1 Filled `LICENSE` copyright (`Copyright 2026 yoadey`); added `"license": "Apache-2.0"` to `frontend/package.json`

## 5. Docs
- [x] 5.1 Added a "Full-Stack lokal" section to `README.md` (`cp .env.example .env` → `make install` → `make dev`, plus the backend-only path and ports)
- [x] 5.2 Synced `.env.example` with CLAUDE.md's env table (`COOKIE_ENCRYPTION_KEYS`, `PAGINATION_HMAC_KEY`, `LOG_LEVEL`, `METRICS_ALLOW_OPEN`, `RETENTION_*`, `API_DEPRECATION_DATE`, `S3_*`); added the `GOMEMLIMIT` row to the CLAUDE.md table

## 6. Verification
- [x] 6.1 `frontend/package.json` remains valid JSON; typecheck unaffected by the metadata-only `license` field
- [x] 6.2 Pre-commit hook is shell-lint clean and no-ops when no Go files are staged (frontend-only commits unaffected)
