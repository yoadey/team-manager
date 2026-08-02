## ADDED Requirements

### Requirement: Long lists are fetched and rendered in bounded pages
The transactions and members lists MUST fetch and render data in bounded
pages using the backend's existing keyset pagination, rather than
fetching and rendering the entire dataset at once.

#### Scenario: A team with years of transaction history
- **WHEN** a team's finances page is opened and the team has more
  transactions than one page holds
- **THEN** only the first page is fetched and rendered initially
- **AND** additional pages load on scroll-near-bottom or an explicit
  "load more" action, not all at once
