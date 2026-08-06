## Context

`contributions` currently exists only as demo seed data
(`frontend/src/mocks/db.ts`'s `CO()`, six months × every member); there is no
`createContribution` operation and never has been — `UpdateContribution` and
`SetContributionPaid` are the only writes. The redesign requested by the
Kassenwart replaces the "one row auto-generated per member per month" shape
with one the treasurer drives entirely by hand: name it, price it, optionally
due-date it, pick who owes it, done. This is the direct analogue of how
`penalties`/`penalty_assignments` already work (a definition + who it's
assigned to), but simpler — a contribution has no reusable catalog, since the
whole point of this change is that nothing recurs automatically.

## Goals / Non-Goals

**Goals:**
- Free-text fee name + optional due date, replacing the fixed `month`.
- One creation action assigns the same fee to several members at once
  (fan-out to one row per member), rather than the Kassenwart repeating a
  single-member form N times.
- A fee can be paid in installments; each installment is a real, bookkept
  income transaction, and the fee's paid-so-far amount is always the live
  sum of the transactions actually linked to it — never a second, separately
  editable number that can drift from the ledger.
- create/update/delete symmetry with `transactions`/`penalties`, since
  contributions are now genuinely user-created.

**Non-Goals:**
- No reusable "fee catalog" the way `penalties` has one — every fee is a
  one-off definition; a recurring fee is created again by hand each period,
  exactly as requested ("wenn er einen regelmäßigen Beitrag über mehrere
  Monate haben will muss er jeden Beitrag manuell anlegen").
- No editing which fee a booked transaction is linked to after creation
  (`UpdateTransactionRequest` does not gain `contributionId`) — linking only
  happens at the moment a payment is recorded. A wrongly-linked transaction
  is deleted and re-created, same as any other transaction correction today
  (`UpdateTransactionRequest` already can't repair every kind of mistake
  either, e.g. it can't move a transaction to a different team).
- No UI/API way to clear an already-set `dueDate` back to "no due date" via
  `UpdateContributionRequest` — this codebase's oapi-codegen setup collapses
  `nullable: true` + optional into a single Go pointer with no way to
  distinguish "field omitted" from "field explicitly null" (confirmed
  against the existing `PenaltyAssignment.penaltyId`/`CancelLeadMinutes`
  fields, the only other nullable-optional fields in the schema — neither
  supports explicit-clear either). Consistent with the rest of this API, not
  a new gap introduced here.
- No DB-level `NOT NULL` on `contributions.name` (see Decisions below).

## Decisions

**`name` stays nullable at the DB level; required only at the API layer.**
CI's `backend-migration-safety` lint unconditionally flags any
`ALTER COLUMN ... SET NOT NULL` (full-table `ACCESS EXCLUSIVE` scan), and its
own established safe replacement — `ADD CONSTRAINT ... CHECK (...) NOT VALID`
then `VALIDATE CONSTRAINT` in a separate migration/transaction, never a raw
`SET NOT NULL` — is meaningfully more migration machinery than a low-traffic,
per-team-small table like `contributions` warrants. `CreateContributionRequest.name`
and `UpdateContributionRequest.name` are `required`/validated
(`minLength: 1`) at the OpenAPI/service layer instead, the same trust
boundary every other "required" text field in this API (`Penalty.label`,
`Transaction.title`) already relies on for update-time invariants (their
`NOT NULL` came for free only because they were `NOT NULL` from their
table's original `CREATE TABLE`, not because this codebase has a general
practice of enforcing requiredness in the DB after the fact).

**Fan-out create is one DB transaction, all-or-nothing.**
`Repository.CreateContributions` loops the given `userIds` inside a single
`pgx` transaction, inserting one row per id with the same atomic
`WHERE EXISTS (SELECT 1 FROM memberships ...)` re-check
`finances.Repository.CreateAssignment` already uses to close the TOCTOU
window between the service layer's membership check and the insert. If any
id fails that re-check (removed from the team in the narrow window between
request and insert), the whole transaction rolls back rather than leaving a
partial fan-out — a fee half-assigned to the roster is worse than one that
visibly failed and can be retried whole.

**`paidAmount`/`status` are computed, never stored.** `ListContributions`
and `getContributionByID` join a `LATERAL` subquery summing
`transactions.amount` for the contribution's linked rows where
`type = 'income'` (`LEFT JOIN LATERAL ... ON true`, using the new partial
index on `transactions.contribution_id`). `status` is derived in Go
(`paidAmount == 0` → `open`, `paidAmount >= amount` → `paid`, else
`partial`) rather than in SQL, matching how `FinanceOverview.OpenPenaltySum`
is already summed in Go rather than SQL elsewhere in this service. Deleting
or editing a linked transaction is automatically reflected — there is no
second write path to keep in sync.

**Linking is income-only, and only enforced at creation.**
`CreateTransactionRequest.contributionId` is rejected (400) when
`type != income` — paying a due doesn't make sense as an expense. This is
*not* re-enforced as a DB constraint or on every future edit: if a linked
transaction is later edited to `type: expense` via the existing
`UpdateTransactionRequest` (which does not carry `contributionId`), the
`paidAmount` sum query's own `type = 'income'` filter already stops counting
it — no separate guard is needed to keep the arithmetic correct, only to
give a clear error at the one point a mismatch could otherwise be created by
mistake.

**`transactions.contribution_id` is `ON DELETE SET NULL`, not `CASCADE`.**
Deleting a contribution (new `DeleteContribution`) must never delete the
income it already booked — that money was genuinely received regardless of
whether the fee record describing it still exists. The transaction survives,
just unlinked.

**Dashboard `contribOpen` count now means "not fully paid" (open + partial),
not just `status = 'open'`.** A partially-paid fee still needs the
Kassenwart's attention the same way a fully-open one does; collapsing
`partial` out of the count would make it undercount real outstanding fees.

## Risks

- **Migration is destructive** (drops `month`/`status`, and this is a
  from-scratch alpha app per `00001_init.sql`'s own precedent for
  irreversible schema changes — see `00012_remove_event_rsvp_deadline.sql`).
  Any contribution row that exists only as seed/demo data today loses its
  month grouping on `Up`; there is no production deployment to migrate.
- **Fan-out create UX**: a Kassenwart selecting "all members" for a large
  roster creates that many rows in one call. `maxContributionsPerTeam`
  (mirroring `maxTransactionsPerTeam`/`maxAssignmentsPerTeam`) and an
  OpenAPI `maxItems` on `userIds` cap this the same way the other
  finance-module flood limits already do.
