## 1. Spec
- [x] 1.1 Add a targeted-team-ids set to event create/update/read schemas in `openapi.yaml`; define a restricted `CrossTeamAttendee` (name, avatarColor, hasPhoto, status, teamName?) with no profile identifiers
- [x] 1.2 Specify create/update requiring `events:write` in all targeted teams; read/RSVP requiring membership in any targeted team
- [x] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Data model + migration
- [x] 2.1 Migration: `event_teams(event_id, team_id)` join (owning `team_id` retained for back-compat). No attendance migration needed — `attendance` is already `UNIQUE(event_id, user_id)` with no `team_id` column.

## 3. Backend
- [x] 3.1 Persist/read targeted teams; visibility = caller is a member of any targeted team
- [x] 3.2 Authorization: create/update validates `events:write` across all targets; read/RSVP validates membership in any target; single-team events keep the existing `AND team_id = $N` fast path unchanged
- [x] 3.3 Attendance assembly dedups by user (already natural via the existing unique constraint); each attendee labelled per the display rule — no badge if they're in the viewer's own team, else the alphabetically-first (by name) team from memberships ∩ targeted teams; expose only the restricted projection for foreign attendees (no email/phone/profile id)
- [x] 3.4 RSVP: a multi-team member has one row and one RSVP

## 4. Frontend
- [x] 4.1 Create form: multi-team picker showing only teams where the creator has `events:write`; server re-validates
- [x] 4.2 Event detail: attendance shows each multi-team member once with their single team badge per the display rule; no badge for viewer's-own-team attendees; no profile navigation for foreign attendees
- [x] 4.3 `de`/`en` strings; MSW handlers for multi-team events

## 5. Docs
- [x] 5.1 Document the controlled multi-team exception to the team-scoping invariant in `CLAUDE.md` (RBAC/Architecture)

## 6. Verification
- [x] 6.1 openapi-drift green; RBAC behavior covered
- [x] 6.2 Backend tests: create allowed only with write-in-all-targets; read/RSVP for any-target member; multi-team member = one row/one RSVP; single-team events unchanged; no foreign PII (email/phone/profile) leaks
- [x] 6.3 `golangci-lint` + `go test ./...` + govulncheck green; migration-rollback + migration-safety green
- [x] 6.4 Frontend lint/typecheck/test/build + bundle budget green
