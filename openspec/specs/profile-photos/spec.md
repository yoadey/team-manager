# profile-photos Specification

## Purpose
TBD - created by archiving change consistent-profile-photos. Update Purpose after archive.
## Requirements
### Requirement: Photos render for event attendance
An event's attendance list MUST show each member's profile photo when they have one.

#### Scenario: Member with a photo in attendance
- **WHEN** a member with a profile photo appears in an event's attendance list
- **THEN** their photo is shown (not just their initials)

### Requirement: Consistent avatar rendering everywhere
Every place a person is rendered MUST use the same avatar logic: photo when available and successfully loaded, otherwise the colored-initials fallback, with consistent sizing.

#### Scenario: Member without a photo
- **WHEN** a member without a photo is rendered anywhere in the app
- **THEN** the same colored-initials fallback is shown

#### Scenario: Same person across screens
- **WHEN** the same member is shown in different parts of the app (attendance, members, notifications, comments)
- **THEN** their avatar looks the same in each place

#### Scenario: Photo URL present but the image fails to load
- **WHEN** a person has a photo URL but the underlying image request fails (expired link, network error, unreachable host)
- **THEN** the colored-initials fallback is shown instead of an empty avatar

