## Context

The finance overview is one monolithic response; other list endpoints (events/members/news/polls/absences) use keyset pagination with optional HMAC-signed cursors (`internal/pagination`). Toggle endpoints exist at `openapi.yaml` `togglePenaltyPaid`/`toggleContribution`. Transactions store a server-stamped date with no client field.

## Goals / Non-Goals

**Goals:**
- No hard cap on which finance rows a client can reach.
- Retry-safe paid/contribution state changes.
- Consistency with the repo's existing keyset pagination idiom.

**Non-Goals:**
- Reworking the overview aggregates (kept; they stay a single summary call).
- A full finance-reporting redesign.

## Decisions

- Add `GET /teams/{teamId}/finances/transactions?cursor=…` (and penalties/assignments if needed) using the existing `internal/pagination` keyset + cursor helpers; the overview keeps aggregates + a bounded first page.
- Replace toggles with idempotent `PUT .../penalty-assignments/{id}/paid` `{"paid": bool}` and `PUT .../contributions/{id}` `{"paid": bool}` (or accept an `Idempotency-Key` on the existing POST — but explicit-value PUT is simpler and preferred). Keep the old routes deprecated for one release if backward compat matters, or replace outright since the frontend is in-repo.
- Optional tx `date`: add an optional field to create/update; default to server date when omitted.
- Regenerate all clients (`make generate` + `make generate-ts`) and update the frontend finance vertical.

## Risks / Trade-offs

- Largest change of the set (spec + both codegens + frontend + tests); do it after the smaller, lower-risk changes.
- Replacing the toggle routes is a breaking API change; since the frontend ships from this repo, update both sides together in the same change.
- Pagination adds cursor plumbing to the finance repo; reuse `internal/pagination` to match existing behavior (unsigned vs HMAC-signed cursors per config).
