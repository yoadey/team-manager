## ADDED Requirements

### Requirement: Per-group attendance threshold
A team MUST be able to store, for each of its roles, an attendance threshold
consisting of a mode — `percent` or `count` — and a non-negative integer
value. A role without a stored threshold MUST resolve to the default
`percent` / `50`. Deleting a role MUST delete its threshold.

#### Scenario: Default when nothing is configured
- **WHEN** the thresholds of a team that never configured any are requested
- **THEN** every role of the team is returned with mode `percent` and
  value `50`

#### Scenario: Storing a percentage threshold
- **WHEN** a threshold of mode `percent`, value `75` is saved for a role
- **THEN** a subsequent read returns mode `percent`, value `75` for that
  role
- **AND** the statistics overview reports that threshold on the role's
  group entries

#### Scenario: Storing an absolute threshold
- **WHEN** a threshold of mode `count`, value `8` is saved for a role
- **THEN** a subsequent read returns mode `count`, value `8` for that role

#### Scenario: Roles not mentioned keep their threshold
- **WHEN** a save request lists only one of a team's two roles
- **THEN** the other role's threshold is unchanged

#### Scenario: Role deletion removes its threshold
- **WHEN** a role with a stored threshold is deleted
- **THEN** no threshold for that role remains

### Requirement: Threshold validation
Saving thresholds MUST reject an unknown mode, a negative value, a `percent`
value above 100 and a `count` value above 10000 with 400 Bad Request, and
MUST reject a role that does not belong to the team in the URL with 422
Unprocessable Entity. A rejected request MUST NOT change any threshold.

#### Scenario: Percentage above 100
- **WHEN** a threshold of mode `percent`, value `101` is submitted
- **THEN** the request fails with 400 and nothing is stored

#### Scenario: Unknown mode
- **WHEN** a threshold with a mode other than `percent` or `count` is
  submitted
- **THEN** the request fails with 400 and nothing is stored

#### Scenario: Role of another team
- **WHEN** a save request for team A lists a role that belongs to team B
- **THEN** the request fails with 422
- **AND** neither team's thresholds change, including for valid roles
  listed in the same request

#### Scenario: Too many items
- **WHEN** a save request lists more than 200 thresholds
- **THEN** the request fails with 400 and nothing is stored

### Requirement: Threshold changes are audited
A successful threshold save MUST be recorded in the audit log with the
acting user and the team, like other team-settings mutations; a rejected
save MUST NOT be recorded as a success.

#### Scenario: Successful save
- **WHEN** a settings-authorized user saves thresholds
- **THEN** an audit record `stats.thresholds_update` with outcome `success`
  is written for that user and team

#### Scenario: Rejected save
- **WHEN** a threshold save is rejected
- **THEN** no successful `stats.thresholds_update` record is written

### Requirement: Threshold authorization
Reading the thresholds MUST require `settings: read` and saving them MUST
require `settings: write` in the team; an unauthenticated request MUST be
rejected with 401.

#### Scenario: Member without settings access
- **WHEN** a member whose roles grant `settings: none` requests or saves
  the thresholds
- **THEN** the request is rejected with 403

#### Scenario: Settings writer saves
- **WHEN** a member with `settings: write` saves thresholds
- **THEN** the request succeeds

#### Scenario: Unauthenticated
- **WHEN** the thresholds are requested without a valid session
- **THEN** the request is rejected with 401

### Requirement: Thresholds are edited in the team settings
The team settings sheet MUST list every role of the team with a control to
choose between percentage and absolute count and a numeric input for the
value, pre-filled with the stored (or default) threshold, and MUST save
changes together with the other team settings.

#### Scenario: Editing a group's threshold
- **WHEN** a settings-authorized user switches the "Spieler" role to
  "count", enters 8 and saves the team settings
- **THEN** the threshold for "Spieler" is stored as mode `count`, value `8`

#### Scenario: Untouched thresholds are not re-saved
- **WHEN** the user saves the team settings without changing any threshold
- **THEN** no threshold save request is sent

#### Scenario: Invalid input is blocked in the form
- **WHEN** the user enters 150 for a role in percentage mode
- **THEN** the form shows a validation error and does not submit

#### Scenario: Thresholds fail to save after the team settings were saved
- **WHEN** the team settings are stored but the threshold save is rejected
- **THEN** the user is told that the team settings were saved and the
  thresholds were not
- **AND** the team name and other settings shown in the app reflect the
  saved values
- **AND** the threshold list is reloaded from the server
