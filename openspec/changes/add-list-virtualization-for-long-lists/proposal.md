## Why

`features/finances/components/FinancesTransactions.tsx:114-126` renders
every transaction in an unbounded flex column; `features/members/
MembersPage.tsx:142-143` does the same for members. No virtualization
library is used anywhere in the frontend (`grep` for `react-window`/
`react-virtuoso`/similar across `frontend/src` finds nothing). For a
sports club with years of transaction history, this is an ever-growing
DOM with no pagination boundary on the client — the backend already has
real keyset pagination (`backend/internal/pagination/`, per CLAUDE.md)
that these lists don't take advantage of client-side.

## What Changes

- Pair the frontend's transaction and member lists with the backend's
  existing keyset pagination (fetch a page at a time, load more on
  scroll/explicit action) rather than fetching and rendering everything
  at once — the primary fix, and consistent with the project's
  "deliberately dependency-light" convention (no new runtime dependency
  required).
- Evaluate whether pagination alone is sufficient or a lightweight
  windowing approach is also needed for very long single pages; document
  the decision and any new dependency justification in `design.md`
  before adding one (per `openspec/config.yaml`'s rule to justify new
  runtime deps).

## Capabilities

### Added Capabilities
- `list-performance`: long, growing lists (transactions, members) are
  paginated client-side against the backend's existing keyset
  pagination, instead of fetching and rendering the entire dataset in
  one unbounded DOM list.

## Impact

- `frontend/src/features/finances/components/FinancesTransactions.tsx`.
- `frontend/src/features/members/MembersPage.tsx`.
- Corresponding React Query hooks in `frontend/src/query/` — adopt
  paginated fetching where they currently fetch a single unbounded page.
- No backend changes (pagination already exists); no migration.
