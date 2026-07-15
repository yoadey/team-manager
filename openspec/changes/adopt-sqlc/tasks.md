## 1. Tooling
- [ ] 1.1 Add `sqlc` (pinned) to `make tools`; pin the identical version in `Makefile`, `.github/workflows/ci.yml`, and Dockerfile if tools are installed there
- [ ] 1.2 Extend `make generate` to run `sqlc generate` alongside oapi-codegen

## 2. Configuration
- [ ] 2.1 Add `backend/sqlc.yaml` (`engine: postgresql`, `sql_package: pgx/v5`, schema=migrations, queries=`internal/db/queries`, out=`internal/db/gen`)
- [ ] 2.2 Add type overrides for uuid, timestamptz, and JSONB permissions to match existing domain types

## 3. Pilot module
- [ ] 3.1 Add `internal/db/queries/<module>.sql` for a small module (e.g. `news`/`absences`) with `:one`/`:many`/`:exec` annotations, each carrying `AND team_id = $N`
- [ ] 3.2 Switch that repository to the generated functions inside existing transactions; keep error wrapping and `pgx.ErrNoRows` → domain-sentinel translation

## 4. Dynamic-query builder
- [ ] 4.1 Add `internal/db/sqlbuilder` with an explicit empty-set signal; WHERE args passed separately from SET args
- [ ] 4.2 Replace the hand-rolled SET builders in `events`/`finances`; keep the existing "DoesNotCorruptSQL" regression test green

## 5. Roll out
- [ ] 5.1 Migrate remaining mostly-static modules (`roles`, parts of `members`/`polls`); leave heavily dynamic queries on pgx

## 6. Verification
- [ ] 6.1 `make tools && make generate` → no diff in generated code (drift check green)
- [ ] 6.2 `make lint` green (generated code passes or is cleanly excluded)
- [ ] 6.3 `make test` green (migrated-module integration tests + `sqlbuilder` unit tests)
- [ ] 6.4 Coverage gate and pin-sync check green; `make vuln` green
