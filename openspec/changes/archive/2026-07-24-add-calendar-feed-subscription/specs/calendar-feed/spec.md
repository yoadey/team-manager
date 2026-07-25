## ADDED Requirements

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
