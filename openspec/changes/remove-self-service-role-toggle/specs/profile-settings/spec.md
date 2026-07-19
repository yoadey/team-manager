## ADDED Requirements

### Requirement: Team-role assignment is not self-service from Profile Settings
The Profile Settings sheet (opened from the account/avatar entry point) MUST NOT let a user change which team roles they themselves hold. Role assignment, including for the signed-in user's own membership, MUST be done exclusively through Members management.

#### Scenario: Profile Settings shows no editable role list
- **WHEN** a user opens Profile Settings
- **THEN** no control in that sheet lets them add or remove one of their own team roles
- **AND** their current roles remain visible read-only elsewhere (e.g. the team switcher, team overview)

#### Scenario: Changing your own roles still works via Members management
- **WHEN** a user with `settings:write` opens their own entry in Members management
- **THEN** they can edit their roles there, through the same role-assignment control used for any other member
