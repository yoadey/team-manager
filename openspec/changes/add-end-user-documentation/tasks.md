## 1. Scaffolding
- [x] 1.1 Create `docs/end-user/README.md` (index, "für wen ist das", links to every chapter)

## 2. Onboarding & roles chapters
- [x] 2.1 `docs/end-user/erste-schritte.md` — invite link/code, first login, "kein Team" state, language/dark-mode switch, short home-dashboard overview
- [x] 2.2 `docs/end-user/rollen-und-rechte.md` — modules, none/read/write levels as a table, multi-role merge

## 3. Feature chapters
- [x] 3.1 `docs/end-user/termine.md` — list/calendar/absences, attendance, nominations, comments, iCal/ICS export
- [x] 3.2 `docs/end-user/mitglieder.md` — member list/profile, multi-role assignment (links to 2.2)
- [x] 3.3 `docs/end-user/finanzen.md` — Umsätze / Strafen (+ catalog) / Beiträge tabs, member vs. admin view
- [x] 3.4 `docs/end-user/news.md` — reading/posting news
- [x] 3.5 `docs/end-user/umfragen.md` — creating a poll, voting, results
- [x] 3.6 `docs/end-user/team-einstellungen.md` — team settings, inviting members (admin side), managing roles (links to 2.2), switching teams
- [x] 3.7 `docs/end-user/statistik.md` — attendance statistics, period filter

## 4. Privacy chapter
- [x] 4.1 `docs/end-user/daten-und-datenschutz.md` — data export, account deletion (re-type-email confirmation), what deletion actually means
- [x] 4.2 Add cross-reference line at the top of `docs/gdpr-data-subject-rights.md` pointing to 4.1

## 5. Docusaurus site
- [x] 5.1 Scaffold `website/` (package.json, docusaurus.config.ts, minimal theme config), `docs` plugin `path: '../docs/end-user'`, `routeBasePath: '/'`, docs-only preset (no blog)
- [x] 5.2 Confirm `npm --prefix website install && npm --prefix website run build` succeeds and renders all chapters

## 6. Publishing
- [x] 6.1 Add `.github/workflows/docs-deploy.yml` (build `website/`, deploy to GitHub Pages), triggered on push to default branch, path-filtered to `docs/end-user/**` and `website/**`, independent from `ci.yml`

## 7. Discoverability
- [x] 7.1 Add "Für Vereinsmitglieder" section to root `README.md` after "Funktionsumfang", linking to `docs/end-user/README.md`
- [x] 7.2 Add PR-template checklist line: keep the relevant `docs/end-user/` chapter in sync with route/feature/permission changes

## 8. Verification
- [x] 8.1 `openspec validate add-end-user-documentation --strict` passes
- [x] 8.2 All internal links within `docs/end-user/**` resolve to existing files
- [x] 8.3 `npm --prefix website run build` succeeds
- [x] 8.4 Existing `ci.yml` jobs remain unaffected (no shared triggers/paths with `docs-deploy.yml`)
