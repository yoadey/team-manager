## ADDED Requirements

### Requirement: Detail sheet URL sync is route-independent
Opening an event or member detail sheet MUST update the browser URL to
that detail's own path (`/events/<id>` or `/members/<id>`) and create a
Back-able history entry, regardless of which top-level route it was
opened from.

#### Scenario: Opening an event from the Home route
- **WHEN** a user on the Home route clicks an upcoming-event card,
  opening its detail sheet
- **THEN** the browser URL updates to `/events/<id>` and a new history
  entry is created

#### Scenario: Opening an event from the notifications sheet
- **WHEN** a user on any route opens the notifications sheet and clicks
  an event-linked notification, opening that event's detail sheet
- **THEN** the browser URL updates to `/events/<id>` and a new history
  entry is created

#### Scenario: Back button closes a detail sheet opened from a different route
- **WHEN** a user opens an event detail sheet from Home (or from the
  notifications sheet) and then presses the browser Back button
- **THEN** the detail sheet closes and the browser returns to the page
  the user was on before opening the detail, rather than navigating past
  the app

#### Scenario: Closing a detail sheet does not leave a resurrectable history entry
- **WHEN** a user opens an event detail sheet from a route other than
  Events and then closes it via the sheet's own close control (not the
  Back button)
- **THEN** pressing Back afterwards does not reopen the sheet that was
  just closed
