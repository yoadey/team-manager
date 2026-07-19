## ADDED Requirements

### Requirement: Single role mapper
The mapping from a role row to its API representation MUST be defined in exactly one place and reused by all modules.

#### Scenario: New permission module added
- **WHEN** the permission model gains a module
- **THEN** only one mapper needs updating for all endpoints to return it

### Requirement: Tenant-scoped snapshot read
The penalty snapshot read during assignment creation MUST be scoped to the owning team in the query itself.

#### Scenario: Snapshot read
- **WHEN** an assignment is created and the penalty snapshot is read
- **THEN** the query filters by both penalty id and team id

### Requirement: No placeholder domain in error responses
Problem+json `type` URIs MUST NOT default to a placeholder/example domain; when unconfigured they MUST use relative paths.

#### Scenario: Error response without configuration
- **WHEN** an error response is produced without `ERROR_TYPE_BASE_URI` set
- **THEN** the `type` is a relative path, not an example domain
