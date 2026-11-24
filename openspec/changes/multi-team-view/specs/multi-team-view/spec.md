## ADDED Requirements

### Requirement: Selecting several teams at once
A user who is a member of more than one team MUST be able to select any
non-empty subset of their teams, or "All teams", as the current view.
Selecting a single team MUST behave exactly like the previous single-team
switch.

#### Scenario: Selecting two teams
- **WHEN** a member of teams A, B and C selects A and B in the team switcher
- **THEN** the view shows data of A and B

#### Scenario: Single team selected
- **WHEN** a user selects only team A
- **THEN** every screen behaves and looks as it did before this change, with
  no team badges, team pickers or filter chips

#### Scenario: "All teams" follows membership changes
- **WHEN** a user has selected "All teams" and is then added to team D
- **THEN** team D is included in the view without the user changing the
  selection

#### Scenario: Selection only contains own teams
- **WHEN** a stored or linked selection contains a team the user is no longer
  a member of
- **THEN** that team is ignored and the remaining teams are shown

### Requirement: No team is privileged when several are selected
While more than one team is selected, no selected team MUST act as a home,
primary or default team. Anything that needs a single team MUST take it from
the item being acted on or from an explicit choice by the user, never from an
implicit default.

#### Scenario: No implicit default team
- **WHEN** teams A and B are selected
- **THEN** no screen, form or action silently assumes A or B as "the" team

#### Scenario: Narrowing back to one team
- **WHEN** a user with teams A and B selected deselects B
- **THEN** the app is in single-team mode for team A, exactly as before this
  change

### Requirement: Selection is persisted and shareable
The selection MUST survive a page reload and MUST be reflected in the URL of
top-level routes. A URL without a selection MUST keep its previous meaning.

#### Scenario: Reload keeps selection
- **WHEN** a user with teams A and B selected reloads the page
- **THEN** teams A and B are still selected

#### Scenario: Old link without selection
- **WHEN** a user opens a link created before this change (no `teams`
  parameter)
- **THEN** the app opens in single-team mode on the stored or default team

### Requirement: Aggregated modules show the union of the selected teams
Home, the event list, the event calendar, the absences tab, the member list,
news, polls and notifications MUST show the items of every selected team for
which the caller has at least read permission on the module, each item
visibly marked with its team.

#### Scenario: Events of two teams in one list
- **WHEN** teams A and B are selected and both have upcoming events
- **THEN** the event list shows the events of A and B in one chronological
  list, each with a badge naming its team

#### Scenario: Team without read permission is left out
- **WHEN** teams A and B are selected and the caller's permission on `events`
  in team B is `none`
- **THEN** the event list shows only team A's events and no request for team
  B's events is made

#### Scenario: Calendar distinguishes teams
- **WHEN** the calendar shows events of teams A and B
- **THEN** each entry is colored by its team and a legend lets the user hide
  and show each team's entries without changing the selection

### Requirement: Finances, statistics and team administration are not merged
Finances, statistics, the team page and the team-scoped parts of settings
MUST always operate on exactly one team and MUST NOT combine data of several
teams. In multi-team mode they MUST offer a picker limited to selected teams
for which the caller has the screen's read permission. When more than one
team qualifies and the user has not yet picked one on that screen, the screen
MUST ask which team to open instead of choosing one itself.

#### Scenario: Finances in multi-team mode
- **WHEN** teams A and B are selected, the caller may read finances in both,
  and opens finances for the first time
- **THEN** the screen asks whether to open A or B, and after the choice shows
  only that team's finances with a picker to switch, and no totals combine
  both teams

#### Scenario: Only one team qualifies
- **WHEN** teams A and B are selected and the caller may read finances only
  in A
- **THEN** finances of A are shown directly without asking

#### Scenario: Screen remembers its own choice
- **WHEN** the user picked B on the finances screen, navigates away and comes
  back
- **THEN** finances of B are shown again, and no other screen or form treats B
  as a default

#### Scenario: Picker does not change the selection
- **WHEN** the user switches the statistics picker from A to B
- **THEN** statistics of B are shown and the overall selection (A and B)
  stays unchanged

### Requirement: Actions use the item's own team
Every action on an item shown in multi-team mode (RSVP, comment, nominate,
edit, cancel, delete, vote, open detail) MUST be sent to the team through
which the item was obtained, and its availability MUST be decided by the
caller's permission in that team — never by the permission in any other
selected team.

#### Scenario: Edit permission differs per team
- **WHEN** teams A and B are selected, the caller has `events:write` in team A
  but only `events:read` in team B, and an event of team A is shown
- **THEN** the event offers edit actions, and saving sends the request to
  team A

#### Scenario: No write permission in the item's team
- **WHEN** teams A and B are selected, the caller has `events:write` in team B
  but only `events:read` in team A
- **THEN** events of team A offer no edit, cancel or delete actions

#### Scenario: RSVP goes through the item's team
- **WHEN** teams A and B are selected and the caller RSVPs to an event
  obtained through team A
- **THEN** the RSVP request is addressed to team A

### Requirement: Items and people appearing through several teams are shown once
An event that targets several selected teams MUST appear once. A person who
is a member of several selected teams MUST appear once in the member list
with a badge for each of those teams, and per-team attributes (roles, group,
title) MUST be shown per team rather than merged. A redacted shared-calendar
event MUST be omitted when the same event is available in full through a
selected team.

#### Scenario: Cross-team event in two selected teams
- **WHEN** an event targets teams A and B and both are selected
- **THEN** the event appears once, its actions use the team in which the
  caller has the higher events permission, and its badge indicates both teams

#### Scenario: Member of two selected teams
- **WHEN** a person is a member of selected teams A (role "Trainer") and B
  (role "Spieler")
- **THEN** the member list shows that person once with badges for A and B,
  and the roles are shown as "Trainer" for A and "Spieler" for B

#### Scenario: Shared event already visible in full
- **WHEN** team C shares its calendar with team A, the caller is a member of
  A and C, and A and C are selected
- **THEN** each of team C's events appears once, in its full (non-redacted)
  form

### Requirement: Reasons stay within their team
Attendance and absence reasons MUST only be shown in the context of the team
whose visibility rules returned them and MUST NOT be displayed on another
team's representation of the same person or event.

#### Scenario: Reason visible in one team only
- **WHEN** a member's absence reason is visible to the caller in team A but
  not in team B, and both teams are selected
- **THEN** the reason is shown on team A's absence entry and not on team B's

### Requirement: Create actions choose a target team
In multi-team mode, creating an item MUST require the user to choose the
target team among the selected teams in which they have write permission on
the module. No team MUST be pre-selected unless exactly one team qualifies or
the context the form was opened from already determines the team. Creating an
event MUST NOT implicitly target several teams; targeting several teams
remains the explicit cross-team option.

#### Scenario: No pre-selected target team
- **WHEN** teams A and B are selected, the caller has `news:write` in both and
  opens the news form
- **THEN** the team field offers A and B with neither selected, and the item
  cannot be saved until a team is chosen

#### Scenario: Single qualifying team
- **WHEN** teams A and B are selected and the caller has `news:write` only in
  A
- **THEN** the news form shows A as the fixed target team

#### Scenario: Team implied by context
- **WHEN** the user opens the create form from team A's finances screen
- **THEN** team A is pre-filled as the target

#### Scenario: Only writable teams offered
- **WHEN** the caller has `events:write` in team A but not in team B
- **THEN** the event form offers only team A as target

#### Scenario: Create button visibility
- **WHEN** the caller has `polls:write` in at least one selected team
- **THEN** the create-poll action is available

### Requirement: Absence can be entered for several teams at once
In multi-team mode, a user MUST be able to record one absence period for
several selected teams in a single step, resulting in one absence per chosen
team, and MUST be told which teams failed if not all succeed.

#### Scenario: Absence for two teams
- **WHEN** the user records an absence from 1 to 7 March for teams A and B
- **THEN** one absence is created in team A and one in team B

#### Scenario: Partial failure
- **WHEN** creating the absence succeeds for team A and fails for team B
- **THEN** the user is told that team B failed and team A's absence is kept

### Requirement: One failing team does not blank the view
If loading a module's data fails for some selected teams, the view MUST still
show the data of the other teams and MUST indicate which teams could not be
loaded, with a way to retry.

#### Scenario: One team unavailable
- **WHEN** loading events succeeds for team A and fails for team B
- **THEN** team A's events are shown together with a notice that team B could
  not be loaded and a retry action
