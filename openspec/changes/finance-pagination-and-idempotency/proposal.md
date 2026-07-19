## Why

Two finance-API findings from the architecture audit remain:

1. **Unpaginated overview with a hard visibility cliff.** `GET /teams/{teamId}/finances` returns all lists in one response; `finances/repository.go` caps display lists at `maxOverviewRows = 1000` and the code itself notes a team over the cap "can no longer see" older rows — there is no API path to transaction 1001.
2. **Non-idempotent toggle RPCs.** `POST .../penalty-assignments/{id}/toggle-paid` and `.../contributions/{id}/toggle` flip state; a client retry after a lost response silently reverts a paid penalty to unpaid.

Optionally: **transactions have no settable date** (`CreateTransactionRequest` has no `date`; the server stamps `time.Now()`), so back-dating a receipt is impossible.

## What Changes

- Add cursor-paginated list endpoints for finance transactions (and penalties/assignments as needed), keeping the overview for aggregates + a first page. Removes the 1000-row cliff and aligns with the keyset pagination used elsewhere.
- Replace the toggle RPCs with idempotent `PUT` setting an explicit target value (`{"paid": true|false}` / `{"paid": …}`), so retries are safe.
- (Optional) Add an optional `date` field to transaction create/update.

## Capabilities

### New Capabilities
- `finance-listing`: paginated, retry-safe access to finance data without a hard visibility cap.

### Modified Capabilities
<!-- none -->

## Impact

- Backend + spec: `openapi/openapi.yaml` (new list endpoints, `PUT` replacing toggles, optional tx date), regenerated `internal/gen` + `frontend/src/api/*` + sqlc, `finances/*` handlers/services/repositories, tests.
- Frontend: finance hooks/components consuming the new endpoints (the TanStack Query finance vertical).
- CI: openapi-drift, backend + frontend gates. **API-affecting** — sequence after the smaller changes.
