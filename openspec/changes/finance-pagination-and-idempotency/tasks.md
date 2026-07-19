## 1. Spec
- [ ] 1.1 Add `GET /teams/{teamId}/finances/transactions` (cursor-paginated) to `openapi.yaml`; keep overview for aggregates + first page — **deferred** (pagination part; see note)
- [x] 1.2 Replace `toggle-paid`/`toggle` with idempotent `PUT …/paid` taking an explicit `{"paid": bool}` value (`SetPaidRequest`); removed the old POST toggle endpoints
- [ ] 1.3 (Optional) Add an optional `date` to transaction create/update — **deferred** with the pagination part
- [x] 1.4 Ran `make generate` + repo-root `make generate-ts`; committed generated output (`api.gen.go`, `rbac_table.gen.go`, `types.gen.ts`, `zod.gen.ts`)

## 2. Backend
- [ ] 2.1 Implement the paginated transactions endpoint using `internal/pagination` keyset + cursors — **deferred**
- [x] 2.2 Idempotent paid-state `PUT`: `SetAssignmentPaid`/`SetContributionPaid` repositories (single `SET paid = $3` / `SET status = $3`, no read-then-write), `SetPenaltyPaid`/`SetContributionPaid` services + handlers reading `{paid}`; server aggregator + interfaces updated
- [ ] 2.3 (Optional) Honor a client-provided transaction date, defaulting to server date — **deferred**

## 3. Frontend
- [x] 3.1 Idempotent updates wired through: `serviceLayerReal` (`setPenaltyPaid`/`setContributionPaid` via `PUT {paid}`), `useSetPenaltyPaidMutation`/`useSetContributionPaidMutation`, `useFinanceActions` (`setPenaltyPaid(id, paid)`/`setContributionPaid(id, paid)`), UI call sites send `!current`, `AppContext` type + wiring, MSW handlers (`PUT …/paid`). Paginated-list consumption **deferred** with 2.1.

## 4. Verification
- [x] 4.1 `make generate`/`generate-ts` produce only the intended diff (openapi-drift green); RBAC table has the two new `PUT …/paid` finance ops
- [x] 4.2 `go test ./... -short` + `golangci-lint run ./...` (0 issues) green; frontend `typecheck` + full test (1135) + `build` + `check:bundle` green
- [x] 4.3 Idempotency covered by tests: repo concurrent-same-value test asserts the state doesn't flip; MSW handler test applies the same value twice; service/handler tests pass the value through

## 5. Deferred (follow-up: finance-pagination)
- [ ] 5.1 Cursor-paginated finance transaction list (removes the 1000-row overview cliff) + optional transaction date. Split out from the idempotency work landed above; the 1000-row cap only affects teams with >1000 rows in a single list (rare) and the pagination rework touches the finance overview UX + a new frontend list flow, warranting its own focused change.
