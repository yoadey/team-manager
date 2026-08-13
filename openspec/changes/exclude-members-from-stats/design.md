## Context

`memberships` already carries team-scoped, non-identity fields like `group`
that are edited via the same member profile form used for name/contact
details. `exclude_from_stats` follows that precedent: team-scoped (a person
excluded on one team is not automatically excluded on another team they
also belong to), and edited through the existing `MemberFormSheet`/
`memberFormSchema.ts` path rather than a new settings surface.

## Goals / Non-Goals

**Goals:**
- A flagged member's personal quota, single-member view, and matrix
  row/column disappear from statistics.
- A flagged member's historical event-level attendance (whether they said
  yes/no/maybe to a specific past event) is unaffected in per-event
  aggregates — flipping the flag never rewrites what "actually happened" at
  a given event.
- No behavior change for any existing member (column defaults to `false`).

**Non-Goals:**
- No cascading effect on the member's own ability to RSVP, comment, or use
  the app — this is a statistics-visibility flag only, not a membership
  status change.
- No bulk/"exclude several members at once" UI — each is toggled
  individually via their own profile form, matching how `group` is edited
  today. A dedicated settings list (considered and explicitly not chosen)
  can be a follow-up if profile-by-profile toggling proves inconvenient at
  scale.

## Decisions

**`EventStats` stays unfiltered; every other stats query filters on
`m.exclude_from_stats = false`.** This is the one place this feature's
semantics are not simply "delete all trace of this person from
statistics" — it is specifically "stop computing *this person's own*
statistics." `EventStats`, `MemberStats`'s matrix-adjacent roster join, and
`SingleMemberStats` differ in what they're answering:
- `MemberStats`/`SingleMemberStats`/matrix rows: "what is this member's
  attendance record" → excluded members are dropped from the roster join
  entirely.
- `EventStats`: "how many people attended this event" → an excluded
  member's actual historical response still contributes, since the number
  being reported is about the event, not about them.

This is a deliberate product decision, not a technical default — call it
out explicitly in review since a reasonable club admin might expect
"excluded" to mean invisible everywhere, including event-level counts. If
future feedback wants event-level counts to also drop excluded members
retroactively, that's a follow-up change, not something to guess at now.

**`SingleMemberStats` for an excluded member**: return the same "no data"
shape the endpoint already returns for a user with no attendance history
in range, rather than a new error code — an excluded member is not an
invalid target, their statistics are just empty by policy. Confirm the
existing endpoint shape supports an explicitly-empty result before adding
any new response variant.

## Risks

- **None database-destructive**: additive column, default `false`.
- **Product-decision risk**: the EventStats-unfiltered choice is the one
  place this feature could plausibly be implemented "wrong" relative to a
  club admin's mental model. Documented above and in the OpenSpec
  requirement text so it's an explicit, reviewable decision.
