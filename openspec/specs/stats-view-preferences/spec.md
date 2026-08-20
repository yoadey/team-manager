# stats-view-preferences Specification

## Purpose
TBD - created by archiving change stats-personal-presets. Update Purpose after archive.

## Requirements

### Requirement: Last selected statistics range is persisted per user per team
The statistics page's date-range selection (a fixed preset or a custom
from/to pair) MUST be saved per (team, user) and automatically restored the
next time that user opens the statistics page for that team, without
requiring any action.

#### Scenario: Selection survives a reload
- **WHEN** a user selects a 6-month range on the statistics page and
  reloads the page
- **THEN** the statistics page shows the 6-month range again, without the
  user reselecting it

#### Scenario: First visit uses the existing default
- **WHEN** a user opens the statistics page for a team for the first time
- **THEN** no saved selection exists yet, and the existing default (last 3
  months) is used

### Requirement: Named custom date-range presets, private per user
A user MUST be able to save a from/to date range under a chosen name,
private to themselves within a team, and see it as a selectable option
alongside the fixed presets on every future visit until they delete it.

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
