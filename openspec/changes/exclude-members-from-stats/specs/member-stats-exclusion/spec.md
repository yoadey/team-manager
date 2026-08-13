## ADDED Requirements

### Requirement: Per-member statistics exclusion flag
A membership MUST support an `excludeFromStats` flag, editable by a caller
with `members:write` via the member's profile, defaulting to `false`. The
flag has no effect on the member's ability to use the app (RSVP, comments,
etc.) — it only affects whether personal-quota statistics are computed for
them.

#### Scenario: Excluding a member
- **WHEN** a caller with `members:write` sets `excludeFromStats: true` on a
  member's profile
- **THEN** the member's profile is updated and their RSVP/app access is
  unaffected

#### Scenario: Default is unchanged behavior
- **WHEN** a membership is created without specifying `excludeFromStats`
- **THEN** it defaults to `false` and the member is included in statistics
  as before

### Requirement: Excluded members are omitted from personal-quota statistics
A member flagged `excludeFromStats` MUST be omitted from the statistics
overview's member quotas, the single-member statistics view, the attendance
matrix's rows/columns, and the absence table.

#### Scenario: Excluded member omitted from the overview
- **WHEN** the statistics overview is requested for a team with an excluded
  member
- **THEN** that member does not appear in the returned member quotas

#### Scenario: Excluded member's single-member view
- **WHEN** a single-member statistics view is requested for an excluded
  member
- **THEN** the response reflects no computed attendance statistics for them

#### Scenario: Excluded member omitted from the absence table
- **WHEN** the absence table is requested for a date range in which an
  excluded member missed an event
- **THEN** that member's row does not appear in the absence table

### Requirement: Excluded members' historical responses still count in event-level statistics
An excluded member's past explicit attendance responses MUST continue to
contribute to per-event statistics (how many people responded to a given
event), since that aggregate describes the event, not the excluded
member's personal quota.

#### Scenario: Past response counted in event turnout
- **WHEN** an excluded member had responded "yes" to a past event before
  being excluded
- **THEN** that event's per-event statistics still count their response
