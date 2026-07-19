## 1. Shared effective-status expression
- [ ] 1.1 Extract the event summary's effective-status CASE into a shared SQL snippet reusable by events and stats
- [ ] 1.2 Confirm the event-summary behavior is unchanged (its existing tests stay green)

## 2. Stats queries
- [ ] 2.1 Replace explicit-only `FILTER (WHERE a.status ...)` in `stats/repository.go` with the effective-status expression (yes/counted)
- [ ] 2.2 Reconcile ex-member handling between event-level and member-level aggregations (apply the same membership filter)

## 3. Tests
- [ ] 3.1 Add stats tests: opt-out event with no responses → attending; overlapping absence → not attending
- [ ] 3.2 Add a test reconciling event-level vs member-level counts after a member leaves

## 4. Verification
- [ ] 4.1 `cd backend && make test` green (stats + events)
- [ ] 4.2 `make lint` green; coverage gate holds
