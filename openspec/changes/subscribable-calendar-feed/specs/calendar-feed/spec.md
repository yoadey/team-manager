## ADDED Requirements

### Requirement: Subscribable auto-refreshing calendar feed
The app MUST provide a subscribable calendar feed at a stable URL that returns live `text/calendar` and reflects current events without the user re-exporting.

#### Scenario: Subscribe once, stays current
- **WHEN** a user subscribes their calendar app to the feed URL and an event is later added or changed
- **THEN** on the calendar app's next refresh the feed reflects the change without any manual re-export

#### Scenario: Importable into common calendar apps
- **WHEN** the feed URL is added in Google Calendar, Apple Calendar, or Outlook as a subscribed calendar
- **THEN** the team's events appear at the correct local times

### Requirement: Selectable feed content
The subscriber MUST be able to choose which categories the feed contains — any subset of the event types and whether birthdays are included — and the selection MUST apply to the existing subscription URL without re-subscribing. By default the feed contains all event types and birthdays.

#### Scenario: Default feed
- **WHEN** a user has not customized their feed content
- **THEN** the feed contains all event types (training, tournaments, performances, …) and birthdays

#### Scenario: Restrict to some categories
- **WHEN** a user sets their feed to include only tournaments and excludes birthdays
- **THEN** on the calendar app's next refresh the feed contains only tournament events and no birthdays, using the same subscription URL

### Requirement: Feed includes member birthdays
The calendar feed MUST include each visible team member's birthday as a yearly, all-day recurring entry.

#### Scenario: Member with a birthday
- **WHEN** a team member has a stored birthday the subscriber may see
- **THEN** the feed contains an all-day, yearly-recurring entry on that member's birthday

#### Scenario: Member without a birthday
- **WHEN** a team member has no stored birthday
- **THEN** no birthday entry is emitted for that member

### Requirement: Feed authenticated by a secret, revocable token
The feed MUST be reachable only via an unguessable token in the URL, scoped to the subscribing member's visibility, and revocable by regenerating the token.

#### Scenario: Valid token
- **WHEN** a request presents a valid feed token
- **THEN** the feed for that member's team is returned, limited to events and birthdays that member may see

#### Scenario: Revoked token
- **WHEN** a user regenerates their feed token
- **THEN** requests using the previous token are rejected and no longer return calendar data

#### Scenario: No sensitive contact data
- **WHEN** the feed is generated
- **THEN** it contains event schedule/location, titles, and birthdays only — never members' emails or phone numbers
