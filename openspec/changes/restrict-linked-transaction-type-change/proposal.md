## Why

`flexible-membership-fees` (`openspec/changes/flexible-membership-fees/design.md`,
"Linking is income-only, and only enforced at creation") deliberately chose not
to re-check a transaction's link when it is edited: the `paidAmount` sum query
already filters `type = 'income'`, so the arithmetic stays correct even if a
linked transaction's `type` is later changed away from `income`. That trade-off
undercounted the human cost: `PATCH /teams/{teamId}/finances/transactions/{id}`
(`backend/internal/finances/service.go`'s `UpdateTransaction`) let a treasurer
change a transaction's `type` from `income` to `expense` while it still carried
a `contributionId`/`penaltyAssignmentId`, silently dropping it out of that fee's
or fine's `paidAmount` sum with no error, confirmation, or audit trail — the fee
would revert from "paid"/"partial" to "open" with no visible cause. Retyping a
transaction is also not how an incorrect link is meant to be fixed; there was
never an endpoint to edit `contributionId`/`penaltyAssignmentId` itself.

## What Changes

- `UpdateTransaction` now rejects (400, `ErrCannotChangeTypeOfLinkedTransaction`)
  a request that changes `type` away from `income` on a transaction that still
  has `contributionId` or `penaltyAssignmentId` set. This amends the
  "only enforced at creation" trade-off in `flexible-membership-fees/design.md`
  for this one case; the rest of that trade-off (no DB constraint, no
  re-validation of the mutual-exclusivity invariant on update) is unchanged.
- Editing any other field on a linked transaction — most importantly `amount`,
  the treasurer's main correction path for a typo'd payment — remains allowed
  and unaffected.
- The correction path for a wrongly-linked transaction stays delete-and-recreate,
  as `UpdateTransactionRequest` still cannot change `contributionId`/
  `penaltyAssignmentId` itself; the new error message says so explicitly.

## Capabilities

- `membership-fees` (MODIFIED): the "only enforced at creation" trade-off now
  carries one update-time exception.

## Impact

- `backend/internal/finances/service.go`: new `ErrCannotChangeTypeOfLinkedTransaction`
  sentinel; `UpdateTransaction` fetches the existing row (new
  `financeRepo.GetTransaction`) only when the patch changes `type` away from
  `income`, and rejects the change if the row is still linked.
- `backend/internal/finances/repository.go`: the previously-private
  `getTransactionByID` is exported as `GetTransaction` and reused both as the
  new guard's lookup and as `UpdateTransaction`'s existing no-op-patch fallback.
- `backend/internal/finances/handler.go`: maps the new sentinel to `400 Bad Request`.
- `backend/openapi/openapi.yaml`: documents the restriction on `updateTransaction`.
- `backend/internal/finances/service_test.go`: covers reject-on-linked-contribution,
  reject-on-linked-penalty-assignment, allow-on-unlinked, and
  allow-amount-only-change-on-linked.
