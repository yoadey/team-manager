## 1. Dead-code import hacks
- [ ] 1.1 Remove `var _ = time.Time{}` in `auth/handler.go`, `events/service.go`, `members/service.go`, `members/handler.go`; run `gofumpt`/`goimports` to drop now-unused imports

## 2. Deduplicate toGenRole
- [ ] 2.1 Add one shared mapper (in `teams`, where `RoleRow` lives); replace the 4 copies with calls to it

## 3. Tenant scoping
- [ ] 3.1 Add `AND team_id = $2` to the `CreateAssignment` snapshot SELECT (`finances/repository.go`)

## 4. Error type URI default
- [ ] 4.1 Change `apierror` default from the example domain to relative (or empty); update `CLAUDE.md`; keep the `backend-lint` type-URI check green

## 5. Batch role insert
- [ ] 5.1 Replace the per-row `membership_roles` insert in `members/repository.go` with an `UNNEST` batch insert

## 6. Verification
- [ ] 6.1 `cd backend && make test` green
- [ ] 6.2 `make lint` green (incl. type-URI-literal check); coverage gate holds
- [ ] 6.3 `make generate` produces no diff (if any generated code touched)
