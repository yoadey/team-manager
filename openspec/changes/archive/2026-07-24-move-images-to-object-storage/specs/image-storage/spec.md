## ADDED Requirements

### Requirement: Images stored in object storage
Team and user photos and logos MUST be stored in an S3-compatible object store. The database MUST hold only an object key reference, not the image bytes.

#### Scenario: Upload persists to object store
- **WHEN** a member with the required permission uploads a team photo
- **THEN** the validated, scaled image is written to the object store
- **AND** the team row stores the object key, not the bytes

#### Scenario: Object store required in production
- **WHEN** the server starts with `COOKIE_SECURE=true` and no object-store configuration
- **THEN** startup fails with a clear error

### Requirement: Membership-gated presigned delivery
Image GET endpoints MUST verify team membership before issuing access, and MUST deliver the image via a short-lived presigned URL rather than streaming bytes through the application server.

#### Scenario: Authorized retrieval
- **WHEN** a team member requests a team photo
- **THEN** the server verifies membership and responds with a redirect to a short-TTL presigned URL

#### Scenario: Non-member retrieval
- **WHEN** a non-member requests the team photo
- **THEN** the request is rejected before any presigned URL is issued

### Requirement: Upload validation preserved
Image uploads MUST continue to enforce the existing limits: at most 2 MB, only JPEG or PNG, scaled to at most 800×800.

#### Scenario: Oversized upload
- **WHEN** an upload exceeds 2 MB
- **THEN** it is rejected with a 413 error and nothing is written to the object store
