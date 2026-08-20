## Context

`memberships` already carries one admin-only cosmetic free-text field,
`group` (`TEXT`, nullable, `maxLength: 100` in the handler), edited only via
`PATCH /teams/{teamId}/members/{membershipId}`
(`x-rbac-module: members`, no `x-rbac-self-service`) — a route that, per
`internal/middleware/authz.go`'s `hasRequiredPermission`, requires
`members:write` for *every* caller on *every* field, including a member
editing their own row. The default seeded "Member" role
(`internal/teams/repository.go`) only grants `members: read`, so a plain
member cannot use this path to change anything about themselves today —
`MemberFormSheet`'s Edit button is shown whenever `isMe` regardless of
`canWrite` (`MemberSheets.tsx`), but submitting would 403. That mismatch is
a pre-existing gap, out of scope here: this change does not touch
`UpdateMember`'s authorization at all.

Self-service member data does exist elsewhere in this codebase, just via
dedicated narrow endpoints rather than by loosening the admin path:
`POST /teams/{teamId}/members/{membershipId}/photo` variants
(`auth/me/photo`) let a member manage their own photo outside
`members:write`, and `events.Service.SetAttendance` lets a caller act on
their own attendance for free while requiring `events:write` to act on
someone else's (`if callerID != userID { requireCallerEventsWrite }`).
`title` follows the `SetAttendance` shape most closely: same endpoint,
ownership-checked, "self is free, someone else needs the module's write
permission" — except here "someone else" isn't a supported case at all
(see Decisions).

## Goals / Non-Goals

**Goals:**
- A member can set/clear a short (≤40 char), purely cosmetic label for
  themselves without needing `members:write`.
- A `members:write` holder can still set/clear any member's title (e.g. to
  moderate one), through the existing admin member-edit path.
- Title shows in small, clearly secondary text in the two places explicitly
  asked for: the member list and an event's attendance list.
- No RBAC/permission semantics attach to a title — it is cosmetic text
  only, never interpreted, never granted anything.

**Non-Goals:**
- Not fixing the pre-existing `MemberFormSheet` self-edit-without-write gap
  described above — the new self-service surface for title is a separate,
  narrow, dedicated endpoint (`PUT .../members/{membershipId}/title`), not
  a widening of `UpdateMember`.
- No profanity/content moderation beyond length — same trust level as
  `group`, `news` post bodies, or event comments, all free text a member
  can already write today.
- No title history/audit trail beyond the existing generic audit log entry
  every member-update-shaped write already gets.
- Title is not shown in every place a member's name appears (e.g. news
  authorship, poll votes, finance rows) — only the two surfaces requested,
  which are also the two places `group` already prints today.

## Decisions

**A dedicated self-service endpoint, not `x-rbac-self-service: true` on
`UpdateMember`.** Marking the whole `PATCH .../members/{membershipId}`
route self-service would let a `members:read`-only caller submit a patch
containing *any* of that endpoint's fields — name, phone, address, `group`,
`excludeFromStats` — for their own row, silently changing what "self-
service" has meant on this route since it shipped, and reopening exactly
the kind of privilege-adjacent-field question `ErrCannotChangeOthersEmail`'s
long comment already had to reason through once. A new, narrow, single-
field endpoint (`PUT .../members/{membershipId}/title`,
`SetMemberTitleRequest { title: string }`) keeps that blast radius at
"just the title," reviewable independent of the existing, already
carefully-commented `UpdateMember`/`repository.go` authorization logic.

**Self-only, not self-or-others, on the new endpoint.** Unlike
`SetAttendance` (where an organizer legitimately nominates *other* people's
attendance), there's no legitimate "set someone else's fun title for them"
action here — a title is something a member gives *themselves*. The new
endpoint rejects a caller whose `membershipId` isn't their own with
`ErrCannotSetOthersTitle` (403), mirroring `ErrCannotChangeOthersEmail`'s
shape. A `members:write` holder who does need to clear or edit someone
else's title (moderation) uses the existing admin path instead — `title` is
simply added to `UpdateMemberRequest` alongside `group`.

**Attendance list: fold into the existing secondary line, don't add a
fourth line.** `AttendanceRowItem` rows are already compact (34px avatar,
`p: '8px'`) and the line under the name is already dual-purposed: the
member's RSVP comment when present and visible to the caller, else
`group`. Title is lower-priority than both (it's decoration, not
information relevant to the event), so it's appended to the `group` branch
only (`[group, title].filter(Boolean).join(' · ')`) rather than claiming
its own line in a view already tight on vertical space. The member list has
real spare room (`p: '12px 14px'`, no line count reasoning forces the roles
subtitle up), so it gets its own dedicated line there, placed between name
and roles.

**40-character cap.** Both example titles in the request ("Orgaente",
"Witzbeauftragter") are well under 20 characters; 40 gives headroom for
slightly longer ones while keeping the field obviously not a bio or
description — enforced the same way `group`'s 100-char cap is, via
`validate.MaxLen` in the handler, plus `maxLength: 40` in the OpenAPI schema
for client-side validation.

**Nullable `TEXT` column, no default, empty string clears.** Same shape as
`group`. `SetMemberTitleRequest.title` is a required (non-optional) string
field precisely so a client can send `""` to explicitly clear a title
without a separate delete endpoint — the handler stores `NULL` when the
trimmed value is empty, `title` (trimmed) otherwise.

## Risks / Trade-offs

- **New endpoint instead of extending `UpdateMember`** costs one more
  route/handler/service method to maintain, versus one extra `if` in an
  existing one. Traded deliberately for keeping `UpdateMember`'s
  authorization semantics — already the most heavily-commented, security-
  reviewed code in this module — completely untouched.
- **Attendance-list title folded into the same line as `group`** means a
  member with both a long group and a long title could visually crowd that
  line. Accepted: both are already free text up to 100/40 chars respectively
  with no existing truncation guard on `group` there either — not a
  regression introduced by this change.
