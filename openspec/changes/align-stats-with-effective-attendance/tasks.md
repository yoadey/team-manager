## 1. Shared effective-status expression
- [x] 1.1 Extract the effective-status CASE (+ absence EXISTS) into `internal/attendance` (exported `EffectiveStatusExpr`/`AbsenceCoversExpr`), reused by events and stats
- [x] 1.2 Confirm the event-summary behavior is unchanged: events now references the shared const (byte-identical SQL); events unit tests stay green

## 2. Stats queries
- [x] 2.1 Rewrite `stats/repository.go` (MemberStats/EventStats/SingleMemberStats) to be roster-driven and use the effective-status expression (yes/counted)
- [x] 2.2 Reconcile ex-member handling: EventStats now joins `memberships` (scores current members only), matching MemberStats — a departed member's retained attendance row no longer inflates event counts. Also fixed `has_photo` to consider `photo_object_key` (post object-storage migration).

## 3. Tests
- [x] 3.1 Add stats tests: opt-out event with no responses → attending (`..._OptOutDefaultsToAttending`); covering absence → not attending (`..._AbsenceDefaultsToNotAttending`)
- [x] 3.2 EventStats test updated to seed a membership (previously counted a non-member's attendance); roster-driven join reconciles event-level vs member-level counts

## 4. Verification
- [x] 4.1 `go test ./internal/events/... ./internal/stats/... -short` + full `go test ./... -short` green (integration cases run in CI)
- [x] 4.2 `go vet` + `gofmt` clean (full `make lint` runs in CI); coverage gate confirmed by CI
