## Context

`urlState.ts`'s `buildPath`/`parseLocation` are the single source of
truth for the state↔URL mapping (see the file's header comment).
`AppState.route` (the active top-level tab) and `AppState.sheet` (the
currently open sheet, if any) are independent pieces of state — opening
an event/member detail sheet does not itself change `route`. Most call
sites open the detail sheet from its own route (`EventsPage`,
`MembersPage`), where `route` already happens to match, masking the bug.
Two call sites don't: `components/cards.tsx`'s `EventCard` (rendered on
Home) and `NotificationsSheet.tsx` (reachable from any route via the
always-visible bell icon in `AppShell`).

## Goals

- A detail sheet's presence in the URL must not depend on which route it
  was opened from.
- Preserve today's push-vs-replace intent (opening a sheet is a Back-able
  navigation; a filter tweak or closing a sheet is not) instead of
  papering over the bug with a raw `pushState` on every change.

## Decisions

- **Derive the detail path segment from `detail.kind` rather than
  `route`.** `UrlState.detail` already fully identifies the target
  (`event` → `/events/<id>`, `member` → `/members/<id>`); re-deriving the
  same information from `route` was redundant and is exactly what broke.
  This is a smaller, more targeted change than switching `state.route`
  itself whenever a detail sheet opens, which would also affect nav
  highlighting and `ensureRouteData`/RBAC gating for routes the caller
  never navigated to.
- **Refine `isNavigation` to key off the detail-open/close transition
  first, route comparison second.** The previous `prev.route !==
  next.route || (!prev.detailId && !!next.detail)` conflated "opening a
  detail" with "the route component of the URL changed", which are the
  same thing only when the detail is opened from its own route. Without
  this refinement, closing a detail sheet that was opened from a
  different route (e.g. Home → event → close) would itself look like a
  route change and push a *second* entry, so a single Back press would
  reopen the very sheet the user just closed instead of leaving the
  page. Splitting the two into "opening always pushes" / "closing always
  replaces" / "a route switch with no detail on either side pushes"
  fixes this symmetrically for both same-route and cross-route opens.

## Risks

- None beyond the usual URL-format risk already covered by
  `urlState.test.ts`'s round-trip tests; `parseLocation` is unchanged, so
  no new parse cases are introduced.
