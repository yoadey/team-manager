## MODIFIED Requirements

### Requirement: Image delivery can be proxied through the backend
Team/user photo and logo delivery MUST support a configurable proxy mode where the backend streams the object store's bytes directly, as an alternative to redirecting the client to a presigned object-store URL, for deployments where the object store is not reachable from the browser. This applies to the signed-in user's own photo endpoint (`GET /auth/me/photo`) as well as team member photos, team photos, and team logos.

#### Scenario: Proxy mode enabled
- **WHEN** the backend is configured for proxy image delivery and a client requests a team photo
- **THEN** the backend streams the image bytes directly in the response, with no redirect to the object store

#### Scenario: Proxy mode enabled for the signed-in user's own photo
- **WHEN** the backend is configured for proxy image delivery and the signed-in user requests their own photo (`GET /auth/me/photo`)
- **THEN** the backend streams the image bytes directly in the response, with no redirect to the object store

#### Scenario: Redirect mode (default) unchanged
- **WHEN** the backend is configured for redirect image delivery (the existing default)
- **THEN** the client receives a 302 redirect to a short-lived presigned object-store URL, as before

#### Scenario: Access control preserved in proxy mode
- **WHEN** a client without team membership requests a team photo while the backend is in proxy mode
- **THEN** the request is rejected before any object-store bytes are streamed, the same as redirect mode's presign gating
