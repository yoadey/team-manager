## Why

"Mein Konto" (`ProfileSheet` in `frontend/src/features/team/components/NavSheets.tsx:181-576`) is a single flat sheet that already holds ~9-11 unrelated entries: avatar/photo upload, name/email, color-scheme picker, language picker, Web Push toggle, per-team push-category preferences, logout, GDPR export/account-deletion, and legal links. There is no grouping — every future personal setting (more push categories, more account preferences, etc.) has no obvious place to go except appended to the bottom of the same list, which is already unwieldy.

## What Changes

- Replace `ProfileSheet` with a dedicated, ungated `/settings` route (`SettingsPage`) that has its own persistent category sidebar (desktop: sidebar + content pane; mobile: category list that opens a per-category detail view with a back affordance).
- Carve the existing flat content into five categories: Profil, Darstellung & Sprache, Benachrichtigungen (includes the Web Push toggle and per-team push-category preferences, unchanged), Datenschutz, Rechtliches. Logout stays a pinned footer action outside the categories.
- Re-wire the two existing entry points (desktop sidebar account row, mobile header avatar button) to navigate to the new route instead of opening the old sheet; both keep their current position and appearance. Add a discoverability entry to the mobile "Mehr" overflow sheet. No new item is added to the main nav (rail/bottom-nav) — Settings stays outside that list, exactly like the old profile entry point did.
- Remove `ProfileSheet`, `PushCategoryPreferencesPanel`, `openProfile`, and the `profile` sheet type entirely (verified: no callers beyond the two entry points and their own unit test).

## Capabilities

### New Capabilities
- `settings-page`: a dedicated, category-navigated settings page replacing the flat account sheet.

## Impact

- Frontend only, no OpenAPI/backend change (`make generate`/`make generate-ts` not implicated).
- New: `frontend/src/features/settings/` (`SettingsPage.tsx`, `settingsCategories.ts`, `components/{Profile,Appearance,Notifications,Privacy,Legal}Panel.tsx`, `index.ts`, tests).
- Changed: `frontend/src/context/urlState.ts` (`Route`, `ALL_ROUTES`, `ROUTE_MODULE`), `frontend/src/context/AppContext.tsx` (`ensureRouteData`, `SheetType`, `AppContextValue`, removal of `openProfile` wiring), `frontend/src/features/team/hooks/useTeamActions.ts` (remove `openProfile`), `frontend/src/features/team/components/NavSheets.tsx` (remove `ProfileSheet`/`PushCategoryPreferencesPanel`, add `settings` to `MoreSheet`), `frontend/src/features/team/index.ts`, `frontend/src/sheets/index.tsx` (`sheetMeta`), `frontend/src/layouts/AppShell.tsx` (both entry points), `frontend/src/pages/index.tsx` (`RouteScreen`), `frontend/src/layouts/pageMeta.ts`, `frontend/src/i18n/de.ts` + `en.ts`.
- Tests: `NavSheets.test.tsx`'s `ProfileSheet` blocks migrate into new panel test files; `useTeamActions.test.ts`'s `openProfile` test is removed; `AppContext.test.tsx` gains a test that the absences-prefetch moved onto `go('settings')`; `RouteScreen.test.tsx` gains a `settings` case (asserted ungated); new `SettingsPage.test.tsx`.
