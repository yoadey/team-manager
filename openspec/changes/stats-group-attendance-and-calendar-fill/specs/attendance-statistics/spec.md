## ADDED Requirements

### Requirement: Per-event attendance broken down by group
The statistics overview MUST report, for every event it lists, one entry per
group (a team role held by at least one current member) with that group's
`yes`, `counted` and `pending` numbers and its resolved attendance
threshold. The numbers MUST use the same roster and effective-attendance
status as the event's own figures, and a member holding several roles MUST
be counted in each of those groups.

#### Scenario: Group fully present
- **WHEN** an event in range has two members with the role "Trainer", both
  with an effective status of "yes"
- **THEN** the event's "Trainer" group reports `yes = 2`, `counted = 2` and
  `pending = 0`

#### Scenario: Group partly present
- **WHEN** an event in range has four members with the role "Spieler", of
  whom three are effectively "yes" and one is "no"
- **THEN** the event's "Spieler" group reports `yes = 3` and `counted = 4`

#### Scenario: Member with two roles counts in both groups
- **WHEN** a member holding both "Trainer" and "Mitglied" attended an event
- **THEN** that member is included in the `yes` and `counted` numbers of
  both the "Trainer" and the "Mitglied" group for that event

#### Scenario: Missing responses are reported separately
- **WHEN** a member of a group has no explicit response to an opt-in event
  and no covering absence
- **THEN** that member is included in the group's `pending` number and not
  in its `counted` number

#### Scenario: Not-nominated and stats-excluded members are not expected
- **WHEN** a member of a group is marked not nominated for an event, or is
  covered by an absence flagged not relevant for statistics
- **THEN** that member is in neither the group's `counted` nor its
  `pending` number

#### Scenario: Group figures reconcile with the event row
- **WHEN** every current member of a team holds the role "Mitglied"
- **THEN** each event's "Mitglied" group `yes`/`counted`/`pending` equal
  that event's own `yes`/`nominated`/`pending`

### Requirement: Missing responses count as not present in the line-up
The per-event line-up MUST treat a member who gave no response as not
present: the event's "x of y attended" line and every group indicator MUST
be measured against everyone expected (`counted + pending`), not only
against those who responded. The event's `nominated`, `pct` and `enough`
fields and the per-member quotas MUST keep counting responses only.

#### Scenario: Event line counts everyone expected
- **WHEN** an event has three members who attended and twelve who gave no
  response
- **THEN** the events tab shows "3/15 attended" for it

#### Scenario: Quotas are unaffected
- **WHEN** a member gave no response to an opt-in event
- **THEN** that event still does not count towards the member's personal
  attendance quota

### Requirement: Events tab on the statistics page
The statistics page MUST present the per-event line-up in its own tab titled
"Termine", next to the "Quote" and "Matrix" tabs, and no longer below the
per-person quotas.

#### Scenario: Opening the events tab
- **WHEN** a user with `stats: read` opens the statistics page and selects
  the "Termine" tab
- **THEN** the events of the selected date range are listed, each with its
  date, its "x of y attended" line and one indicator per group

#### Scenario: Quota tab no longer lists events
- **WHEN** the "Quote" tab is shown
- **THEN** it contains the team average and the per-person quotas only

### Requirement: Group indicator per event
Each event row in the events tab MUST show one indicator per group: a check
mark when the group is complete (everyone expected attended: `yes = counted
+ pending`, with at least one member expected), a "no data" dash when nobody
of the group was expected, otherwise the share of the group that attended
(`yes / (counted + pending)`). An incomplete group below its threshold MUST
be marked red and carry a warning icon (never colour alone), and an
incomplete group MUST never be displayed as 100 %.

#### Scenario: Complete group shows a check mark
- **WHEN** a group has `yes = 5`, `counted = 5` and `pending = 0` for an
  event
- **THEN** its indicator is a check mark, not a percentage

#### Scenario: Members without a response prevent a check mark
- **WHEN** a group has `yes = 3`, `counted = 3` and `pending = 12`
- **THEN** its indicator reads "20%", not a check mark

#### Scenario: Absolute threshold with few responses
- **WHEN** a group has `yes = 3`, `counted = 3`, `pending = 12` and a
  threshold of 8 attendees (mode "count")
- **THEN** its indicator reads "20%" and is marked red

#### Scenario: Nobody of the group responded
- **WHEN** a group has `counted = 0` and `pending = 5` and a threshold of 50
  percent
- **THEN** its indicator reads "0%" and is marked red

#### Scenario: Incomplete group below a percent threshold is red
- **WHEN** a group has `yes = 2`, `counted = 5` and a threshold of 50 percent
- **THEN** its indicator reads "40%", is marked red and shows a warning icon

#### Scenario: Incomplete group at or above a percent threshold is not red
- **WHEN** a group has `yes = 3`, `counted = 5` and a threshold of 50 percent
- **THEN** its indicator reads "60%" and is not marked red

#### Scenario: Incomplete group below an absolute threshold is red
- **WHEN** a group has `yes = 6`, `counted = 10` and a threshold of 8
  attendees (mode "count")
- **THEN** its indicator reads "60%" and is marked red

#### Scenario: Complete group is never red
- **WHEN** a group has `yes = 2`, `counted = 2` and a threshold of 3
  attendees (mode "count")
- **THEN** its indicator is a check mark and is not marked red

#### Scenario: Nobody of the group was expected
- **WHEN** a group has `counted = 0` and `pending = 0` for an event
- **THEN** its indicator is a dash and is not marked red

#### Scenario: Event nobody was expected at
- **WHEN** an event has nobody expected at all (`nominated = 0` and
  `pending = 0`)
- **THEN** its row shows no group indicators, only its "0 of 0 attended"
  line

#### Scenario: Nearly complete group is not shown as 100 %
- **WHEN** a group has `yes = 199` and `counted = 200`
- **THEN** its indicator reads "99%", not "100%"

### Requirement: Matrix shows the attendance count next to the member name
The attendance matrix MUST render each member's attendance count ("yes"
total) in the column directly after the member-name column, before the
first event column.

#### Scenario: Count column position
- **WHEN** the matrix is rendered with at least one event column
- **THEN** the second column of every row is that member's attendance count
- **AND** the event columns follow it in chronological order

#### Scenario: Count stays visible while scrolling on desktop
- **WHEN** the matrix is scrolled horizontally on a desktop viewport
- **THEN** the member name and the attendance count both stay pinned at the
  left edge

#### Scenario: Only the name is pinned on a phone
- **WHEN** the matrix is scrolled horizontally on a mobile viewport
- **THEN** the member name stays pinned and the attendance count scrolls
  with the event columns
