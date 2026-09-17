## MODIFIED Requirements

### Requirement: Named custom date-range presets, private per user
A user MUST be able to save a from/to date range under a chosen name,
private to themselves within a team, and see it as a selectable option
alongside the fixed presets on every future visit until they delete it. A
partial update (renaming, or moving only one bound) that would leave the
preset's `to` date before its `from` date MUST be rejected with a 400 Bad
Request, never persisted and never reported as a server error.

#### Scenario: Saving a named preset
- **WHEN** a user names and saves a custom date range as "Saison 2026/27"
- **THEN** it appears as a selectable option the next time they view
  statistics for that team

#### Scenario: Presets are private
- **WHEN** a user saves a named preset
- **THEN** other team members do not see it in their own statistics view

#### Scenario: Deleting an active preset
- **WHEN** a user deletes a preset that is their currently selected range
- **THEN** their selection falls back to that preset's last-known dates
  rather than erroring or silently reverting to the default range

#### Scenario: A single-bound update that would invert the range is rejected
- **WHEN** a user has a preset with `from=2026-01-01` and `to=2026-06-01`,
  and PATCHes it with only `{"from":"2026-07-01"}` (a value past the
  stored `to`, with no `to` supplied in the same request)
- **THEN** the request fails with 400 Bad Request, the preset's stored
  range is left unchanged, and no 500 Internal Server Error is logged
