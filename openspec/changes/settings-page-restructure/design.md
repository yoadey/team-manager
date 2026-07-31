## Context

Account settings today live entirely in `ProfileSheet`, a single scrollable sheet opened from two entry points (desktop sidebar account row, mobile header avatar). The app's routing is state-based (`state.route`, `context/urlState.ts`), not a router library, and has no existing precedent for a full page with a persistent sub-navigation — the closest analog, `TeamPage`, is a flat card list that opens one-level-deep "page sheets" for each sub-area, not a sidebar-in-page layout. Neither MUI `Tabs` nor `Accordion` are used anywhere in the frontend.

## Goals / Non-Goals

**Goals:**
- Give every current and future personal setting an obvious category to live in.
- Preserve all existing behavior (Web Push toggle + category preferences, GDPR export/delete, language/color-scheme, legal links, logout) unchanged.
- Keep the two existing entry points visually identical; only their destination changes.

**Non-Goals:**
- Changing the main nav (rail/bottom-nav) — Settings is deliberately not added there, mirroring how the old profile entry point was never part of it either.
- Deep-linkable settings categories (bookmarkable `/settings/notifications` etc.) — out of scope, see decision below.

## Decisions

- **New route, not a sheet.** `settings` is added to `Route`/`ALL_ROUTES` with `ROUTE_MODULE.settings: null` — the same "always visible, ungated" bucket as `home`, since this is the caller's own personal data, not team-scoped.
- **Mobile category switching uses local component state, not a URL param.** `urlState.ts` has a deliberately narrow contract for what's bookmark-relevant (route + a few named list filters). A settings category isn't content anyone links to or expects Back to restore precisely — widening `UrlState`/`buildPath`/`parseLocation` for it isn't worth the added surface. Trade-off: browser Back from a mobile category-detail view leaves `/settings` entirely rather than returning to the category list; accepted, matching how `TeamPage`'s page-sheets already don't get bespoke URL segments beyond the two hardcoded `detail` kinds (`event`/`member`).
- **Logout stays outside the category list.** It's a destructive, non-configurable action, not a setting — pinned as a footer item below the categories (both desktop sidebar and mobile list) so its prominence doesn't regress from today's sheet.
- **`ProfileSheet`/`openProfile` are deleted, not kept as a fallback.** Verified via grep: the only callers are the two `AppShell.tsx` entry points and `useTeamActions.test.ts`'s own unit test. No other feature opens the `profile` sheet.
- **The absences prefetch moves from `openProfile` to `ensureRouteData('settings')`.** `useTeamActions.ts`'s `openProfile` speculatively prefetched `queryKeys.myAbsences(teamId)` into the React Query cache on "profile opened" (nothing currently reads it — kept warm for whenever something does). Since `app.go('settings')` already calls `ensureRouteData(route)`, that's the natural new home for the same trigger, keeping the speculative-cache behavior alive without inventing a new hook.
- **Settings registry is the single source of truth for categories.** `settingsCategories.ts` exports an ordered `{key, labelKey, icon, Component}[]`; `SettingsPage` renders the sidebar/list purely from this array. A new setting inside an existing theme goes into that category's panel component; a genuinely new kind of setting gets one new registry entry — never a growing flat list again.

## Risks / Trade-offs

- Mobile category-detail view has no dedicated Back-button URL segment (see decision above) — browser Back exits Settings instead of returning to the category list. Accepted as a minor, pre-existing-pattern-consistent UX cost.
- This is the first sidebar-in-page layout in the codebase — no existing component to copy exactly; built from primitives already used elsewhere (`Box`, `ButtonBase`, `NEUTRAL`/`buildTokens` tokens), following `DesktopNavRail`'s visual language for the sidebar rows.
