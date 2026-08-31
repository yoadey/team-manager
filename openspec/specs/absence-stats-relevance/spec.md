# absence-stats-relevance Specification

## Purpose
Defines how a planned absence can be flagged as not relevant for attendance statistics: any member may flag their own absence, flagging a colleague's requires `events:write`, and an event date covered by such a flagged absence is excluded entirely from that member's attendance statistics — neither attending, absent, nor counted toward the total — unless an explicit response for that date overrides the flag.

## Requirements

### Requirement: Absence owner can always flag their own absence as not relevant for statistics
Any team member MUST be able to set or clear `notRelevantForStats` on their
own absence, without needing any module permission beyond team membership.

#### Scenario: Owner flags their own absence
- **WHEN** a member sets `notRelevantForStats: true` on their own absence
- **THEN** the request succeeds regardless of the member's role permissions

### Requirement: Flagging another member's absence requires events:write
Setting `notRelevantForStats` on an absence belonging to a different team
member MUST require the caller to hold `write` permission on the `events`
module for that team.

#### Scenario: Privileged caller flags a colleague's absence
- **WHEN** a caller with `events:write` sets `notRelevantForStats` on
  another member's absence
- **THEN** the request succeeds

#### Scenario: Unprivileged caller attempts to flag a colleague's absence
- **WHEN** a caller without `events:write` attempts to set
  `notRelevantForStats` on another member's absence
- **THEN** the request is rejected as forbidden
- **AND** the absence is unchanged

### Requirement: A not-relevant absence is excluded from statistics entirely
An event date covered by an absence flagged `notRelevantForStats` MUST NOT
count toward that member's attendance statistics in any way — not as
attending, not as absent, and not toward the counted total — as opposed to
a normal absence, which counts as not attending.

#### Scenario: Not-relevant absence excluded from a member's quote
- **WHEN** a member's attendance quote is computed for a range containing
  an event date covered by a `notRelevantForStats` absence, with no
  explicit response for that event
- **THEN** that event date contributes to neither the numerator nor the
  denominator of the quote

#### Scenario: Normal absence still counts as not attending
- **WHEN** a member's attendance quote is computed for a range containing
  an event date covered by an absence without `notRelevantForStats` set,
  with no explicit response for that event
- **THEN** that event date counts as not attending, unchanged from today's
  behavior

#### Scenario: Explicit response overrides the absence
- **WHEN** a member explicitly responds to an event whose date is also
  covered by a `notRelevantForStats` absence
- **THEN** the explicit response is what counts, not the absence flag
