## MODIFIED Requirements

### Requirement: Images stored in object storage
Team and user photos and logos MUST be stored in an S3-compatible object
store. The database MUST hold only an object key reference, not the image
bytes — there is no legacy BYTEA column and no backfill path, since no
deployment predates object storage.

#### Scenario: Upload persists to object store
- **WHEN** a member with the required permission uploads a team photo
- **THEN** the validated, scaled image is written to the object store
- **AND** the team row stores the object key, not the bytes

#### Scenario: Object store required in production
- **WHEN** the server starts with `COOKIE_SECURE=true` and no object-store
  configuration
- **THEN** startup fails with a clear error

#### Scenario: has_photo reflects only the object key
- **WHEN** any query computes whether a user or team has a photo/logo
- **THEN** it evaluates `*_object_key IS NOT NULL` only, with no legacy
  `*_data IS NOT NULL` fallback
