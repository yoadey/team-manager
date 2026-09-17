## Context

`flexible-membership-fees` introduced `transactions.contributionId` /
`transactions.penaltyAssignmentId` and made a transaction's `type` an
income-only requirement, but enforced that requirement only at creation
(`CreateTransaction`), not on every later edit — see that change's
`design.md`, "Linking is income-only, and only enforced at creation". The
stated reasoning was purely arithmetic: `paidAmount` is a live `SUM()` over
linked transactions filtered `WHERE type = 'income'`, so a linked
transaction whose `type` is changed to `expense` simply stops contributing
to the sum, and the contribution's/assignment's derived `status` recomputes
correctly on the next read. No separate guard was judged necessary "to keep
the arithmetic correct."

What that reasoning didn't weigh: a treasurer who changes a linked
transaction's `type` gets no signal that doing so detaches it from a fee or
fine. The contribution silently reverts from `paid`/`partial` to `open` (or
the assignment from `paid` to unpaid) with nothing in the response, the UI,
or the audit trail explaining why — indistinguishable from the payment
never having been recorded. A treasurer reviewing "why does this member
still owe €20" has no path back to "someone re-typed transaction #123 as an
expense."

## Goals

- Preserve the money: a linked transaction can never silently stop counting
  toward its contribution/penalty assignment due to an unrelated-looking
  field edit.
- Keep the correction path for a genuinely wrong link unchanged: delete and
  recreate the transaction, exactly as `flexible-membership-fees` already
  established (`UpdateTransactionRequest` has never supported repairing
  `contributionId`/`penaltyAssignmentId` itself).
- Change nothing else about linked-transaction editing — `amount`, `title`,
  `date`, `category`, `note` must remain freely editable on a linked
  transaction, since restricting those would regress a legitimate use case
  (fixing a typo'd payment amount) with no corresponding safety benefit.

## Decisions

**Guard `type` only, and only when it actually changes away from `income`.**
`UpdateTransaction` checks the incoming patch before touching the
repository: if `patch.Type == nil`, or `*patch.Type == "income"`, the guard
is skipped entirely — no extra read. Only a patch that sets `type` to
`expense` on a transaction triggers a `GetTransaction` lookup to check
`ContributionID`/`PenaltyAssignmentID`. This keeps the common case (no type
change) at the original one-query cost, and matches
`ErrPenaltyAssignmentRequiresIncome`'s existing create-time pattern of
gating on income-ness specifically rather than gating updates broadly.

**Reject with 400, not a silent unlink.** The alternative — automatically
clearing `contributionId`/`penaltyAssignmentId` when `type` moves off
`income` — was considered and rejected: it would still be a surprising,
unannounced side effect of an edit that looks unrelated, just a different
flavor of "the treasurer didn't mean for this to happen." An explicit 400
with an actionable message (delete and recreate) is more legible than an
automatic unlink the treasurer has to notice happened.

**No DB constraint.** Same reasoning `flexible-membership-fees/design.md`
already gives for the mutual-exclusivity invariant: a transaction's type
can only be *changed* via this one service method (`UpdateTransaction`), so
the one Go-level check is the only place the invariant could ever be
broken. A trigger or `CHECK` constraint would duplicate that with no
additional safety.

**`GetTransaction`, not a new query.** The repository already had a
private `getTransactionByID` used as `UpdateTransaction`'s no-op-patch
fallback. Exporting and reusing it for the new guard's lookup avoids a
second near-identical query; there is exactly one row shape
(`TransactionRow`) either caller needs.

## Risks / Trade-offs

- **An extra read on every type-changing update.** Accepted: type changes
  are rare relative to amount/note corrections, and the alternative (a
  broader guard or a DB trigger) either costs more or duplicates logic that
  belongs in one place per the existing design.
- **This narrows, rather than fully reverses, `flexible-membership-fees`'s
  "only enforced at creation" decision.** The rest of that trade-off stands:
  the mutual-exclusivity invariant (`contributionId` XOR `penaltyAssignmentId`)
  and the income-only-at-creation rule for `CreateTransaction` are still not
  re-checked on every field of every update — only the one field (`type`)
  whose change would silently detach an already-linked payment.
