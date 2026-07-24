## ADDED Requirements

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
