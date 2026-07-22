## ADDED Requirements

### Requirement: Onboarding is self-service
A newly invited member MUST be able to join and start using their team
without asking a developer or admin for help beyond the invite link
itself.

#### Scenario: New member follows the getting-started guide
- **WHEN** a newly invited member reads `docs/end-user/erste-schritte.md`
- **THEN** they can locate their invite link/code, complete first login,
  understand the "kein Team" state if they land on it, and reach a
  working team view without external assistance

### Requirement: Roles and permissions are explained in plain language
The roles/permissions model (`none`/`read`/`write` per module, and how
multiple roles combine) MUST be documented somewhere a non-technical
admin can read, since no such explanation exists anywhere in the
product UI today.

#### Scenario: Admin looks up what a permission level means
- **WHEN** a team admin reads `docs/end-user/rollen-und-rechte.md`
  before assigning or editing a role
- **THEN** they can correctly explain the difference between `none`,
  `read`, and `write` for a module, and what happens when a member
  holds multiple roles, without consulting a developer

### Requirement: Every top-level route has a corresponding doc chapter
Each of the app's top-level routes (events, members, finances, stats,
news, polls, team) MUST have a matching chapter under
`docs/end-user/`, reachable from the chapter index.

#### Scenario: A member wants help with a specific screen
- **WHEN** a member is on any top-level app route and opens
  `docs/end-user/README.md`
- **THEN** they find a chapter covering that route

### Requirement: GDPR self-service is documented for end users
The data export and account deletion flows MUST have a plain-language
companion document, separate from the implementer-facing
`docs/gdpr-data-subject-rights.md`.

#### Scenario: A member wants to export or delete their data
- **WHEN** a member reads `docs/end-user/daten-und-datenschutz.md`
- **THEN** they can find and follow the steps to export their data or
  delete their account, including what account deletion actually does
  (anonymization, not removal of shared team data), in non-technical
  language

### Requirement: Documentation is published and discoverable
The end-user documentation MUST be reachable both as browsable
Markdown in the repository and as a built static site, and MUST be
linked from the project's front door.

#### Scenario: A visitor looks for user documentation from the README
- **WHEN** someone reads the root `README.md`
- **THEN** they find a section linking to `docs/end-user/README.md`

#### Scenario: The documentation site builds
- **WHEN** `npm run build` is run inside `website/`
- **THEN** the build succeeds and produces a static site rendering
  every chapter under `docs/end-user/`
