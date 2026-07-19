## 1. Spec
- [x] 1.1 `openapi.yaml`: `listEventComments` params `limit`/`offset` → `limit`/`cursor`; response array → `{items, nextCursor}` envelope (nextCursor nullable)
- [x] 1.2 Remove the orphaned `offset` component parameter (referenced nowhere else)
- [x] 1.3 `make generate` + repo-root `make generate-ts`; commit generated output (`api.gen.go`, `types.gen.ts`, `zod.gen.ts`)

## 2. Backend
- [x] 2.1 `Repository.ListComments(eventID, teamID string, limit int, cur *CommentCursor)`: keyset on `(created_at, id)` ASC, no OFFSET
- [x] 2.2 `Service.ListComments(..., cursor string) ([]gen.EventComment, *string, error)`: decode cursor, over-fetch by one, encode next `CommentCursor` via the shared `*pagination.Paginator`
- [x] 2.3 `Handler.ListEventComments`: parse `limit`/`cursor`, return the envelope; map `ErrInvalidCursor` → 400
- [x] 2.4 Events `NewService` already receives the shared `*pagination.Paginator` (ListEvents used it); no wiring change needed

## 3. Frontend
- [x] 3.1 `serviceLayerReal.events.listComments` walks the `{items, nextCursor}` envelope via `fetchAllPages`; retired the now-unused `fetchAllOffsetPages` helper
- [x] 3.2 MSW handler returns `{items, nextCursor: null}` for the comments GET

## 4. Verification
- [x] 4.1 `make generate`/`generate-ts` produce only the intended diff (openapi.yaml, api.gen.go, types.gen.ts, zod.gen.ts; rbac table unchanged — same method/path/module); openapi-drift green
- [x] 4.2 `go test ./... -short` + `golangci-lint run ./...` (0 issues) green; frontend `typecheck` + full test (1137) + `build` + `check:bundle` green
- [x] 4.3 Pagination covered by tests: repo keyset test (paging whole thread, no repeats, chronological, team-scoped); service next-cursor/last-page/cursor-decode; handler envelope + invalid-cursor→400; MSW round-trip walks all comments; serviceLayerReal test walks keyset pages forwarding the cursor
