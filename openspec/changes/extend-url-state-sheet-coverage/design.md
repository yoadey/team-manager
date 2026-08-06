## Context

`UrlState.detail` currently only models `{ kind: 'event' | 'member', id
}`. `PAGE_SHEETS` in `AppContext.tsx` lists nine sheet types as
page-level, of which only two map to a URL segment today. The other
seven silently lose their open state on refresh/share.

## Goals

- Every page-level sheet that represents a genuine navigable destination
  (settings, role management, calendar sharing) is restorable from its
  URL.
- Don't force URL-addressability onto sheets where it adds little value
  and meaningful complexity (in-progress create/edit forms with
  unsubmitted local state that can't be reconstructed from a URL alone).

## Decisions

- **Deep-link `teamSettings`, `roles`, `calendarShares`,
  `sharedCalendars` (no sub-id needed) and `roleForm` (with the target
  role id) — these are genuine destinations a user might bookmark or
  share.**
- **Do not deep-link `eventForm`/`memberForm`.** These are transient
  create/edit forms holding local, unsubmitted draft state (per the
  existing `MAX_UPLOAD_BYTES`/base64-photo-in-memory pattern noted
  elsewhere in `AppContext.tsx`) that a URL can't reconstruct — a
  refreshed or shared link would open an empty form at best, which is
  not meaningfully better than the current behavior of losing the sheet
  entirely. `PAGE_SHEETS`'s comment is updated to state this explicitly
  so the gap is a documented decision, not an oversight.
- **Extend `UrlState.detail`'s shape (or add a sibling field) rather than
  introducing a second parallel routing structure**, keeping `buildPath`/
  `parseLocation` as the single source of truth for URL⇄state
  translation.

## Risks

- **Broader `parseLocation` surface increases the chance of an invalid/
  stale URL (e.g. a `roleForm` id for a role that no longer exists).**
  Mitigate by falling back to the underlying route (today's existing
  behavior for an unrecognized path) rather than erroring.
