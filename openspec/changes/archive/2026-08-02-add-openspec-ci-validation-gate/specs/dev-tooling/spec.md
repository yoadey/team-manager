## ADDED Requirements

### Requirement: OpenSpec changes are validated in CI
Every pull request that adds or modifies files under `openspec/changes/`
MUST run `openspec validate --strict` in CI and fail the build on any
validation error.

#### Scenario: PR adds a change proposal missing a scenario
- **WHEN** a PR adds a requirement to a change's spec delta with no
  `#### Scenario:` block
- **THEN** the `openspec-validate` CI job fails
- **AND** the PR cannot merge without fixing the proposal
