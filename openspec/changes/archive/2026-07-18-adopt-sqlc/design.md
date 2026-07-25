## Context

`make generate` currently runs only oapi-codegen (`oapi-codegen.yaml`). Repositories hand-write SQL, including dynamic UPDATE SET clauses and `fmt.Sprintf`-composed strings using fixed internal constants. Migrations under `internal/db/migrations` are the schema source of truth.

## Goals / Non-Goals

**Goals:**
- Type-safe, generated Go for static queries, reducing hand-written SQL surface and the Sprintf/SET-builder fragility.
- A tested `sqlbuilder` for the genuinely dynamic queries, removing the `SET id = $N` no-op trick.
- Keep the codegen-driven, drift-checked workflow the repo already uses.

**Non-Goals:**
- Migrating every query to sqlc (it is weak at dynamic queries; those stay pgx).
- Introducing an ORM.

## Decisions

- **sqlc pinned** and installed via `make tools`; version pinned identically across `go.mod`/Makefile/ci.yml/Dockerfile (the pin-sync CI check enforces this).
- `sqlc.yaml` uses `sql_package: pgx/v5`, `schema: internal/db/migrations`, `queries: internal/db/queries`, with type overrides for uuid / timestamptz / JSONB permissions to match existing domain types.
- Generated functions accept the pgx `DBTX` interface, compatible with existing `WithReadTx`/advisory-lock transactions.
- Migrate one small module first (e.g. `news`/`absences`) as a pilot; keep `events`/`finances` primarily on the `sqlbuilder` refactor.

## Risks / Trade-offs

- Type-override friction for uuid/JSONB must be resolved so generated rows match domain types.
- Generated code must pass golangci-lint (`sqlclosecheck`/`rowserrcheck`) or be cleanly excluded; keep it out of the coverage denominator like other `gen` packages.
- Team-scoping (`AND team_id = $N`) must not be lost when moving to generated queries.
