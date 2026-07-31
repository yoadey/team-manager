## Context

`.github/workflows/ci.yml` has no path filtering today: every job runs on
every push/PR. The file already has a `concurrency` block that cancels a
superseded run specifically because it's expensive (~18 jobs, including
Playwright E2E, Lighthouse, a two-language CodeQL matrix, and a ZAP DAST
scan) — but a run that *does* complete still pays full price even when, say,
only `helm/team-manager/values-prod.yaml` changed.

## Goals

- Skip a job's whole dependency chain when its area of the repo
  (frontend/backend/helm) isn't touched by the diff.
- Never skip a check for a change that could plausibly affect it — treat
  ambiguity as "run everything" rather than trying to be precise.
- No new third-party GitHub Action.

## Decisions

- **Hand-rolled `git diff`, not `dorny/paths-filter` or similar.** Every
  external action in this workflow is pinned to an immutable commit SHA
  specifically to defend against tag-hijacking (see the `trivy-action`/Helm
  install steps' own comments on this). Introducing a new action to save a
  dozen lines of shell isn't worth that additional supply-chain surface,
  especially since `backend-migration-safety` already established the
  pattern this reuses: `fetch-depth: 0`, `git fetch origin <base>`, and a
  base-ref diff (three-dot for PRs; two-dot against `github.event.before`
  for pushes, since there's no merge-base ambiguity on a linear push).
- **Fail open.** Every filter output defaults to `true` ("run this area")
  whenever the diff can't be resolved — no `before` SHA (first push to the
  branch), a fetch failure, or any other error. This is the opposite
  default from `backend-migration-safety`, which fails loudly on a broken
  diff because a false negative there hides a real DDL hazard; here a false
  negative only wastes CI minutes, while a false positive (skipping a job
  that should have run) risks merging something unbuilt or untested. The
  asymmetry between the two failure costs is what picks the default.
- **A short, static list of cross-cutting files short-circuits straight to
  "run everything"** — `.github/workflows/ci.yml` itself, `docker-compose.yml`,
  `.zap/rules.tsv`, `.github/trivyignore-*.txt` — rather than trying to map
  each one to exactly the jobs it affects. They change rarely, so being
  conservative there costs almost nothing, and the real mapping is exactly
  the kind of multi-file coupling this file already flags as error-prone
  elsewhere (e.g. `backend-lint`'s tool-pin-sync check cross-references
  `docker-compose.yml`, `helm/team-manager/values.yaml`, `ci.yml`, and
  `scheduled-security-scan.yml` against each other).
- **`security-secrets` (TruffleHog) stays unconditional.** It's a whole-repo
  scan already priced for that (10 min timeout), and a secret can land in a
  file type no path filter would think to flag (a stray `.env`, a config
  file at repo root).
- **Gate every conditional job explicitly**, rather than relying solely on
  GitHub's default skip-propagation through `needs` (a job whose `needs`
  entry was skipped is itself skipped by default). Downstream jobs like
  `frontend-build` would already skip transitively once `frontend-lint`/
  `frontend-typecheck`/`frontend-test` skip, but writing the same `if:
  needs.changes.outputs.frontend == 'true'` on `frontend-build` too makes
  the gating self-documenting and doesn't silently break if a future edit
  reorders or drops one of those `needs` entries.
- **CodeQL's `language: [go, typescript]` matrix becomes two fixed jobs**,
  not a per-entry gate — `go` only needs to run when backend changed,
  `typescript` only when frontend changed, but a job-level `if:` can only
  read `github`/`needs`/`vars`/`inputs`, not `matrix` (that's step-level
  only). The first push of this change tried `if: matrix.language == ... `
  at the job level and GitHub Actions rejected the entire workflow file
  outright (the run showed 0 scheduled jobs) rather than just failing that
  one job — `actionlint` catches this class of error locally, which plain
  YAML validation does not, since the file is syntactically valid YAML the
  whole time.

## Risks

- The static cross-cutting file list could miss a real dependency added
  later (e.g. a new shared config file introduced by an unrelated change).
  Mitigated by the fail-open default and by keeping the list short and
  reviewed alongside any future file that plays a similar cross-referenced
  role.
- The new `changes` job becomes a dependency nearly every other job now
  waits on. Kept to a single checkout + one shell step with a 5-minute
  timeout to bound the added latency.
