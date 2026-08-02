## Context

`Poll` has `anonymous`, `options[]`, `totalVotes`, `myVote`. `PollOption.voters` already lists voters for (non-anonymous) polls but without identifiers. The team member list is already available client-side. Anonymity must be preserved: an anonymous poll must never leak who voted.

## Goals / Non-Goals

**Goals:**
- Readable per-option voter lists and a user×option matrix for non-anonymous polls.
- Voter avatars render consistently with the rest of the app.

**Non-Goals:**
- Changing vote semantics or multi-select rules.
- Revealing any identity for anonymous polls.

## Decisions

- Extend voter objects with `userId` (+ `membershipId` for the photo URL) server-side, populated **only** when `anonymous == false`. For anonymous polls, `voters` stays identity-free (counts/percentages only).
- No new endpoint: the matrix is derived on the client from `options[].voters`. Rows = union of voters across options (optionally all members, showing non-voters); columns = options in order, numbered `1..n`.
- Popup has a tab/toggle between "by option" and "matrix"; both use the shared avatar component so photos are consistent (depends on `consistent-profile-photos` for a stable photo URL from `membershipId`).
- Enforce `polls` read permission for the details (same gate as viewing the poll).

## Risks / Trade-offs

- Privacy is the core risk: the server, not the client, must decide whether identities are included — gate strictly on `anonymous == false`. The client never receives identities for anonymous polls.
- Matrix width for 4 options is fine; rows scale with team size — virtualize only if needed (teams are small).
- Depends on `consistent-profile-photos` to render voter avatars; can ship the lists with initials first and photos once that lands.
