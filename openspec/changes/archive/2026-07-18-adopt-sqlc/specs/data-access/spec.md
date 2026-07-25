## ADDED Requirements

### Requirement: Static queries are type-generated
Static SQL queries MUST be defined in `.sql` files and compiled to type-safe Go by sqlc, with the generated output checked in and kept drift-free.

#### Scenario: Regenerate is clean
- **WHEN** `make generate` runs
- **THEN** sqlc regenerates the query package
- **AND** there is no diff against the checked-in generated code

### Requirement: Dynamic queries use a tested builder
Dynamic queries (variable UPDATE SET clauses, keyset predicates) MUST be built through a unit-tested builder, not string concatenation with a no-op fallback clause.

#### Scenario: Empty update set
- **WHEN** an update is requested with no changed columns
- **THEN** the builder reports an empty set explicitly
- **AND** no `SET id = id` style placeholder statement is emitted

### Requirement: Tenant scoping preserved
Every by-id query MUST remain scoped to the owning team (`AND team_id = $N`) after migration to generated queries.

#### Scenario: Cross-team lookup
- **WHEN** a record id from another team is queried within a team context
- **THEN** the generated query returns no row for the mismatched team
