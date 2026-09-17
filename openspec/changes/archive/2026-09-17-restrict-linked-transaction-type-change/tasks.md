## 1. Backend: reject type changes on a linked transaction
- [x] 1.1 `internal/finances/service.go`: add
      `ErrCannotChangeTypeOfLinkedTransaction` sentinel; `financeRepo`
      interface gains `GetTransaction`
- [x] 1.2 `UpdateTransaction`: when the patch sets `type` to something other
      than `income`, look up the existing transaction and reject if it has
      `contributionId` or `penaltyAssignmentId` set; skip the lookup
      entirely when the patch doesn't touch `type` or sets it to `income`
- [x] 1.3 `internal/finances/repository.go`: export the previously-private
      `getTransactionByID` as `GetTransaction`; reuse it both for the new
      guard and as `UpdateTransaction`'s existing no-op-patch fallback
- [x] 1.4 `internal/finances/handler.go`: map
      `ErrCannotChangeTypeOfLinkedTransaction` to `400 Bad Request`

## 2. OpenAPI contract
- [x] 2.1 `backend/openapi/openapi.yaml`: document the restriction on
      `updateTransaction` (type change rejected while linked; other fields,
      including amount, remain freely editable)
- [x] 2.2 `cd backend && make generate` — confirm no diff in
      `internal/gen/api.gen.go` beyond the description-carrying doc comment
- [x] 2.3 repo-root `make generate-ts` — confirm no diff (description text
      only, no schema/type shape change)

## 3. Tests
- [x] 3.1 `service_test.go`: reject type change on a transaction linked to
      a contribution
- [x] 3.2 `service_test.go`: reject type change on a transaction linked to
      a penalty assignment
- [x] 3.3 `service_test.go`: allow type change on an unlinked transaction
      (no regression)
- [x] 3.4 `service_test.go`: allow amount-only change on a linked
      transaction, and confirm `GetTransaction` is not called for a patch
      that doesn't touch `type`

## 4. Verification
- [x] 4.1 `openspec validate restrict-linked-transaction-type-change --strict`
- [x] 4.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [x] 4.3 `cd backend && go build ./... && go test ./internal/finances/...`
- [x] 4.4 `cd backend && golangci-lint run ./internal/finances/...`
