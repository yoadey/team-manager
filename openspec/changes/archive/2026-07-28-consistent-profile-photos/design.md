## Context

Photos are gated by a `hasPhoto` boolean and fetched from a per-entity path; `api/map.ts`'s `ln(hasPhoto, path)` builds a cache-busted URL or returns null. Members use `/teams/{teamId}/members/{membershipId}/photo`. `AttendanceRow` lacks `membershipId`, so its `hasPhoto` is unusable. Different components render avatars independently, so fallbacks and sizes drift.

## Goals / Non-Goals

**Goals:**
- Photos render at events (and everywhere a person appears).
- One code path decides photo-vs-initials, sizing, and cache-busting.

**Non-Goals:**
- Changing how photos are stored/served (object storage stays as is).
- Changing the `hasPhoto` gating model.

## Decisions

- Add `membershipId` to `AttendanceRow` (backend already knows it when assembling the matrix). Keep `userId` too.
- Introduce one shared `<Avatar>` (or `avatarProps` helper) taking `{name, avatarColor, hasPhoto, photoUrl?}` and reuse the existing `ln()` rule to derive the URL; every person-rendering site uses it.
- Inventory call sites and migrate them: event attendance rows, event detail, poll voters (ids from `poll-vote-visibility`), members list, notifications, comments, absences. Where a site lacks `hasPhoto`/`membershipId`, extend that payload minimally.

## Risks / Trade-offs

- Scope discovery: an audit is needed to find all avatar renderers; the change is only "done" when they share one component — track the list in tasks.
- Additive API field is low-risk; regenerate and commit clients to keep openapi-drift green.
- Cache-busting must stay consistent so photos update after a change without breaking caching everywhere.
