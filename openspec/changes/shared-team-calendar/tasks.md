## 1. Spec
- [x] 1.1 Add grant/revoke/list calendar-share endpoints (settings-gated on the owning team) to `openapi.yaml`
- [x] 1.2 Add a read endpoint returning a redacted `SharedCalendarEvent` (id, title, type, date, startTime, endTime, location) — no attendance/comments/notes/participants
- [x] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Migration + backend
- [x] 2.1 Migration: `calendar_shares(owner_team_id, viewer_team_id, created_at)`, unique per pair
- [x] 2.2 Grant/revoke/list handlers requiring `settings:write` on the owning team
- [x] 2.3 Redacted read: a query selecting only the projection columns (never the full event serializer); authorize on viewer-team membership + an existing grant
- [x] 2.4 Revocation takes effect immediately (read denied once the grant is gone)

## 3. Frontend
- [x] 3.1 Settings UI to manage which teams a calendar is shared with
- [x] 3.2 Shared-calendar view rendering only redacted fields; no attendance/comment UI for shared events
- [x] 3.3 `de`/`en` strings; MSW handlers

## 4. Verification
- [x] 4.1 openapi-drift green (`make generate` / `make generate-ts` produce no diff against the committed output)
- [x] 4.2 Backend tests: grant/revoke/list settings-gated; redacted read returns only time/location/title/type; **no** attendance/comments/notes/participants leak; revoked grant → denied
- [x] 4.3 `golangci-lint` + `go test ./...` green. `migration-rollback`/`migration-safety` reviewed by hand (new migration only adds a table + a `CREATE INDEX CONCURRENTLY`, split into its own `NO TRANSACTION` migration per this repo's safety-lint rules) but not executed — no Docker/Postgres in this sandbox, same limitation noted on prior changes (e.g. `self-service-registration`). `govulncheck` not run — no network egress to vuln.go.dev in this sandbox.
- [x] 4.4 Frontend lint/typecheck/test/build + bundle budget green
