## Context

Structured request logs are already clean (method/path/status/ids only). The leak is specifically the audit `attrs` email on login success/failure. bcrypt's 72-byte limit is a library property; there is no pre-check. `CSRFOriginCheck` is defense-in-depth atop `SameSite=Lax`.

## Goals / Non-Goals

**Goals:**
- No plaintext email retained in the audit log; keep an analyzable correlation token.
- Reject over-length passwords deterministically instead of silent truncation.
- Strengthen the CSRF fallback without breaking legitimate non-browser clients.

**Non-Goals:**
- Replacing the audit trail or its retention policy.
- Making CSRF a hard block on all header-less clients (SameSite remains primary).

## Decisions

- **Audit email → `email_hash`**: `hmac.New(sha256, auditKey)` over the lowercased email. Reuse `PAGINATION_HMAC_KEY`/an existing 32-byte key or add `AUDIT_HMAC_KEY`; document it. Login success/failure records the hash, not the address. Add an erase-time scrub of that user's rows only if `actor_id` is present (failed-login rows for a mistyped address have no actor and are covered by retention).
- **Password length**: reject `len(password) > 72` bytes with a validation problem (400/422) in the login and set-password paths, before bcrypt.
- **CSRF**: also consult `Sec-Fetch-Site`; for a state-changing, cookie-authenticated request with neither `Origin` nor `Sec-Fetch-Site`, reject. Keep allowing a matching `Origin`/same-origin `Sec-Fetch-Site`.

## Risks / Trade-offs

- Hashing email in audit removes the ability to read the address directly; that is the point — correlation via hash remains for brute-force analysis.
- The CSRF tightening could affect exotic non-browser clients that send no fetch-metadata and no Origin; document it and keep it scoped to mutating methods.
- Introducing a new key requires config plumbing and a `CLAUDE.md` entry; prefer reusing an existing key to avoid key sprawl.
