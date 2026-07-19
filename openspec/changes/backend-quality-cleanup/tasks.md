## 1. Dead-code import hacks
- [x] 1.1 Removed `var _ = time.Time{}` + orphaned "ensure time is used" comments in `auth/handler.go`, `events/service.go`, `members/service.go`, `members/handler.go`; dropped the now-unused `time` imports

## 2. Deduplicate toGenRole
- [x] 2.1 Exported the canonical mapper as `teams.ToGenRole`; removed the 3 copies in `events`/`roles`/`members` (roles' JSON-marshal variant folded into the explicit one — same output) and pointed all callers at `teams.ToGenRole`

## 3. Tenant scoping
- [x] 3.1 `CreateAssignment` snapshot read now `WHERE id = $1 AND team_id = $2`; added a regression test rejecting a penalty from another team (`..._RejectsPenaltyFromAnotherTeam`)

## 4. Error type URI default
- [x] 4.1 `apierror` default changed from the example domain to relative `/errors/`; updated the test's expected base and the doc comment (CLAUDE.md's "relative paths when unset" is now accurate)

## 5. Batch role insert
- [x] 5.1 `members.SetRoles` replaced the per-row loop with a single `INSERT … SELECT $1::uuid, r FROM unnest($2::uuid[])`, shrinking advisory-lock hold time (mirrors `CreateSeries`)

## 6. Verification
- [x] 6.1 `go test ./... -short` green
- [x] 6.2 `golangci-lint run ./...` 0 issues; `make generate` no drift (pure-Go change)
