## Why

User-reported bug (production deployment with proxy image delivery enabled): "Bei mir wird nichts angezeigt, er versucht da auf die S3 URL zurückzufallen, obwohl der Proxy aktiv ist" — the user's own photo doesn't render, because the client still ends up requesting the (unreachable, from that client) S3 presigned URL even though the deployment has proxy image delivery turned on.

`image-delivery-proxy`'s "Proxy mode enabled" requirement already claims coverage for "Team/**user** photo and logo delivery", and `teams.Handler` (`GetTeamPhoto`/`GetTeamLogo`) and `members.Handler` (`GetMemberPhoto`) do branch on `config.Config.ImageDeliveryProxyEnabled` via `SetImageDeliveryProxyEnabled`, streaming bytes directly in proxy mode. `auth.Handler.GetMyPhoto` (`GET /auth/me/photo` — the signed-in user's own photo, used by the frontend's top-right avatar and profile sheet) was missed: it unconditionally returned a 302 redirect to a presigned object-store URL, with no `imageDeliveryProxyEnabled` field, no `SetImageDeliveryProxyEnabled` method, and never wired to `cfg.ImageDeliveryProxyEnabled` in `cmd/server/main.go`. In deployments where proxy mode was enabled specifically because the object store isn't reachable from the browser, this made the current user's own photo permanently unloadable while every other member's photo (served via `GetMemberPhoto`) worked fine.

## What Changes

- `auth.Handler.GetMyPhoto` gains the same redirect-vs-proxy branch as `members.Handler.GetMemberPhoto`: streams image bytes directly when `imageDeliveryProxyEnabled` is set, otherwise redirects (unchanged default).
- `auth.Service` gains `GetMyPhotoBytes`, mirroring `members.Service.GetMemberPhotoBytes`.
- `openapi.yaml`'s `getMyPhoto` operation gains the `200` (`PhotoBytes`) response alongside the existing `302`/`404`, matching `getTeamPhoto`/`getMemberPhoto`; regenerated Go (`internal/gen`) and TS (`frontend/src/api/types.gen.ts`) clients.
- `cmd/server/main.go` wires `authHandler.SetImageDeliveryProxyEnabled(cfg.ImageDeliveryProxyEnabled)`, matching the existing `teamsHandler`/`membersHandler` wiring.

## Capabilities

### Modified Capabilities
- `image-delivery-proxy`: the user's own photo endpoint now actually honors proxy mode, closing the gap between the existing "Team/user photo" wording and what was implemented.

## Impact

- Backend: `internal/auth/handler.go`, `internal/auth/service.go`, `cmd/server/main.go`, `openapi/openapi.yaml`, `internal/gen/api.gen.go` (generated).
- Frontend: `frontend/src/api/types.gen.ts` (generated only — the client never calls this endpoint through the typed client; the photo URL is a plain `<img>`/CSS `background-image` src built in `frontend/src/api/map.ts`, so no frontend logic changes).
- CI: backend lint/test/build, `backend-openapi-drift` (must stay green after regenerating both clients), frontend lint/typecheck/test/build.
