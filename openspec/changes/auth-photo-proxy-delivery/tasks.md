## 1. API contract
- [x] 1.1 `openapi.yaml`: add `200: PhotoBytes` response to `getMyPhoto`, matching `getTeamPhoto`/`getMemberPhoto`.
- [x] 1.2 `make generate` (backend) — regenerate `internal/gen/api.gen.go` (`GetMyPhoto200ImageResponse`).
- [x] 1.3 `make generate-ts` (repo root) — regenerate `frontend/src/api/types.gen.ts`.

## 2. Backend proxy-mode support for the user's own photo
- [x] 2.1 `auth.Service.GetMyPhotoBytes`: stream the user's photo bytes from object storage, mirroring `members.Service.GetMemberPhotoBytes`.
- [x] 2.2 `auth.Handler`: add `imageDeliveryProxyEnabled` field + `SetImageDeliveryProxyEnabled`; `GetMyPhoto` branches on it (bytes vs. redirect), mirroring `members.Handler.GetMemberPhoto`.
- [x] 2.3 `cmd/server/main.go`: wire `authHandler.SetImageDeliveryProxyEnabled(cfg.ImageDeliveryProxyEnabled)`.

## 3. Tests
- [x] 3.1 `auth/handler_test.go`: `mockAuthService.GetMyPhotoBytes`; new tests `TestHandler_GetMyPhoto_ProxyMode_StreamsBytes` and `TestHandler_GetMyPhoto_ProxyMode_NotFound_Returns404`, mirroring the equivalent `members` tests.
- [x] 3.2 `teams/handler_test.go`: `fakeAuthSvc.GetMyPhotoBytes` added so it still satisfies `auth.authService`.

## 4. Verification
- [x] 4.1 `cd backend && go build ./...` / `make build`
- [x] 4.2 `cd backend && make lint` (golangci-lint, gofumpt)
- [x] 4.3 `cd backend && make test-unit` (all packages green)
- [x] 4.4 `cd frontend && npm run typecheck && npm run lint` (generated `types.gen.ts` change is additive-only)
