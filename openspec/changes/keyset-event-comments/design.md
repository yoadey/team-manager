## Context

Event comments are ordered oldest-first (`created_at ASC`) so a reader sees the
conversation in chronological order. The list is currently `LIMIT/OFFSET`
paginated and returns a bare `EventComment[]`. `internal/pagination` already
provides the keyset cursor encode/decode (optionally HMAC-signed) used by every
other list endpoint, and the frontend already walks keyset envelopes to
completion via `fetchAllPages`.

## Goals / Non-Goals

**Goals:**
- One consistent pagination idiom across every list endpoint.
- Remove the OFFSET scan-cost growth on high-comment events.

**Non-Goals:**
- Changing comment ordering (stays `created_at ASC`).
- Exposing incremental "load more" in the UI — the client still fetches all
  pages; only the transport changes.

## Decisions

- Keyset on `(created_at, id)` ascending — `id` is the unique tiebreaker so two
  comments sharing a `created_at` timestamp still have a total order and no row
  is skipped or repeated across pages. The predicate is
  `(created_at, id) > (cursorCreatedAt, cursorId)`.
- Response becomes `{ items: EventComment[], nextCursor: string|null }`, matching
  members/news/absences/transactions.
- Encode the cursor with the shared `*pagination.Paginator` so `PAGINATION_HMAC_KEY`
  signing applies uniformly.
- Drop the `offset` component parameter from the spec — it is referenced by no
  other operation once this endpoint stops using it. `pagination.Parse` (the
  limit+offset helper) stays in the library (still exported and unit-tested); it
  simply has no production caller afterward.

## Risks / Trade-offs

- Response shape of one endpoint changes (array → envelope). The frontend is
  in-repo and updated in the same change; the MSW mock and the generated client
  move together, and `serviceContract`/handler tests guard the round trip.
- A stale client pinned to the old array shape would break — acceptable because
  the SPA ships from this repo and there is no external API consumer contract.
