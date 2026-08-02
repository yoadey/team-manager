## 1. Investigation

- [ ] 1.1 Trace the current fetch path feeding `FinancesTransactions`
      and `MembersPage` (loader in `AppContext.tsx` or elsewhere) and
      confirm neither list currently supports client-side search/filter
      across the full unpaginated dataset (if it does, scope that
      separately per `design.md`'s risk note)

## 2. Transactions pagination

- [ ] 2.1 Wire the finances-transactions loader to the backend's keyset
      pagination (bounded page size, cursor-based "load more")
- [ ] 2.2 `FinancesTransactions.tsx`: render the current page, trigger
      loading the next page on scroll-near-bottom or an explicit action
- [ ] 2.3 Tests covering pagination boundary behavior (page size, cursor
      advance, end-of-list)

## 3. Members pagination

- [ ] 3.1 Same as 2.1-2.3 for the members loader/`MembersPage.tsx`

## 4. Verification

- [ ] 4.1 `npm run typecheck`
- [ ] 4.2 `npm test`
- [ ] 4.3 `npm run lint`
- [ ] 4.4 `npm run build` + `check:bundle` (confirms no new dependency
      regressed the bundle-size gate, if one was added)
