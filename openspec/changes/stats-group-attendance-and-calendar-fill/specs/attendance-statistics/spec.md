## ADDED Requirements

### Requirement: Per-event attendance broken down by group
The statistics overview MUST report, for every event it lists, one entry per
group (a team role held by at least one current member) with that group's
`yes` and `counted` numbers and its resolved attendance threshold. The
numbers MUST use the same roster and effective-attendance counting as the
event's own `yes`/`nominated` figures, and a member holding several roles
MUST be counted in each of those groups.

#### Scenario: Group fully present
- **WHEN** an event in range has two members with the role "Trainer", both
  with an effective status of "yes"
- **THEN** the event's "Trainer" group reports `yes = 2` and `counted = 2`

#### Scenario: Group partly present
- **WHEN** an event in range has four members with the role "Spieler", of
  whom three are effectively "yes" and one is "no"
- **THEN** the event's "Spieler" group reports `yes = 3` and `counted = 4`

#### Scenario: Member with two roles counts in both groups
- **WHEN** a member holding both "Trainer" and "Mitglied" attended an event
- **THEN** that member is included in the `yes` and `counted` numbers of
  both the "Trainer" and the "Mitglied" group for that event

#### Scenario: Pending responses are not counted
- **WHEN** every member of a group has no explicit response to an opt-in
  event and no covering absence
- **THEN** that group reports `counted = 0` for the event
- **AND** the group is not presented as complete

#### Scenario: Group figures reconcile with the event row
- **WHEN** every current member of a team holds the role "Mitglied"
- **THEN** each event's "Mitglied" group `yes`/`counted` equal that event's
  own `yes`/`nominated`

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
mark when the group is complete (`counted > 0` and `yes = counted`), a "no
data" dash when `counted = 0`, otherwise the attendance percentage. An
incomplete group below its threshold MUST be marked red, and an incomplete
group MUST never be displayed as 100 %.

#### Scenario: Complete group shows a check mark
- **WHEN** a group has `yes = 5` and `counted = 5` for an event
- **THEN** its indicator is a check mark, not a percentage

#### Scenario: Incomplete group below a percent threshold is red
- **WHEN** a group has `yes = 2`, `counted = 5` and a threshold of 50 percent
- **THEN** its indicator reads "40%" and is marked red

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

#### Scenario: Group without counted responses
- **WHEN** a group has `counted = 0` for an event
- **THEN** its indicator is a dash and is not marked red

#### Scenario: Event without any counted response
- **WHEN** an event has no counted response at all (`nominated = 0`)
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
