# account-navigation-labels Specification

## Purpose
Defines that the sidebar's account entry point and the sheet it opens must be labeled to match their actual content: describing account settings only, with no wording implying role management or role display that the sheet does not contain.
## Requirements
### Requirement: The account entry point's label matches its content
The label on the sidebar's account button and the title of the sheet it opens MUST describe what that sheet actually contains, and MUST NOT reference content (such as role management) the sheet does not show.

#### Scenario: Opening the account sheet
- **WHEN** a user opens the account sheet from the sidebar
- **THEN** the button's label and the sheet's title both describe account settings only
- **AND** neither label implies role editing or role display, since the sheet contains neither

