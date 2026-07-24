## ADDED Requirements

### Requirement: Default UI copy does not presuppose one sport
Default, non-user-authored UI strings (event-type labels, form-field placeholders) that every team sees regardless of what sport they practice MUST NOT imply a specific sport such as dance/formation.

#### Scenario: Viewing the event-type selector
- **WHEN** a user views the event-type options in the event create/edit form
- **THEN** the labels are phrased generically (e.g. competition-oriented), not tied to a single sport

#### Scenario: Viewing the event title field's placeholder
- **WHEN** a user views the empty title field in the event create/edit form
- **THEN** the placeholder example does not name a sport-specific term (e.g. a dance formation)
