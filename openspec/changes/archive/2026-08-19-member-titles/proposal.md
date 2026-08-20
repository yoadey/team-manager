## Why

There is no way for a member to attach a small, informal, self-chosen label
to themselves — something like "Orgaente" or "Witzbeauftragter": in-joke
team roles that carry no permissions and aren't part of RBAC (`roles`), just
a bit of personality shown alongside the person's name wherever there's
room for it (the member list, an event's attendance list). Today the only
per-membership free-text field close to this, `group`, is admin-only
(editable exclusively via `PATCH /teams/{teamId}/members/{membershipId}`,
gated on `members:write`) and isn't even wired into any edit form
(`frontend/src/features/members/components/MemberSheets.tsx`'s
`MemberFormSheet` never renders a `group` input) — there is no path, self-
or admin-driven, for a plain member to set anything like this about
themselves.

## What Changes

- **New self-service `title` field on a membership** (`memberships.title`,
  nullable `TEXT`, max 40 characters): a short, optional, purely cosmetic
  label the member picks for themselves. Not a role, not RBAC-relevant, not
  moderated by permission level beyond the team having `members` visible to
  them at all (`members: none` hides it, same as every other members-module
  read).
- **New self-service endpoint** `PUT /teams/{teamId}/members/{membershipId}/title`
  (`x-rbac-module: members`, `x-rbac-self-service: true`): lets a member set
  or clear (empty string) *their own* title without needing `members:write`
  — mirroring how `POST .../auth/me/photo` already lets a member manage
  their own photo outside the admin-gated member-update path. Setting
  another membership's title through this endpoint is rejected
  (`ErrCannotSetOthersTitle`), the same shape as the existing
  `ErrCannotChangeOthersEmail` guard on `UpdateMember`.
- **`title` also added to the existing admin path**: `Member`,
  `UpdateMemberRequest`, and `AttendanceRow` all gain an optional `title`
  string, so a `members:write` holder can set or clear any member's title
  from Members management (e.g. to moderate an inappropriate one) the same
  way they already manage `group`.
- **Display, small and secondary, wherever there's room:**
  - Member list (`MembersPage.tsx`): a new small line under the member's
    name, above the existing roles subtitle.
  - Event attendance list (`AttendanceRowItem` in `EventDetailSheet.tsx`):
    folded into the row's existing secondary line (today: RSVP comment, or
    else `group`) — that line is the only spare room in an already-compact
    row, so title joins `group` there (`group · title`) rather than adding
    a fourth line.
- **Self-service editing surface**: `ProfilePanel.tsx` (Profile Settings —
  today read-only: avatar, name, email) gains a small "Titel" text field +
  save action, calling the new self-service endpoint. This is the only
  profile field a plain member (default role: `members: read`) can edit
  about themselves today; existing name/phone/address edits still go
  through the admin-gated `MemberFormSheet`, unchanged.

## Capabilities

### New Capabilities
- `member-titles`: a self-service, non-RBAC, cosmetic short label per
  membership, editable by the member themselves (own title only) or by a
  `members:write` holder (any member's title), displayed in small
  secondary text in the member list and event attendance list.

## Impact

- Database: new migration `backend/internal/db/migrations/00032_member_title.sql`
  (`ALTER TABLE memberships ADD COLUMN title TEXT`).
- API contract: `backend/openapi/openapi.yaml` — `Member`,
  `UpdateMemberRequest`, `AttendanceRow` gain `title`; new
  `SetMemberTitleRequest` schema; new `setMemberTitle` operation. Regenerate
  `internal/gen/api.gen.go`, `internal/middleware/rbac_table.gen.go`,
  `frontend/src/api/types.gen.ts`.
- Backend: `internal/members/{model.go,repository.go,service.go,handler.go}`,
  `internal/events/{repository.go,service.go}` (AttendanceRow title
  passthrough), `internal/server/server.go`.
- Frontend: `features/members/{types.ts,MembersPage.tsx}`,
  `features/members/components/MemberSheets.tsx`,
  `features/members/hooks/useMemberActions.ts`,
  `features/members/memberFormSchema.ts`,
  `features/events/components/EventDetailSheet.tsx`,
  `features/settings/components/ProfilePanel.tsx`, `api/map.ts`,
  `services/serviceLayerReal.ts`, `mocks/{db.ts,handlers.ts}`,
  `context/AppContext.tsx`, `services/serviceContract.test.ts`,
  `i18n/{en.ts,de.ts}`.
