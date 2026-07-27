## 1. Spec + backend
- [ ] 1.1 Add `membershipId` to `AttendanceRow` in `openapi.yaml`
- [ ] 1.2 Populate `membershipId` when assembling the attendance matrix (`internal/events`/`internal/attendance`), preserving team-scoping
- [ ] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Shared avatar
- [ ] 2.1 Add one shared avatar component/helper built on `api/map.ts`'s `ln()` URL rule (photo when `hasPhoto`, else colored initials; consistent size + cache-busting)
- [ ] 2.2 Route event attendance rows through it (fixes photos at events using the new `membershipId`)

## 3. Audit + migrate all person renderers
- [ ] 3.1 Inventory every place a person/avatar is rendered (event detail, poll voters, members, notifications, comments, absences)
- [ ] 3.2 Migrate each to the shared component; extend any payload minimally where `hasPhoto`/id is missing
- [ ] 3.3 Remove the now-duplicated ad-hoc avatar rendering that this change orphaned

## 4. Verification
- [ ] 4.1 openapi-drift green (regenerated clients committed)
- [ ] 4.2 Backend tests: `AttendanceRow.membershipId` populated and team-scoped
- [ ] 4.3 Frontend tests: photos render at events; fallback-to-initials still works; each migrated site uses the shared component
- [ ] 4.4 `golangci-lint` + `go test ./...` green; frontend lint/typecheck/test/build + bundle budget green
