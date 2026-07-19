## ADDED Requirements

### Requirement: Event comment listing uses keyset pagination
Listing an event's comments MUST use keyset (cursor) pagination and return an
opaque `{items, nextCursor}` envelope, ordered oldest-first, so pagination cost
does not grow with an event's comment count and the endpoint matches every other
list in the API.

#### Scenario: Paging a busy event
- **WHEN** an event has more comments than a single page
- **THEN** the first page returns the oldest comments plus a `nextCursor`
- **AND** following `nextCursor` returns the next comments in chronological order
- **AND** no comment is skipped or repeated across pages

#### Scenario: Last page
- **WHEN** the final page of comments is returned
- **THEN** `nextCursor` is null
