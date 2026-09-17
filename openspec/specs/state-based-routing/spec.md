# state-based-routing Specification

## Purpose
Defines how the application's state-based navigation (no router dependency; `state.route` drives what renders) is mirrored into the browser URL, so that every navigation is Back-able and every view is shareable as a link. In particular, a detail sheet's path is keyed off the detail's own kind rather than the top-level route it was opened from: opening an event or member detail updates the URL to `/events/<id>` or `/members/<id>` and pushes a history entry wherever it was opened from, since opening a detail does not itself switch the top-level route.

## Requirements

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
