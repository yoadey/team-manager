## ADDED Requirements

### Requirement: CI skips jobs for unchanged areas of the monorepo
CI MUST NOT run backend jobs when no backend files changed, and MUST NOT
run frontend jobs when no frontend files changed, on a given push or pull
request.

#### Scenario: Frontend-only change
- **WHEN** a pull request only modifies files under `frontend/`
- **THEN** backend jobs (OpenAPI drift, lint, test, build, license check,
  govulncheck, coverage gate, migration rollback, migration safety lint,
  backend container scan, DAST) report as skipped rather than running

#### Scenario: Backend-only change
- **WHEN** a pull request only modifies files under `backend/`
- **THEN** frontend jobs (lint, typecheck, test, security audit, license
  check, build, E2E, Lighthouse, frontend container scan) report as
  skipped rather than running

### Requirement: Cross-cutting changes always run the full pipeline
A change to a file whose effect isn't confined to `frontend/`, `backend/`,
or `helm/` MUST run every job, rather than being mapped to a narrower
subset.

#### Scenario: The CI workflow itself changes
- **WHEN** a pull request modifies `.github/workflows/ci.yml`
- **THEN** every job in the workflow runs regardless of whether
  frontend/backend/helm files also changed

#### Scenario: A shared cross-referenced file changes
- **WHEN** a pull request modifies `docker-compose.yml`, `.zap/rules.tsv`,
  or a `.github/trivyignore-*.txt` file
- **THEN** every job in the workflow runs

### Requirement: Path-diff failures never suppress a check
If the workflow cannot determine which paths changed, it MUST run every
job rather than skip any.

#### Scenario: No usable base ref
- **WHEN** the push event's `before` SHA is unresolvable (e.g. the first
  push to a protected branch) or the diff computation otherwise fails
- **THEN** every job's path filter defaults to true
