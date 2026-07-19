## 1. Enforce lint warnings
- [ ] 1.1 Change `"lint"` to `eslint . --max-warnings 0`; fix or justifiably scope-disable surfaced warnings

## 2. Pre-commit backend
- [ ] 2.1 Extend `.husky/pre-commit` to run `gofumpt`/`golangci-lint` on staged `*.go` (keep it fast)

## 3. TypeScript strictness
- [ ] 3.1 Enable `noUncheckedIndexedAccess` in `tsconfig.app.json`; fix resulting type errors
- [ ] 3.2 Evaluate `exactOptionalPropertyTypes`; enable if the fix cost is bounded, else note as follow-up

## 4. License
- [ ] 4.1 Fill the `LICENSE` copyright line; add `"license": "Apache-2.0"` to `frontend/package.json` (and root if applicable)

## 5. Docs
- [ ] 5.1 Add a "Full stack locally" section to `README.md` (`make install`/`make dev`/compose, ports)
- [ ] 5.2 Sync `.env.example` with CLAUDE.md's env table (`COOKIE_ENCRYPTION_KEYS`, `PAGINATION_HMAC_KEY`, `LOG_LEVEL`, `RETENTION_*`); add `GOMEMLIMIT` to the CLAUDE.md table

## 6. Verification
- [ ] 6.1 `npm run lint` (zero warnings) + `npm run typecheck` + `npm test` green
- [ ] 6.2 `npm run build` + `check:bundle` under budget
- [ ] 6.3 Pre-commit hook exercised on a staged Go change
