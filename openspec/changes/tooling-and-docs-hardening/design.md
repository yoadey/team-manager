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
- `noUncheckedIndexedAccess: true`; fix resulting errors (guards/`?.`/non-null where proven). Evaluate `exactOptionalPropertyTypes` separately — enable only if the fix cost is bounded.
- `LICENSE`: fill `Copyright 2026 yoadey` (or the owner's preferred string); add SPDX to both `package.json`s.
- README: add a "Full stack locally" section (`cp .env.example .env && make install && make dev`, ports), and sync `.env.example` with CLAUDE.md's env table (add `COOKIE_ENCRYPTION_KEYS`, `PAGINATION_HMAC_KEY`, `LOG_LEVEL`, `RETENTION_*`, `GOMEMLIMIT` note).

## Risks / Trade-offs

- `noUncheckedIndexedAccess` (and `exactOptionalPropertyTypes`, enabled alongside it) surfaced 251 errors across ~50 files when tried together; the fix cost turned out bounded (mostly two repeating patterns -- guard/`??`/restructured lookups, and an `opt()` omit-key helper for optional wire/DTO fields) and was completed in this change rather than split out.
- Enforcing warnings required touching several component files (genuine complexity/max-params refactors, not threshold changes); those diffs stayed mechanical/behavior-preserving, with the existing `*.test.tsx` suites as the check.
