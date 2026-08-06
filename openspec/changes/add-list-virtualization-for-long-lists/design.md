## Context

`FinancesTransactions` and `MembersPage` both render their full list from
data already loaded in full (e.g. via `AppContext`'s finances-overview
loader), not via a paginated fetch — there is no `useInfiniteQuery` usage
anywhere in `frontend/src/query/`, and no virtualization library is a
dependency. The backend already exposes real keyset pagination
(`backend/internal/pagination/`) for exactly these list endpoints, per
CLAUDE.md.

## Goals

- Bound the amount of data fetched and rendered for transactions and
  members as team history grows, without adding an unjustified new
  runtime dependency.
- Prefer using pagination the backend already supports over adding a
  client-only windowing library, per the project's "deliberately
  dependency-light" convention.

## Decisions

- **Pursue server-side pagination first (fetch a bounded page, load more
  on scroll or an explicit "load more" action), not DOM virtualization,
  as the primary fix.** Virtualization only bounds *rendering* cost but
  still fetches and holds the entire dataset in memory/network; keyset
  pagination bounds both, and the backend capability already exists —
  this is the smaller, more consistent change.
- **Only add a virtualization library (e.g. `react-window`) if, after
  moving to paginated fetching, a single page's render cost is still a
  measured problem** (e.g. an operator configures a very large page
  size). This is deliberately deferred: it's a real new runtime
  dependency requiring the justification `openspec/config.yaml` calls
  for, and premature before pagination's impact is measured.
- **Identify the exact current fetch path for each list as the first
  implementation task**, since both components currently receive
  already-loaded data as props from a parent/context loader rather than
  fetching directly — the loader, not the list component, is where
  pagination needs to be introduced.

## Risks

- **Introducing "load more" changes existing UX** (currently: everything
  visible immediately, e.g. for in-page search/filter across the full
  list). If either list supports client-side search across all items
  today, paginating breaks that unless search moves server-side too —
  confirm current behavior before implementing, and scope search
  handling explicitly if so.
