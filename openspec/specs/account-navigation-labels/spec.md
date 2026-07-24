# account-navigation-labels Specification

## Purpose
TBD - created by archiving change kleinere-findings. Update Purpose after archive.
## Requirements
### Requirement: The account entry point's label matches its content
The label on the sidebar's account button and the title of the sheet it opens MUST describe what that sheet actually contains, and MUST NOT reference content (such as role management) the sheet does not show.

#### Scenario: Opening the account sheet
- **WHEN** a user opens the account sheet from the sidebar
- **THEN** the button's label and the sheet's title both describe account settings only
- **AND** neither label implies role editing or role display, since the sheet contains neither

