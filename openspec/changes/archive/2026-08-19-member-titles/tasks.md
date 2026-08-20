## 1. Database
- [x] 1.1 `00032_member_title.sql`: `ALTER TABLE memberships ADD COLUMN title TEXT` (nullable, no default), `+ goose Down` drops it
- [x] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's `backend-migration-rollback` (up→down→up) and `backend-migration-safety` gates

## 2. OpenAPI
- [x] 2.1 `Member`: add optional `title` (`string`, `maxLength: 40`)
- [x] 2.2 `UpdateMemberRequest`: add optional `title` (`string`, `maxLength: 40`)
- [x] 2.3 `AttendanceRow`: add optional `title` (`string`, `maxLength: 40`)
- [x] 2.4 New `SetMemberTitleRequest` schema: required `title` (`string`, `maxLength: 40`, empty string allowed to clear)
- [x] 2.5 New `PUT /teams/{teamId}/members/{membershipId}/title`, `operationId: setMemberTitle`, `x-rbac-module: members`, `x-rbac-self-service: true`, request `SetMemberTitleRequest`, `200` response `Member`
- [x] 2.6 `cd backend && make generate` (commit `internal/gen/api.gen.go`, `internal/middleware/rbac_table.gen.go`)
- [x] 2.7 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: members module
- [x] 3.1 `model.go`: `MemberRow.Title *string`, `MemberPatch.Title *string`
- [x] 3.2 `repository.go`: `ListMembers`/`getMemberByMembershipIDQ`/`scanMemberRow` select+scan `m.title`; `UpdateMember` patches `title` alongside `group` (existing `patch.Group != nil` block pattern); new `SetMemberTitle(ctx, membershipID, teamID, callerUserID, title *string) (*MemberRow, error)` — looks up the membership's `user_id` scoped to `teamID`, returns `pgx.ErrNoRows` if not found, returns `ErrCannotSetOthersTitle` if `callerUserID` doesn't match, else updates `title` and reads the row back (reuse `getMemberByMembershipIDQ`); new `ErrCannotSetOthersTitle` var
- [x] 3.3 `service.go`: thin passthrough `SetMemberTitle` wrapping the repository call, converting `*MemberRow` to `gen.Member`
- [x] 3.4 `handler.go`: `validateMemberPatch` gains a `body.Title != nil` branch (`validate.MaxLen(*body.Title, 40, "title")`); new `SetMemberTitle` handler method — trims the request title, validates `MaxLen` (40) on the trimmed value, maps trimmed-empty to `nil`, calls `svc.SetMemberTitle`, maps `pgx.ErrNoRows`→404, `ErrCannotSetOthersTitle`→403, audits success/failure (new `audit.EventMemberUpdate`-style call, reusing the existing event kind)
- [x] 3.5 `internal/server/server.go`: wire `SetMemberTitle` into the members handler registration

## 4. Backend: events module (AttendanceRow passthrough)
- [x] 4.1 `model.go`/wherever `AttendanceRow`'s row type lives: add `Title *string`
- [x] 4.2 `repository.go`: attendance-listing query selects `m.title` alongside `m."group"`, scan into the new field
- [x] 4.3 `service.go`: map the new field onto `gen.AttendanceRow.Title`

## 5. Backend: tests
- [x] 5.1 `repository_test.go`: `SetMemberTitle` happy path, not-found, `ErrCannotSetOthersTitle`, clearing via empty string, `UpdateMember` still sets `title` via the admin path
- [x] 5.2 `service_test.go`: passthrough coverage
- [x] 5.3 `handler_test.go`: `PUT .../title` 200/403/404, `MaxLen` rejection (>40 chars) on both `PATCH .../members/{id}` and `PUT .../title`; RBAC classification of the new self-service route itself is covered generically by `genrbac`'s generated table + existing middleware tests, not duplicated per-package
- [x] 5.4 `internal/events` tests: attendance list includes `title` when set

## 6. Frontend
- [x] 6.1 `features/members/types.ts`: `Member.title?: string`
- [x] 6.2 `api/map.ts`: map `title` on `Member` and `AttendanceRow`
- [x] 6.3 `services/serviceLayerReal.ts`: `members.setMyTitle(membershipId, title)` calling the new endpoint; `members.update` (existing) passes `title` through as part of the bulk patch
- [x] 6.4 `mocks/db.ts` + `mocks/handlers.ts`: seed `title: undefined` by default on demo members; handler for `PUT .../members/:id/title` enforcing self-only (403 otherwise) and 40-char validation; existing update-member handler accepts `title`
- [x] 6.5 `features/members/memberFormSchema.ts`: add optional `title` (max 40) to the admin edit form schema
- [x] 6.6 `features/members/components/MemberSheets.tsx`: add a `title` `Field` to `MemberFormSheet` (after `address`), ungated like every other profile field on that form (the form itself is reachable only via the existing `canWrite || isMe` edit-button gate)
- [x] 6.7 `features/members/hooks/useMemberActions.ts`: thread `title` through the existing save-member action; new `setMyTitle` action calling the service method
- [x] 6.8 `context/AppContext.tsx`: expose `setMyTitle`
- [x] 6.9 `features/members/MembersPage.tsx`: new small text line under the member's name (above the existing roles subtitle), shown only when `m.title` is set
- [x] 6.10 `features/events/components/EventDetailSheet.tsx` (`AttendanceRowItem`): extend the existing `group`-else-branch to `[row.group, row.title].filter(Boolean).join(' · ')`
- [x] 6.11 `features/settings/components/ProfilePanel.tsx`: add a small "Titel" text field + save button calling `setMyTitle`, with inline 40-char validation and success/error feedback
- [x] 6.12 `i18n/{en.ts,de.ts}`: `members.fieldTitle`/`fieldTitlePlaceholder`/`fieldTitleError`/`fieldTitleHint` (reused by both the admin edit form and Profile Settings, rather than a separate `profile.*` namespace)

## 7. Frontend: tests
- [x] 7.1 `MembersPage.test.tsx`: title line renders when set, absent when not
- [x] 7.2 `EventDetailSheet.test.tsx`: attendance row shows `group · title` combined correctly (title only, group only, both, neither)
- [x] 7.3 `MemberSheets.test.tsx`: title field save + 40-char validation
- [x] 7.4 `ProfilePanel.test.tsx`: self-service title set/clear, validation, error handling
- [x] 7.5 `services/serviceContract.test.ts`: new "member titles" describe block (self set/clear, cross-member rejection, admin path) against the MSW demo backend

## 8. Verification
- [x] 8.1 `openspec validate member-titles --strict`
- [x] 8.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [x] 8.3 `cd backend && make lint`
- [x] 8.4 `cd backend && make test`
- [x] 8.5 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
