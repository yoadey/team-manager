## ADDED Requirements

### Requirement: DB connection pool size is configurable and documented against autoscaling
The backend's database connection pool size MUST be configurable via
environment variables rather than hardcoded, and the required Postgres
`max_connections` (or connection-pooler requirement) relative to
`autoscaling.maxReplicas` MUST be documented.

#### Scenario: Operator sizes Postgres for max replica count
- **WHEN** an operator configures `autoscaling.maxReplicas` in
  `values-prod.yaml`
- **THEN** CLAUDE.md and `docs/operations.md` document the formula
  (`maxReplicas × DB_MAX_CONNS`) an operator needs to size Postgres's
  `max_connections` or provision a pooler against
