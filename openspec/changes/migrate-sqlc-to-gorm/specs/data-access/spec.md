## MODIFIED Requirements

### Requirement: Static queries are type-safe at the Go-struct level
Repository queries MUST be issued through GORM against models whose struct tags describe the existing schema (`internal/db/migrations` remains the schema source of truth; GORM `AutoMigrate` MUST NOT be used). Hand-written SQL string concatenation with variable data MUST NOT be used.

#### Scenario: Repository query round-trips domain types
- **WHEN** a repository issues a query through GORM
- **THEN** the result is scanned into typed Go structs (including `uuid.UUID` and JSONB-backed fields) without a manual `pgx.Rows` scan loop

### Requirement: Dynamic queries use explicit column whitelisting
Dynamic queries (variable `UPDATE ... SET` clauses) MUST be built by passing an explicit `map[string]any` of only the changed columns to GORM's `.Updates(...)`, never a struct-based `.Save()`/`.Updates(struct)` call (which silently skips Go zero-valued fields) and never hand-rolled placeholder-index arithmetic.

#### Scenario: Empty update set
- **WHEN** an update is requested with no changed columns
- **THEN** the patch helper reports an empty set explicitly
- **AND** no update statement is executed

### Requirement: Tenant scoping preserved
Every by-id query against a team-scoped model MUST remain scoped to the owning team (`team_id`), enforced by a registered GORM callback that fails a query issued without the team-scoping helper rather than allowing it to run unscoped.

#### Scenario: Cross-team lookup
- **WHEN** a record id from another team is queried within a team context
- **THEN** the query returns no row for the mismatched team

#### Scenario: Missing team scope
- **WHEN** a repository issues a query against a team-scoped model without applying the team-scoping helper
- **THEN** the query fails instead of executing unscoped
