## Why

`.github/workflows/ci.yml` currently runs its full ~18-job suite (frontend
lint/typecheck/test/build/E2E/Lighthouse, backend lint/test/build/
govulncheck/coverage/migration checks, a CodeQL matrix, TruffleHog, three
Trivy container scans, Helm lint, ZAP DAST) on every push and pull request,
regardless of which part of the monorepo actually changed. A docs-only or
Helm-values-only PR still pays for the full Playwright E2E run, Lighthouse
CI, both CodeQL languages, and a govulncheck pass — none of which can catch
anything that PR touches — needlessly extending queue time and burning
runner minutes.

## What Changes

- Add a `changes` job that computes which areas of the repo changed
  (`frontend/`, `backend/`, `helm/`) via `git diff` against the PR base ref
  or the push's `before` SHA — no new third-party action, keeping this
  workflow's existing everything-is-SHA-pinned supply-chain posture intact.
- Gate every frontend job (lint, typecheck, test, security audit, license
  check, build, E2E, Lighthouse, frontend Trivy image scan) on
  `needs.changes.outputs.frontend`.
- Gate every backend job (OpenAPI drift, lint, test, build, license check,
  govulncheck, coverage gate, migration rollback, migration safety lint,
  backend Trivy image scan, ZAP DAST) on `needs.changes.outputs.backend`.
- Gate `helm-lint` and the backup-image Trivy scan on
  `needs.changes.outputs.helm`.
- Split the `security-sast` CodeQL matrix so the `go` entry gates on backend
  and the `typescript` entry gates on frontend.
- Leave `security-secrets` (TruffleHog) unconditional — it's a cheap,
  whole-repo scan, and a secret can land in a file no path filter would
  flag.
- Fail open: any change to `.github/workflows/ci.yml` itself, or to a small
  set of cross-cutting files (`docker-compose.yml`, `.zap/rules.tsv`,
  `.github/trivyignore-*.txt`), or any failure to resolve a diff base,
  forces every job to run rather than trying to map it to a narrower
  subset.

## Capabilities

### Added Capabilities
- `ci-pipeline`: CI now builds/tests only the parts of the monorepo whose
  files actually changed on a given push or pull request, instead of
  unconditionally running the full suite every time.

## Impact

- `.github/workflows/ci.yml` only.
- No application code, migration, RBAC, or OpenAPI changes.
- Required-status-check names are unchanged; a skipped job reports a
  passing/neutral status to branch protection, the standard GitHub
  path-filter pattern.
