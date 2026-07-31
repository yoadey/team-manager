## ADDED Requirements

### Requirement: A member can configure which notification categories are pushed, per team
The system MUST let an authenticated team member independently enable or
disable Web Push delivery for each notification category (attendance,
events, news, polls, absence) within a specific team, without affecting any
other team they belong to.

#### Scenario: Member disables push for one category in one team
- **WHEN** a member sets `news: false` for team A via
  `PUT /teams/{teamId}/push-preferences`
- **THEN** future `news` notifications in team A are not pushed to that
  member's subscriptions
- **AND** `news` notifications in any other team the member belongs to are
  still pushed, and every other category in team A is still pushed

#### Scenario: Reading current preferences
- **WHEN** a member calls `GET /teams/{teamId}/push-preferences`
- **THEN** the response reflects their last-saved preferences for that team,
  or all categories enabled if they've never changed anything

### Requirement: Preferences default to fully enabled
A team member who has never configured push preferences MUST receive push
notifications for every category their module permissions already allow —
identical to the pre-existing, non-configurable behavior.

#### Scenario: Member has no stored preferences
- **WHEN** a notification is created for a member who has never called
  `PUT /teams/{teamId}/push-preferences` for that team
- **THEN** a push is sent for it exactly as if every category were enabled,
  subject only to the existing module-permission gate

### Requirement: The preference gate is independent of the permission gate
A push MUST be sent only when the recipient both has read access to the
notification's module (or it is self-standing) AND has not disabled that
notification's category for that team; either condition failing suppresses
the push.

#### Scenario: Permission allows but preference disables
- **WHEN** a recipient has `events:read` in a team but has disabled the
  `events` category there
- **THEN** no push is sent for an `event_created` notification in that team,
  even though the in-app feed still shows it

#### Scenario: Preference allows but permission denies
- **WHEN** a recipient has enabled the `polls` category in a team but their
  current `polls` module permission is `none`
- **THEN** no push is sent for a `poll` notification in that team, matching
  the existing permission-gate behavior
