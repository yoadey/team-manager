## Context

Teamverwaltung has a complete backend/frontend feature set (events,
members, finances, news, polls, roles/RBAC, invites, GDPR
self-service) and thorough developer docs, but nothing written for the
people the app is actually for. Two Explore passes over the repo
confirmed: no help/FAQ/onboarding UI anywhere in `frontend/src`, no
`docs/user-guide`-style directory, no docs-site tooling in any
`package.json`, and no prior OpenSpec proposal covering end-user docs.
This change closes that gap.

## Goals / Non-Goals

**Goals:**
- Every one of the app's 8 top-level routes (`home` folded into
  onboarding, `events`, `members`, `finances`, `stats`, `news`,
  `polls`, `team`) has a corresponding, findable doc chapter.
- The roles/permissions model (`none`/`read`/`write` per module,
  multi-role merge) is explained in plain language for the first time
  anywhere in the product.
- GDPR self-service (export, account deletion) has a non-technical
  companion to the existing implementer-facing doc.
- The docs are published somewhere a non-technical reader can browse
  comfortably (not just raw GitHub markdown), without adding a second
  language toolchain to a Go+TypeScript monorepo.

**Non-Goals:**
- English translation of the content (see proposal's Out of Scope).
- Any in-app UI change (help link, tooltips) — content only.
- A search/analytics/feedback layer on the docs site — a plain
  Docusaurus docs-only setup is enough for a first version.

## Decisions

- **German-only for this change.** The requester's own ask was in
  German, and the realistic audience (amateur sports clubs) is
  DACH-based. Maintaining two live drafts while the chapter structure
  is still being validated would double authoring/review cost and
  directly work against the "keep docs from going stale" goal that
  motivated this change in the first place.
- **Docusaurus over MkDocs for the static site.** Docusaurus stays
  inside the Node toolchain the monorepo already has (frontend uses
  Vite/npm); MkDocs would introduce Python as a third language
  ecosystem purely for a docs build. Docusaurus also has first-class
  i18n routing, which materially lowers the cost of the planned
  English follow-up.
- **Content lives in `docs/end-user/`, not inside `website/`.**
  Docusaurus' `docs` plugin is pointed at `../docs/end-user` via
  `path`/`routeBasePath` config rather than vendoring content into the
  site app. This keeps the markdown itself immediately readable on
  GitHub (matching the existing `docs/operations.md` /
  `docs/gdpr-data-subject-rights.md` convention) and avoids a second
  copy to keep in sync.
- **`daten-und-datenschutz.md` is a new, separate document**, not a
  section appended to `docs/gdpr-data-subject-rights.md`. That file is
  written for implementers (SQL migrations, endpoint contracts,
  "Status: implemented" notes); mixing end-user copy into it would
  force an awkward tone shift mid-document or get skipped entirely by
  the audience it's meant for. A one-line cross-reference in each
  direction keeps them discoverable from one another.
- **`docs-deploy.yml` is a separate workflow from `ci.yml`, path-filtered.**
  Every existing CI job gates app PRs; a docs-only change shouldn't run
  through — or be blocked by — that pipeline, and app PRs shouldn't pay
  for a Docusaurus build they didn't touch.
- **No new CI freshness/link-checker job for staleness.** A checklist
  line in the PR template mirrors the repo's existing enforcement style
  for a similar problem (`de`/`en` i18n catalog sync) and is
  proportionate to today's doc surface (11 short files). Revisit with
  real tooling only if the checklist proves not to be followed.

## Risks / Trade-offs

- **Translation debt**: shipping German-only risks English never
  following. Mitigation: the proposal names it as an explicit,
  separately tracked follow-up rather than a silent gap, and the
  Docusaurus i18n scaffold is left ready for an `en` locale.
- **Docusaurus adds a new npm app to maintain** (dependency updates,
  Dependabot surface). Accepted trade-off for a materially better
  non-technical reading experience than raw GitHub markdown; kept as
  small as possible (docs-only preset, no blog/versioning).
- **GitHub Pages must be enabled manually** by a repo admin outside of
  this change; until then `docs-deploy.yml` will fail at the deploy
  step (the build step still validates the site compiles). Called out
  explicitly in the proposal's Impact section so it isn't mistaken for
  a broken workflow.
