## Why

The repo has zero end-user documentation. Every existing doc
(`README.md`, `CONTRIBUTING.md`, `CLAUDE.md`, `docs/operations.md`,
`docs/gdpr-data-subject-rights.md`) is written for developers or
operators. Team members and team admins — the people who actually use
Teamverwaltung day to day — have no onboarding guide, no explanation of
the roles/permissions model (`none`/`read`/`write` per module), and no
plain-language walkthrough of the GDPR self-service features (data
export, account deletion). This is a real support-burden and
onboarding-friction gap, not a cosmetic one.

## What Changes

- New `docs/end-user/` directory: an index (`README.md`) plus 10 chapter
  files covering onboarding/invites, roles & permissions, and each of
  the app's feature areas (events, members, finances, news, polls, team
  settings, stats, privacy/GDPR).
- One cross-reference line added to the top of
  `docs/gdpr-data-subject-rights.md`, pointing to the new plain-language
  privacy chapter.
- New `website/` directory: a Docusaurus site whose `docs` plugin reads
  directly from `../docs/end-user` (no content duplication), built for
  eventual publication via GitHub Pages.
- New `.github/workflows/docs-deploy.yml`: builds and deploys the
  Docusaurus site on push to the default branch, path-filtered to
  `docs/end-user/**` and `website/**` — kept separate from `ci.yml` so
  it never blocks or slows down app PRs.
- `README.md`: new "Für Vereinsmitglieder" section linking to the new
  docs.
- `.github/pull_request_template.md`: one added checklist line asking
  authors to keep the relevant `docs/end-user/` chapter in sync with
  route/feature/permission changes.

## Capabilities

### New Capabilities
- `end-user-docs`: German-language, task-oriented documentation for
  club members and team admins, covering onboarding, every top-level
  app route, the roles/permissions model, and GDPR self-service —
  published as a static site and kept discoverable from the README.

### Modified Capabilities
<!-- none -->

## Impact

- New: `docs/end-user/*.md` (11 files), `website/**` (Docusaurus app),
  `.github/workflows/docs-deploy.yml`.
- Modified: `README.md` (new section), `.github/pull_request_template.md`
  (one checklist line), `docs/gdpr-data-subject-rights.md` (one
  cross-reference line at the top).
- No impact on application code, the OpenAPI contract, or `ci.yml`'s
  existing jobs. `docs-deploy.yml` is a new, isolated workflow.
- Manual, non-code step: a repo admin must enable GitHub Pages
  (Settings → Pages → Source: GitHub Actions) for `docs-deploy.yml` to
  actually publish; the workflow will otherwise fail on deploy until
  that's done.

## Out of Scope / Future Work

- **English translation.** The app's UI is fully bilingual (de/en), but
  this change ships German-only content, matching the primary
  DACH-club audience and avoiding two live drafts while the chapter
  structure is still being validated. Tracked as an explicit follow-up
  change once the German content has settled; the Docusaurus i18n
  scaffold is left in a state that a future `en` locale can be added to
  without restructuring.
- **In-app "Hilfe" entry point.** No navigation link to the new docs is
  added inside the product in this change — that's a distinct
  product/UX change (nav entry, bilingual strings, component, tests,
  accessibility review), not a docs change. Tracked as the natural
  next step once the content itself is validated with real users.
