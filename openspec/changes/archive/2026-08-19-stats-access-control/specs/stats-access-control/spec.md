## ADDED Requirements

### Requirement: Dedicated `stats` RBAC module gates the statistics area
The statistics area (overview, attendance matrix, per-member statistics,
saved date-range presets, and the last-selected date range) MUST be gated
by its own `stats` module (`none | read | write`), independent of the
`events` module.

#### Scenario: Member without stats access cannot view statistics
- **WHEN** a caller whose effective `stats` permission is `none` requests
  any statistics endpoint
- **THEN** the request is forbidden, even if the caller holds `events: read`
  or higher

#### Scenario: Member with stats:read can view statistics
- **WHEN** a caller whose effective `stats` permission is `read` requests
  the statistics overview, attendance matrix, or per-member statistics
- **THEN** the request succeeds

### Requirement: Default Member role grants stats:read
A newly created team's default `Member` role MUST grant `read` on the
`stats` module, matching the tier already granted for `events`, `members`,
`news`, `polls`, and `settings`. The default `Admin` role MUST grant
`write`.

#### Scenario: New team creation
- **WHEN** a team is created
- **THEN** its `Admin` role has `stats: write`
- **AND** its `Member` role has `stats: read`

### Requirement: Defining a personal statistics preset requires stats:write
Creating, renaming, or deleting a saved statistics date-range preset MUST
require `write` permission on the `stats` module. Viewing the statistics
area — including the caller's existing saved presets and their
last-selected date range — MUST require only `read`. Saving the caller's
last-selected date range MUST also require only `read`, since it is an
automatic side effect of viewing rather than a deliberate act of defining
a preset.

#### Scenario: Read-only member can view but not define a preset
- **WHEN** a caller with `stats: read` (and not `write`) requests to view
  the statistics page, their saved presets, or their last-selected range
- **THEN** each request succeeds
- **WHEN** the same caller attempts to create, rename, or delete a preset
- **THEN** the request is forbidden

#### Scenario: Write holder can define a preset
- **WHEN** a caller with `stats: write` creates, renames, or deletes a
  saved date-range preset
- **THEN** the request succeeds

#### Scenario: Last-selected range is saved for a read-only member
- **WHEN** a caller with `stats: read` (and not `write`) views the
  statistics page with a given date range active
- **THEN** that range is saved as their last selection for next visit,
  without requiring `stats: write`

### Requirement: Existing teams are backfilled with stats access
When the `stats` module is introduced, every existing team's system
`Admin` role MUST be backfilled to `stats: write` and system `Member` role
to `stats: read`, so access does not silently change for teams that
haven't touched their role configuration. Any other existing role MUST be
backfilled to `stats: none`.

#### Scenario: Existing Admin role backfilled
- **WHEN** the backfill runs against a team whose system `Admin` role
  predates the `stats` module
- **THEN** that role's permissions include `stats: write` afterwards

#### Scenario: Existing Member role backfilled
- **WHEN** the backfill runs against a team whose system `Member` role
  predates the `stats` module
- **THEN** that role's permissions include `stats: read` afterwards

#### Scenario: Existing custom role backfilled to none
- **WHEN** the backfill runs against a non-system custom role that
  predates the `stats` module
- **THEN** that role's permissions include `stats: none` afterwards
- **AND** an admin can subsequently raise it via the existing role editor
