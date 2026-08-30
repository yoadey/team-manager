# deployment-setup Specification

## Purpose
Defines how a fresh deployment bootstraps its database schema: a single initial goose migration takes an empty database to the full current schema (rather than replaying an accumulated history that never existed), applies and rolls back cleanly, and carries no dead legacy `photo_data`/`photo_mime`/`logo_data`/`logo_mime` BYTEA columns or code paths, since object-store-backed image storage is the only supported mechanism.
## Requirements
### Requirement: A fresh install applies one initial-setup migration
A brand-new deployment MUST reach the full current schema by applying a
single initial goose migration, not by replaying an accumulated history of
incremental migrations from a prior deployment that never existed.

#### Scenario: Fresh database bootstrap
- **WHEN** `goose up` (or the backend's own startup migration runner) runs
  against an empty database
- **THEN** exactly one migration file applies and leaves the database at the
  full current schema

#### Scenario: Rollback and re-apply
- **WHEN** the initial-setup migration is applied, then rolled back
  (`down-to 0`), then re-applied
- **THEN** the database ends at the same schema with no errors

### Requirement: No dead legacy image-storage columns
The schema MUST NOT carry `photo_data`/`photo_mime`/`logo_data`/`logo_mime`
BYTEA columns or any code path that reads them, since no deployment predates
object-store-backed image storage.

#### Scenario: Fresh install's users/teams tables
- **WHEN** the initial-setup migration completes
- **THEN** `users` and `teams` have `photo_object_key` (and `teams` has
  `logo_object_key`) but no `*_data`/`*_mime` columns

