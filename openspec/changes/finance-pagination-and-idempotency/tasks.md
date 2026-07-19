## 1. Spec
- [x] 1.1 Add `GET /teams/{teamId}/finances/transactions` (cursor-paginated) to `openapi.yaml`; keep overview for aggregates + first page
- [x] 1.2 Replace `toggle-paid`/`toggle` with idempotent `PUT …/paid` taking an explicit `{"paid": bool}` value (`SetPaidRequest`); removed the old POST toggle endpoints
- [x] 1.3 Add an optional `date` to transaction create/update (`CreateTransactionRequest`/`UpdateTransactionRequest`), defaulting to server date when omitted
- [x] 1.4 Ran `make generate` + repo-root `make generate-ts`; committed generated output (`api.gen.go`, `rbac_table.gen.go`, `types.gen.ts`, `zod.gen.ts`)

## 2. Backend
- [x] 2.1 Implement the paginated transactions endpoint using `internal/pagination` keyset + cursors: `Repository.ListTransactionsPage` (keyset on `(date, created_at, id)` DESC, no OFFSET), `Service.ListTransactions` (over-fetch by one, encode `TxCursor`), `Handler.ListTransactions`, server aggregator delegation, `NewService` now takes the shared `*pagination.Paginator`
- [x] 2.2 Idempotent paid-state `PUT`: `SetAssignmentPaid`/`SetContributionPaid` repositories (single `SET paid = $3` / `SET status = $3`, no read-then-write), `SetPenaltyPaid`/`SetContributionPaid` services + handlers reading `{paid}`; server aggregator + interfaces updated
- [x] 2.3 Honor a client-provided transaction date, defaulting to server date: `Service.CreateTransaction` uses `body.Date` when set, `TransactionPatch.Date` threaded through `UpdateTransaction`

## 3. Frontend
- [x] 3.1 Idempotent updates wired through: `serviceLayerReal` (`setPenaltyPaid`/`setContributionPaid` via `PUT {paid}`), `useSetPenaltyPaidMutation`/`useSetContributionPaidMutation`, `useFinanceActions` (`setPenaltyPaid(id, paid)`/`setContributionPaid(id, paid)`), UI call sites send `!current`, `AppContext` type + wiring, MSW handlers (`PUT …/paid`)
- [x] 3.2 Paginated transactions consumed in the walk-all-pages idiom: `serviceLayerReal.finances.listTransactions(teamId)` follows `{items, nextCursor}` via `fetchAllPages` (full history, no overview cap); MSW handler serves `GET …/transactions` as a single-page envelope; `date` forwarded on add/update transaction in both the real client and the MSW mock

## 4. Verification
- [x] 4.1 `make generate`/`generate-ts` produce only the intended diff (openapi-drift green); RBAC table has the new `GET/POST …/transactions` and the two `PUT …/paid` finance ops
- [x] 4.2 `go test ./... -short` + `golangci-lint run ./...` (0 issues) green; frontend `typecheck` + full test (1137) + `build` + `check:bundle` green
- [x] 4.3 Idempotency covered by tests: repo concurrent-same-value test asserts the state doesn't flip; MSW handler test applies the same value twice; service/handler tests pass the value through
- [x] 4.4 Pagination covered: repo integration test pages the whole history with no repeats and newest-first order + team scoping; service tests cover next-cursor/last-page/cursor-decode; handler tests cover default limit + invalid-cursor→400; date tests at service, repo, and MSW round-trip level

## 5. Notes
- Pagination scope is transactions only — that is the list the audit flagged as having a hard visibility cliff (no API path to transaction 1001). Penalty assignments and contributions remain served through the bounded overview list; extending the same keyset helper to them is a mechanical follow-up if a team ever exceeds the cap there (far rarer than transaction volume).
