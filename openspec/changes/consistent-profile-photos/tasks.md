## 1. Spec + backend
- [x] 1.1 Add `membershipId` to `AttendanceRow` in `openapi.yaml`
- [x] 1.2 Populate `membershipId` when assembling the attendance matrix (`internal/events`/`internal/attendance`), preserving team-scoping
- [x] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Shared avatar
- [x] 2.1 Add one shared avatar component/helper built on `api/map.ts`'s `ln()` URL rule (photo when `hasPhoto`, else colored initials; consistent size + cache-busting) — already existed as `Av` (`frontend/src/components/ui.tsx`) and `photoUrl()`; added the shared `memberPhotoUrl()` URL-building helper used by every mapper below
- [x] 2.2 Route event attendance rows through it (fixes photos at events using the new `membershipId`) — `EventDetailSheet.tsx` already rendered attendance rows via `<Av>`; `mapAttendanceRow` now resolves `photo` from `membershipId`/`hasPhoto` instead of always `null`

## 3. Audit + migrate all person renderers
- [x] 3.1 Inventory every place a person/avatar is rendered (event detail, poll voters, members, notifications, comments, absences) — all already rendered through `<Av>`; the gap was solely in the API mappers always passing `photo: null`
- [x] 3.2 Migrate each to the shared component; extend any payload minimally where `hasPhoto`/id is missing — added `authorMembershipId` (`EventComment`), `memberMembershipId` (`Absence`), `actorMembershipId` (`AppNotification`) to the OpenAPI schema + backend population; poll voters and members already carried `membershipId`/`hasPhoto`
- [x] 3.3 Remove the now-duplicated ad-hoc avatar rendering that this change orphaned — none found; every call site already used `<Av>`

## 4. Verification
- [x] 4.1 openapi-drift green (regenerated clients committed)
- [x] 4.2 Backend tests: `AttendanceRow.membershipId` populated and team-scoped (plus `EventComment.authorMembershipId`, `Absence.memberMembershipId`, `AppNotification.actorMembershipId`)
- [x] 4.3 Frontend tests: photos render at events; fallback-to-initials still works; each migrated site uses the shared component
- [x] 4.4 `golangci-lint` + `go test ./...` green; frontend lint/typecheck/test/build + bundle budget green
