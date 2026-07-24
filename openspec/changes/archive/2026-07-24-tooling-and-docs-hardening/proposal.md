## Why

Low-risk tooling and documentation findings from the audit remain:

- **ESLint warnings don't block CI** — `"lint": "eslint ."` has no `--max-warnings 0`, so `no-explicit-any`/`complexity`/`no-console` warnings can accumulate unbounded.
- **Pre-commit hook covers only the frontend** — `.husky/pre-commit` runs `cd frontend && npx lint-staged`; Go formatting/lint issues surface only in CI, contradicting CLAUDE.md's "commits enforce quality" claim.
- **TypeScript strictness leaves value** — `tsconfig.app.json` has `strict: true` but not `noUncheckedIndexedAccess` / `exactOptionalPropertyTypes`, relevant given the many `Record` accesses in the state layer.
- **`LICENSE` copyright placeholder** — stock Apache-2.0 with `Copyright [yyyy] [name of copyright owner]` unfilled; license not referenced in `package.json`.
- **README lacks the full-stack path** — only the frontend quick-start; `docker compose up`, the backend, and `make install`/`make dev` are undocumented for a new contributor reading the repo's front door.

## What Changes

- `"lint": "eslint . --max-warnings 0"`; fix or ratchet any surfaced warnings.
- Extend the pre-commit hook to run Go formatting/lint on staged Go files.
- Enable `noUncheckedIndexedAccess` (and evaluate `exactOptionalPropertyTypes`); fix resulting type errors.
- Fill the `LICENSE` copyright line; add `"license": "Apache-2.0"` to both `package.json`s.
- Add a full-stack section to `README.md` (compose, `make install`/`make dev`, ports) and sync `.env.example` with the documented env vars.

## Capabilities

### New Capabilities
- `dev-tooling`: enforced lint/format gates locally and in CI, stricter typing, and accurate onboarding docs.

### Modified Capabilities
<!-- none -->

## Impact

- `frontend/package.json`, `.husky/pre-commit`, `frontend/tsconfig.app.json`, `LICENSE`, `README.md`, `.env.example`, `backend/package.json`? (n/a), `CLAUDE.md`. Possibly source fixes for surfaced lint/type errors.
- CI: frontend lint/typecheck must stay green after the ratchet.
