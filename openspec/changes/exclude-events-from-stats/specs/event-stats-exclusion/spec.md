## ADDED Requirements

### Requirement: Per-event statistics exclusion flag
An event MUST support an `excludeFromStats` flag, settable at creation and
edit time, defaulting to `false`. The flag has no effect on the event's
visibility, RSVP, comments, notifications, or cancellation behavior — it
only affects whether the event is considered by attendance statistics.

#### Scenario: Creating an excluded event
- **WHEN** a user with `events:write` creates an event with
  `excludeFromStats: true`
- **THEN** the event is created normally and appears in the event list,
  supports RSVP and comments like any other event

#### Scenario: Default is unchanged behavior
- **WHEN** an event is created without specifying `excludeFromStats`
- **THEN** it defaults to `false` and is included in statistics as before

### Requirement: Series seeding with per-occurrence override
Setting `excludeFromStats` on a recurring series' template MUST seed the
flag onto every occurrence generated for that series. Editing an existing
series with scope "series" MUST update the template and every future
(`date >= today`) occurrence. Editing a single occurrence with scope
"single" MUST change only that occurrence, independent of the series
template and other occurrences.

#### Scenario: Creating a series with the flag set
- **WHEN** a user creates a recurring series with `excludeFromStats: true`
- **THEN** every generated occurrence has `excludeFromStats: true`

#### Scenario: Series-scoped edit updates future occurrences
- **WHEN** a user edits an existing series' `excludeFromStats` with scope
  "series"
- **THEN** the series template and every occurrence dated today or later
  are updated
- **AND** occurrences dated before today are unchanged

#### Scenario: Single-occurrence override
- **WHEN** a user edits one occurrence of a series with scope "single" to
  flip `excludeFromStats`
- **THEN** only that occurrence's flag changes
- **AND** the series template and all other occurrences are unaffected

### Requirement: Excluded events are omitted from all statistics
Every attendance-statistics computation (overview quotes, per-event stats,
absence stats, single-member stats, the attendance matrix) MUST exclude
events flagged `excludeFromStats`, as if they did not exist for statistics
purposes, while such events remain fully visible and functional everywhere
else in the application.

#### Scenario: Excluded event omitted from personal quote
- **WHEN** a member's attendance quote is computed for a date range
  containing an excluded event
- **THEN** that event does not contribute to the quote's numerator or
  denominator

#### Scenario: Excluded event omitted from the attendance matrix
- **WHEN** the attendance matrix is requested for a date range containing
  an excluded event
- **THEN** the matrix has no column for that event

#### Scenario: Excluded event still shows its own attendance summary
- **WHEN** a user views an excluded event's own detail page
- **THEN** its attendance summary (who responded yes/no/maybe) is shown
  exactly as for any non-excluded event
