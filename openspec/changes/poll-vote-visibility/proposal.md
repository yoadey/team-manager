## Why

For non-anonymous polls (`anonymous: false`), members want to see **who voted for what**. Today `PollOption.voters` (`openapi.yaml` ~line 2416) only carries `{name, color, hasPhoto}` — enough for a small avatar stack, but there's no per-option voter list surfaced in a readable way and no cross-tab matrix. Voters also lack a stable id, so their photos can't be rendered (see `consistent-profile-photos`).

## What Changes

- Add a **voter details popup** for non-anonymous polls with two views:
  1. **By option** — under each option, the full list of members who picked it (avatar + name).
  2. **Matrix** — a table with the options numbered `1..n` across the top and one row per user, marking which options that user selected.
- Add a stable `userId` (and `membershipId` for photo URLs) to each voter entry so avatars render and the matrix can key rows by user. Anonymous polls expose no voter identities.

## Capabilities

### New Capabilities
- `poll-visibility`: reveal per-option and per-user vote breakdowns for non-anonymous polls.

## Impact

- Spec/backend: `openapi.yaml` `PollOption.voters` gains `userId`/`membershipId`; `internal/polls` repository/service populate them **only for non-anonymous polls**, respecting `polls` read permission; regenerate clients; tests. No new endpoint (data rides the existing poll payload).
- Frontend: poll components — voter popup with the two views, matrix built from `options[].voters`, uses the shared avatar rendering; `frontend/src/i18n/{de,en}.ts`; MSW handlers.
- CI: openapi-drift, backend + frontend gates. **API-affecting (additive).**
