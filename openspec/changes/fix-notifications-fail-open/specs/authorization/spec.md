## ADDED Requirements

### Requirement: Notification module classification fails closed
A notification whose module cannot be classified as one of the known
`events|members|finances|news|polls|settings` modules MUST be treated as
requiring the most restrictive access (denied), never as an unrestricted
or public item.

#### Scenario: Notification of an unrecognized type
- **WHEN** a notification row has a type not mapped to a known module
  (e.g. written by a newer server version an older reader doesn't
  recognize yet)
- **THEN** the notification is excluded from a member's activity feed
  unless the member's permissions would grant access under fail-closed
  evaluation
- **AND** it is never shown to a member solely because its module is
  unclassified
