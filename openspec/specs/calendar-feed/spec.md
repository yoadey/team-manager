# calendar-feed Specification

## Purpose
Defines a personal, subscribable `.ics` calendar feed per team member: a member with `events` read access can issue, rotate, and revoke a secret token bound to their identity, the feed is servable at a stable token URL without a login session, each request re-evaluates the token holder's current membership and `events` read permission so access changes take effect immediately, cancelled events are excluded, the subscriber can restrict the feed to chosen event types and toggle birthdays without re-subscribing, and the feed never leaks member emails or phone numbers.
## Requirements
### Requirement: A member can obtain a personal calendar subscription link
The system MUST let a team member with at least "read" access to the
`events` module obtain a URL they can add to an external calendar app to see
that team's events, kept up to date automatically.

#### Scenario: Issuing a feed link
- **WHEN** a member with `events` read access calls
  `POST /teams/{teamId}/calendar-feed/token`
- **THEN** a token bound to that member and team is created (or an existing
  one rotated) and a ready-to-use subscription URL is returned

#### Scenario: Re-issuing rotates the link
- **WHEN** a member who already has an active token calls
  `POST /teams/{teamId}/calendar-feed/token` again
- **THEN** the previous token stops working and a new URL is returned

### Requirement: A member can revoke their calendar feed link
The system MUST let a member invalidate their own feed link at any time,
without operator intervention.

#### Scenario: Revoking a link
- **WHEN** a member calls `DELETE /teams/{teamId}/calendar-feed/token`
- **THEN** the corresponding token stops serving the feed on any subsequent
  request

### Requirement: The feed is servable without a login session
`GET /calendar-feed/{token}.ics` MUST be reachable by an external calendar
client that cannot present this application's session cookie, and MUST
return content in the iCalendar (`text/calendar`) format.

#### Scenario: Fetching the feed with only the token
- **WHEN** a request is made to `GET /calendar-feed/{token}.ics` with a
  valid, active token and no session cookie
- **THEN** the response is `200 OK` with `Content-Type: text/calendar`
  containing that team's non-cancelled events as `VEVENT` entries

### Requirement: The feed reflects the token holder's current access
Each feed request MUST re-evaluate the token holder's current team
membership and `events` module read permission — access granted or revoked
after the token was issued MUST take effect on the very next request.

#### Scenario: Token holder still has access
- **WHEN** the feed is requested with a token whose holder is still a team
  member with `events` read access
- **THEN** the feed is served normally

#### Scenario: Token holder lost access
- **WHEN** the feed is requested with a token whose holder has since left the
  team, or whose `events` permission has since been set to "none"
- **THEN** the request is rejected as not found, without revealing whether
  the token itself is otherwise well-formed or previously valid

#### Scenario: Revoked token
- **WHEN** the feed is requested with a token that has been revoked (via
  re-issue or explicit revocation)
- **THEN** the request is rejected as not found

### Requirement: The feed excludes cancelled events
Cancelled events MUST NOT appear in the rendered feed, matching the
existing one-time `.ics` export's behavior.

#### Scenario: A cancelled event is omitted
- **WHEN** the feed is rendered for a team that has both active and
  cancelled events
- **THEN** only the active events appear as `VEVENT` entries

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

