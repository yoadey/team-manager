## Context

The repo is deliberately strict elsewhere (golangci-lint, coverage gates, drift checks) but the frontend lint gate and local Go hooks lag. `noUncheckedIndexedAccess` will surface real `Record`/array-access assumptions.

## Goals / Non-Goals

**Goals:**
- Warnings are enforced, not advisory; Go quality is checked pre-commit; typing is stricter; docs let a newcomer run the full stack; the license is complete.

**Non-Goals:**
- A broad `no-explicit-any` purge — ratchet incrementally; only make surfaced blockers pass.
- Reworking the docs site or adding rendered API docs (separate).

## Decisions

- `--max-warnings 0` in the lint script; where the current tree has warnings, fix the cheap ones and, if needed, scope-disable with justification (the repo already uses justified disables).
- Pre-commit: add a Go step for staged `*.go` (`gofumpt -w` + `golangci-lint run --fast` on changed packages, or move lint-staged to repo root with a `*.go` mapping). Keep it fast.
- `noUncheckedIndexedAccess: true` and `exactOptionalPropertyTypes: true`, both enabled together; fix resulting errors (guards/`?.`/non-null where proven, not blanket assertions).
- `LICENSE`: fill `Copyright 2026 yoadey` (or the owner's preferred string); add SPDX to both `package.json`s.
- README: add a "Full stack locally" section (`cp .env.example .env && make install && make dev`, ports), and sync `.env.example` with CLAUDE.md's env table (add `COOKIE_ENCRYPTION_KEYS`, `PAGINATION_HMAC_KEY`, `LOG_LEVEL`, `RETENTION_*`, `GOMEMLIMIT` note).

## Risks / Trade-offs

- `noUncheckedIndexedAccess`/`exactOptionalPropertyTypes` surface many small type errors (251 across ~50 files); fix mechanically rather than blanket-asserting past them.
- Enforcing warnings may require touching unrelated files; keep those diffs mechanical.
