## 1. Scaffolding
- [x] 1.1 Create `frontend/src/features/settings/settingsCategories.ts` — ordered category registry `{key, labelKey, icon, Component}[]` with a doc comment on where to add new entries.
- [x] 1.2 Create `frontend/src/features/settings/SettingsPage.tsx` — desktop sidebar+pane / mobile list+detail (local `useState`), pinned Logout footer.
- [x] 1.3 Create `frontend/src/features/settings/index.ts` barrel exporting `SettingsPage`.

## 2. Extract panels from `ProfileSheet`
- [x] 2.1 `components/ProfilePanel.tsx` — avatar/photo upload, name, email.
- [x] 2.2 `components/AppearancePanel.tsx` — color-scheme picker, language picker.
- [x] 2.3 `components/NotificationsPanel.tsx` — Web Push toggle + per-team category preferences (move `PushCategoryPreferencesPanel` here).
- [x] 2.4 `components/PrivacyPanel.tsx` — GDPR export, account deletion (email-confirm flow).
- [x] 2.5 `components/LegalPanel.tsx` — Impressum/Datenschutz links.
- [x] 2.6 Remove `ProfileSheet` + `PushCategoryPreferencesPanel` from `NavSheets.tsx`.

## 3. Routing wiring
- [x] 3.1 `urlState.ts`: add `'settings'` to `Route`/`ALL_ROUTES`; `ROUTE_MODULE.settings: null`; update the doc comment.
- [x] 3.2 `AppContext.tsx`: `ensureRouteData` gains a `route === 'settings'` branch that prefetches `queryKeys.myAbsences(teamId)` (moved from `openProfile`); remove `openProfile` (type, all 3 wiring sites, `useTeamActions.ts`).
- [x] 3.3 `features/team/index.ts` / `sheets/index.tsx`: remove `profile` from `teamSheetMap` and `sheetMeta`'s `titles`; remove `'profile'` from `SheetType`.
- [x] 3.4 `pages/index.tsx`: lazy-import `SettingsPage`, add `case 'settings'` to `RouteScreen`.
- [x] 3.5 `layouts/pageMeta.ts`: add `settings` entry to `defs`.

## 4. Entry points
- [x] 4.1 `AppShell.tsx`: both `app.openProfile` call sites → `app.go('settings')`.
- [x] 4.2 `NavSheets.tsx`'s `MoreSheet`: add a `settings` row (always visible, ungated).

## 5. i18n
- [x] 5.1 `de.ts` + `en.ts`: add `nav.settings`, `page.settingsSubtitle`, `settings.category.{profile,appearance,notifications,privacy,legal}`; remove `sheet.profile`.

## 6. Tests
- [x] 6.1 Port `NavSheets.test.tsx`'s `ProfileSheet` / `ProfileSheet — color scheme` / `ProfileSheet — push notification preferences` blocks into the corresponding new panel test files.
- [x] 6.2 `useTeamActions.test.ts`: remove the `openProfile` test.
- [x] 6.3 `AppContext.test.tsx`: add a test that `go('settings')` prefetches `myAbsences`; update the `sheetErrorBoundaryKey` test's use of `'profile'` as a sheet-type example to a still-existing type.
- [x] 6.4 `RouteScreen.test.tsx`: mock `@/features/settings`, add a `settings` case, and a regression test proving it renders even when `can()` returns false (ungated).
- [x] 6.5 New `SettingsPage.test.tsx`: category list renders all 5 + Logout; desktop shows content pane on selection; mobile (`useCompact` mocked true) shows list-only then detail-then-back.
- [x] 6.6 New per-panel tests as needed to cover anything not already ported in 6.1 (e.g. `NotificationsPanel` push behavior).

## 7. Verification
- [x] 7.1 `openspec validate settings-page-restructure --strict` passes.
- [x] 7.2 `cd frontend && npm run lint` — 0 errors.
- [x] 7.3 `cd frontend && npm run typecheck` — 0 errors.
- [x] 7.4 `cd frontend && npm test` — all suites green.
- [x] 7.5 `cd frontend && npm run build` (+ `check:bundle` budget).
- [x] 7.6 Manual check via `npm run dev`: both entry points reach `/settings`; all 5 categories functional (Web Push especially); Logout reachable; mobile "Mehr" shows the Settings row.
