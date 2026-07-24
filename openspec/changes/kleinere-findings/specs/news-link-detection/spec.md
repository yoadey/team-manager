## ADDED Requirements

### Requirement: Bare URLs in news text render as clickable links
News body text MUST render any bare URL (`http://`, `https://`, or `www.`-prefixed) as a clickable link, without allowing arbitrary HTML injection from the body text.

#### Scenario: News body containing a bare https URL
- **WHEN** a news item's body contains a bare `https://` URL
- **THEN** that URL renders as a clickable link opening in a new tab
- **AND** the surrounding plain text renders unchanged

#### Scenario: News body containing a www.-prefixed URL
- **WHEN** a news item's body contains a `www.`-prefixed URL with no scheme
- **THEN** it renders as a clickable link whose destination is prefixed with `https://`
- **AND** the displayed link text is unchanged

#### Scenario: News body with no URLs
- **WHEN** a news item's body contains no URL
- **THEN** the body renders as plain text with no links
