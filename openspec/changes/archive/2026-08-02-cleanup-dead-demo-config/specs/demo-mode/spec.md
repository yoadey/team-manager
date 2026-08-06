## ADDED Requirements

### Requirement: No unused configuration from removed implementations
Environment variables that no code path reads MUST NOT remain parsed into
application config, so `.env` documentation accurately reflects what
affects runtime behavior.

#### Scenario: A demo-mode implementation is replaced
- **WHEN** an implementation (e.g. the localStorage mock) is removed and
  replaced by another
- **THEN** any environment variable only that removed implementation
  consumed is deleted from config parsing, `.env.example`, and CLAUDE.md's
  env-var documentation in the same change
