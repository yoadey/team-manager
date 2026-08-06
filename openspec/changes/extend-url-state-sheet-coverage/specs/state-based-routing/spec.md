## ADDED Requirements

### Requirement: Page-level destination sheets are URL-addressable
A page-level sheet that represents a navigable destination (not a
transient create/edit form) MUST be reflected in the URL, so it survives
a page refresh and can be shared as a link.

#### Scenario: Refresh while Team Settings is open
- **WHEN** a user opens Team Settings and refreshes the page
- **THEN** Team Settings reopens automatically from the URL

#### Scenario: Refresh while editing a specific role
- **WHEN** a user opens the role-edit sheet for a specific role and
  refreshes the page
- **THEN** the role-edit sheet reopens for the same role

### Requirement: Transient create/edit forms are excluded from URL coverage
A page-level sheet holding unsubmitted local draft state (event/member
create or edit forms) MUST be excluded from URL coverage, since a URL
cannot reconstruct in-progress form input.

#### Scenario: Refresh while creating a new event
- **WHEN** a user has the event-creation form open and refreshes the
  page
- **THEN** the form does not reopen, and this is documented as
  intentional rather than a gap
