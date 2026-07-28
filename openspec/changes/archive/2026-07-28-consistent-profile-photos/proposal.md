## Why

Profile photos don't show up at events, and avatar rendering is inconsistent across the app. Root cause: the frontend builds a member's photo URL from `membershipId` (`frontend/src/api/map.ts` `ln(hasPhoto, '/teams/{teamId}/members/{membershipId}/photo')`), but `AttendanceRow` (`openapi.yaml` ~line 2218) exposes only `userId` + `hasPhoto`, **not** `membershipId` — so attendance rows can never construct the photo URL and always fall back to initials. Other person-rendering spots (poll voters, notifications, etc.) each roll their own avatar, so behavior differs place to place.

## What Changes

- Add `membershipId` to `AttendanceRow` so event attendance can build the photo URL like everywhere else.
- Audit **every** place a person is rendered (attendance lists, event detail, poll voters, members, notifications, comments, absences) and route them through one shared avatar component that consistently: shows the photo when `hasPhoto`, else the colored initials fallback, with the same sizing/cache-busting.

## Capabilities

### New Capabilities
- `profile-photos`: uniform, correct avatar rendering wherever a person appears.

## Impact

- Spec/backend: `openapi.yaml` `AttendanceRow` gains `membershipId`; `internal/events`/`internal/attendance` populate it; regenerate clients; tests. (Poll voter ids are handled in `poll-vote-visibility`.)
- Frontend: a single shared avatar component/util built on the `ln()` URL rule in `api/map.ts`; refactor person-rendering call sites (event attendance, poll voters, members, notifications, comments, absences) to use it; tests.
- CI: openapi-drift, backend + frontend gates. **API-affecting (additive).**
