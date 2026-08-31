# dev-tooling Specification

## Purpose
Defines the developer-facing quality gates that keep the repository's tooling honest: frontend lint fails the build on warnings rather than letting them accumulate, the pre-commit hook checks staged Go files (not only frontend files), the README documents running the full stack (database, backend, frontend), and CI runs `openspec validate --strict` on every pull request touching `openspec/changes/`.
## Requirements
### Requirement: Lint warnings are enforced
The frontend lint gate MUST fail on warnings, so warning-level rules cannot accumulate silently.

#### Scenario: A new warning is introduced
- **WHEN** code introduces a lint warning
- **THEN** `npm run lint` exits non-zero

### Requirement: Pre-commit covers backend
The pre-commit hook MUST check staged Go files for formatting/lint, not only frontend files.

#### Scenario: Staged Go file with a format issue
- **WHEN** a Go file with a formatting problem is committed
- **THEN** the pre-commit hook flags or fixes it before the commit completes

### Requirement: Onboarding docs cover the full stack
The README MUST document how to run the full stack locally (database + backend + frontend), not only the frontend.

#### Scenario: New contributor reads the README
- **WHEN** a newcomer follows the README
- **THEN** they find the commands to run the full stack (e.g. `make install` / `make dev` / `docker compose up`) and the relevant ports

### Requirement: OpenSpec changes are validated in CI
Every pull request that adds or modifies files under `openspec/changes/`
MUST run `openspec validate --strict` in CI and fail the build on any
validation error.

#### Scenario: PR adds a change proposal missing a scenario
- **WHEN** a PR adds a requirement to a change's spec delta with no
  `#### Scenario:` block
- **THEN** the `openspec-validate` CI job fails
- **AND** the PR cannot merge without fixing the proposal

