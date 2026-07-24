## Why

`GET /teams/{teamId}/events/{eventId}/comments` is the last list endpoint still
paginated with `LIMIT/OFFSET` (`events/repository.go` `ListComments`, ordered
`created_at ASC`). Every other list (events, members, news, polls, absences,
finance transactions) uses keyset pagination with an opaque `{items, nextCursor}`
envelope. The repository's own comment already flags the cost: with OFFSET,
"every page pays a proportionally larger scan cost" as an event's comment count
grows toward the 2000-comment cap. It is also an inconsistency in the API
surface — one endpoint returns a bare array with `limit`/`offset`, all others
return the cursor envelope.

## What Changes

- Convert `listEventComments` to keyset pagination: `limit`/`cursor` params and
  a `{items, nextCursor}` response envelope, ordered `created_at ASC, id ASC`.
- Drop the now-orphaned `offset` OpenAPI parameter (used by no other operation).
- Update the in-repo frontend client to consume the envelope via the existing
  `fetchAllPages` walker, retiring the special-case `fetchAllOffsetPages` helper.

## Capabilities

### New Capabilities
- `event-comments`: comment listing gains keyset pagination and a stable
  `{items, nextCursor}` envelope, removing the OFFSET scan-cost growth.

## Impact

- Backend + spec: `openapi/openapi.yaml` (params + response), regenerated
  `internal/gen` + `frontend/src/api/*`, `events/{repository,service,handler}.go`,
  tests.
- Frontend: `serviceLayerReal.events.listComments` and the MSW comment handler.
- CI: openapi-drift, backend + frontend gates. **API-affecting** (response shape
  of one endpoint changes array → envelope; the frontend ships from this repo and
  is updated in the same change).
