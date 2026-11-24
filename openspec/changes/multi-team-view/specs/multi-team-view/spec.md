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

#### Scenario: Selection shrinks to one team
- **WHEN** a user with teams A and B selected is removed from team B
- **THEN** the app is in single-team mode for team A

#### Scenario: Confirming a selection keeps the route
- **WHEN** a user on the members page changes the selection from A to A and
  B, and may read members in A
- **THEN** the members page stays open and shows members of the selected
  teams

### Requirement: No team is privileged when several are selected
While more than one team is selected, no selected team MUST act as a home,
primary or default team. Anything that needs a single team MUST take it from
the item being acted on or from an explicit choice by the user, never from an
implicit default.

#### Scenario: Teams shown equally
- **WHEN** teams A and B are selected
- **THEN** the header and the team switcher show A and B equally, with
  neither marked as primary

#### Scenario: Narrowing back to one team
- **WHEN** a user with teams A and B selected deselects B
- **THEN** the app is in single-team mode for team A, exactly as before this
  change

### Requirement: Selection is persisted and shareable
The selection MUST survive a page reload, including on detail pages, and MUST
be reflected in the URL of every page while several teams are selected. A URL
without a selection MUST NOT discard a stored selection, and MUST keep its
previous meaning for a user without a stored multi-team selection.

#### Scenario: Reload keeps selection
- **WHEN** a user with teams A and B selected reloads the events page
- **THEN** teams A and B are still selected

#### Scenario: Reload on a detail page keeps selection
- **WHEN** a user with teams A and B selected reloads an open event detail
- **THEN** the same event detail is shown and teams A and B are still
  selected

#### Scenario: Old link without selection
- **WHEN** a user who never selected several teams opens a link created
  before this change
- **THEN** the app opens in single-team mode on the stored team

#### Scenario: Link without selection and a stored multi-team selection
- **WHEN** a user whose stored selection is A and B opens a link without a
  selection parameter
- **THEN** teams A and B stay selected

#### Scenario: Opening a shared selection link
- **WHEN** a member of A and C opens a link selecting teams A and B
- **THEN** only team A is shown, in single-team mode

#### Scenario: Joining via invite
- **WHEN** a user with teams A and B selected joins team D through an invite
  link
- **THEN** the app shows team D in single-team mode

### Requirement: Aggregated modules show the union of the selected teams
Home, the event list, the event calendar, the absences tab, the member list,
news, polls and notifications MUST show the items of every selected team for
which the caller has at least read permission on the module, each item
visibly marked with its team by text or icon, not by color alone.

#### Scenario: Events of two teams in one list
- **WHEN** teams A and B are selected and both have upcoming events
- **THEN** the event list shows the events of A and B in one chronological
  list, each with a badge naming its team

#### Scenario: Team without read permission is left out
- **WHEN** teams A and B are selected and the caller's permission on `events`
  in team B is `none`
- **THEN** the event list shows only team A's events and no request for team
  B's events is made

#### Scenario: Calendar distinguishes teams without relying on color
- **WHEN** the calendar shows events of teams A and B
- **THEN** each entry shows its team's icon or short name in addition to the
  team color

#### Scenario: Team filter inside a view
- **WHEN** teams A and B are selected and the user hides team B with the
  filter chips on the event list
- **THEN** only team A's events are listed and the selection still contains
  A and B

#### Scenario: Pending-RSVP badge counts each event once
- **WHEN** a cross-team event targeting selected teams A and B awaits the
  caller's response
- **THEN** it counts once in the pending-RSVP badge

### Requirement: Route reachable through any selected team
A route MUST be reachable in multi-team mode if any selected team grants the
caller its read permission, and hidden if none does.

#### Scenario: Readable in one team only
- **WHEN** teams A and B are selected and the caller may read finances only in
  team B
- **THEN** the finances route is shown in the navigation

#### Scenario: Readable in no team
- **WHEN** no selected team grants the caller `stats:read`
- **THEN** the statistics route is hidden

### Requirement: Finances, statistics and team administration are not merged
Finances, statistics, the team page (team info, roles, invites, calendar
shares, shared calendars), the roles editor, the team settings and the
calendar subscription MUST always operate on exactly one team and MUST NOT
combine data of several teams.

#### Scenario: No combined totals
- **WHEN** teams A and B are selected and the user views finances
- **THEN** every amount and total shown belongs to exactly one team

### Requirement: Single-team-only screens ask which team to open
In multi-team mode, the screens that are not merged MUST offer a picker
limited to selected teams for which the caller has the screen's read
permission. When more than one team qualifies and the user has not yet picked
one on that screen, the screen MUST ask which team to open instead of
choosing one itself.

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

#### Scenario: Calendar subscription from the merged events page
- **WHEN** teams A and B are selected and the user opens the calendar
  subscription from the events page
- **THEN** the user chooses whether to manage team A's or team B's feed

### Requirement: Per-team personal settings are shown per team
Settings that are stored per membership (profile title, push category and
reminder preferences) MUST be shown in multi-team mode as one section per
selected team, labelled with the team, each saving to its own team.

#### Scenario: Push preferences for two teams
- **WHEN** teams A and B are selected and the user opens notification
  settings with push enabled
- **THEN** one preferences section for A and one for B are shown, and
  changing A's section does not change B's preferences

### Requirement: Actions use the item's own team
Every action on a single-team item shown in multi-team mode (RSVP, comment,
edit, delete, vote, open detail, edit membership) MUST be sent to the item's
team, and its availability MUST be decided by the caller's permission in that
team — never by the permission in another selected team.

#### Scenario: Edit permission differs per team
- **WHEN** teams A and B are selected, the caller has `events:write` in team A
  but only `events:read` in team B, and an event of team A is shown
- **THEN** the event offers edit actions, and saving sends the request to
  team A

#### Scenario: No write permission in the item's team
- **WHEN** teams A and B are selected, the caller has `events:write` in team B
  but only `events:read` in team A
- **THEN** events of team A offer no edit, cancel or delete actions

#### Scenario: Poll vote goes through the poll's team
- **WHEN** teams A and B are selected and the caller votes on a poll of team B
- **THEN** the vote request is addressed to team B

### Requirement: Cross-team events appear once
An event that targets several selected teams MUST appear once, with a badge
naming the selected teams it reaches. RSVP and the caller's own status MUST
use the owning team if it is selected, otherwise the alphabetically first
selected team that reaches the event.

#### Scenario: Shown once
- **WHEN** an event of team A targets team B and both are selected
- **THEN** the event appears once, with a badge naming A and B

#### Scenario: RSVP through the owning team
- **WHEN** an event of team A targets team B, both are selected, and the
  caller RSVPs
- **THEN** the RSVP request is addressed to team A

### Requirement: Cross-team event actions follow the backend's team rules
Edit, delete, duplicate and retarget of a cross-team event MUST be offered
only when the owning team is selected and the caller has `events:write`
there, and MUST be sent through the owning team. Cancel and reactivate MUST be
offered when the caller has `events:write` in any selected team that reaches
the event.

#### Scenario: Owning team selected with write permission
- **WHEN** an event of team A targets team B, both are selected, and the
  caller has `events:write` in A
- **THEN** edit and delete are offered and are sent to team A

#### Scenario: Write only in a non-owning team
- **WHEN** an event of team A targets team B, both are selected, and the
  caller has `events:write` only in B
- **THEN** edit and delete are not offered, and cancel is offered and sent to
  team B

#### Scenario: Owning team not selected
- **WHEN** an event of team A targets teams B and C, the caller is a member of
  A, B and C, and only B and C are selected
- **THEN** edit and delete are not offered, and RSVP goes through team B

### Requirement: Cross-team event detail can be viewed through each reaching team
When several selected teams reach a cross-team event, the detail view MUST
let the user switch which team's view of the event and its attendance is
shown.

#### Scenario: Switching the attendance view
- **WHEN** a cross-team event reaches selected teams A and B and the user
  switches the detail view from A to B
- **THEN** the attendance is shown as read through team B

### Requirement: People and duplicates appear once
A person who is a member of several selected teams MUST appear once in the
member list with a badge for each of those teams, and per-team attributes
(roles, group, title) MUST be shown per team rather than merged. Birthdays
MUST appear once per person. A notification that exists once per targeted
team for the same event change MUST appear once, count once towards the
unread count, and be marked seen in every team that holds it.

#### Scenario: Member of two selected teams
- **WHEN** a person is a member of selected teams A (role "Trainer") and B
  (role "Spieler")
- **THEN** the member list shows that person once with badges for A and B,
  and the roles are shown as "Trainer" for A and "Spieler" for B

#### Scenario: Member detail switcher
- **WHEN** the user opens the detail of a person who is in selected teams A
  and B
- **THEN** the detail shows one membership and lets the user switch to the
  other team's membership

#### Scenario: Birthday once
- **WHEN** a person in selected teams A and B has a birthday in the shown
  month
- **THEN** the calendar shows the birthday once

#### Scenario: Cross-team notification once
- **WHEN** an event of team A targeting team B is created and both are
  selected
- **THEN** the notification feed shows one "event created" entry and the
  unread count increases by one

#### Scenario: Marking a merged notification seen
- **WHEN** the user marks the merged notification of the previous scenario as
  seen
- **THEN** it is marked seen in team A and in team B

### Requirement: Attendance reasons stay within their team
Attendance reasons MUST only be shown in the context of the team whose
visibility rules returned them and MUST NOT be displayed on another team's
representation of the same person or event.

#### Scenario: Reason visible in one team's view only
- **WHEN** a cross-team event reaches selected teams A and B, and an
  attendee's decline reason is visible to the caller through A but not
  through B
- **THEN** the reason is shown in the detail viewed as team A and not when
  viewed as team B

### Requirement: Create actions choose a target team
In multi-team mode, creating an item MUST require the user to choose the
target team among the selected teams in which they have write permission on
the module. No team MUST be pre-selected unless exactly one team qualifies or
the context the form was opened from determines the team. Team-dependent
fields MUST follow the chosen team. Creating an event MUST NOT implicitly
target several teams; targeting several teams remains the explicit cross-team
option.

#### Scenario: No pre-selected target team
- **WHEN** teams A and B are selected, the caller has `news:write` in both and
  opens the news form
- **THEN** the team field offers A and B with neither selected, and the item
  cannot be saved until a team is chosen

#### Scenario: Single qualifying team
- **WHEN** teams A and B are selected and the caller has `news:write` only in
  A
- **THEN** the news form shows A as the fixed target team

#### Scenario: Calendar quick-create with one team filtered
- **WHEN** teams A and B are selected, the calendar filter shows only team A,
  and the user quick-creates an event on a day
- **THEN** team A is pre-filled as the target

#### Scenario: Duplicating an event
- **WHEN** the caller duplicates an event of team A and has `events:write` in
  A
- **THEN** the form is pre-filled with team A as the target

#### Scenario: Nominations follow the chosen team
- **WHEN** the user picks team A in the event form and then changes to team B
- **THEN** the role nominations first offer A's roles with all of them
  selected, and after the change offer B's roles with all of them selected

#### Scenario: Create button visibility
- **WHEN** the caller has `polls:write` in at least one selected team
- **THEN** the create-poll action is available

### Requirement: Absence can be entered for several teams at once
In multi-team mode, a user MUST be able to record one absence period for
several selected teams in a single step, resulting in one absence per chosen
team. No team MUST be pre-selected, and before saving the user MUST be shown
that the reason will be visible to all members of each chosen team. The user
MUST be told which teams failed if not all succeed.

#### Scenario: No team pre-selected
- **WHEN** teams A and B are selected and the user opens the absence form
- **THEN** neither A nor B is chosen, and the absence cannot be saved until at
  least one team is chosen

#### Scenario: Absence for two teams
- **WHEN** the user records an absence from 1 to 7 March for teams A and B
- **THEN** one absence is created in team A and one in team B

#### Scenario: Reason visibility notice
- **WHEN** the user chooses teams A and B and enters a reason
- **THEN** the form states that the reason is visible to all members of A and
  B

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
