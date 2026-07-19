## 1. Spec
- [ ] 1.1 Add `GET /teams/{teamId}/finances/transactions` (cursor-paginated) to `openapi.yaml`; keep overview for aggregates + first page
- [ ] 1.2 Replace `toggle-paid`/`toggle` with idempotent `PUT` taking an explicit `{"paid": bool}` value
- [ ] 1.3 (Optional) Add an optional `date` to transaction create/update
- [ ] 1.4 Run `make generate` + repo-root `make generate-ts`; commit generated output

## 2. Backend
- [ ] 2.1 Implement the paginated transactions endpoint using `internal/pagination` keyset + cursors
- [ ] 2.2 Implement idempotent paid-state `PUT` handlers/services/repositories
- [ ] 2.3 (Optional) Honor a client-provided transaction date, defaulting to server date

## 3. Frontend
- [ ] 3.1 Update the finance TanStack Query hooks/components to consume the paginated endpoint and the `PUT` idempotent updates

## 4. Verification
- [ ] 4.1 `make generate`/`generate-ts` no diff (openapi-drift green)
- [ ] 4.2 `make test` + `make lint` green; frontend `typecheck`/`test`/`build` green
- [ ] 4.3 Coverage + bundle-budget gates hold
