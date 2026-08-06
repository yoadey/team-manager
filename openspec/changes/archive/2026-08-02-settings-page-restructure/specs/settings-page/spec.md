## ADDED Requirements

### Requirement: Settings is a dedicated, category-navigated page
The application MUST provide a `/settings` route showing the caller's personal account settings grouped into categories with persistent navigation, reachable from the existing account entry points (desktop sidebar account row, mobile header avatar button) without adding a new item to the main navigation.

#### Scenario: Desktop sidebar and content pane
- **WHEN** an authenticated user is on a desktop viewport and opens Settings
- **THEN** a category sidebar and a content pane for the selected category are both shown at once

#### Scenario: Mobile category list and detail
- **WHEN** an authenticated user is on a mobile viewport and opens Settings
- **THEN** a category list is shown first; tapping a category shows only that category's content with a back affordance to return to the list

#### Scenario: Existing entry points unchanged in position, new destination
- **WHEN** the user taps the mobile header avatar button or the desktop sidebar account row
- **THEN** the app navigates to the Settings route (not the old profile sheet), and neither button's position or appearance changes

#### Scenario: No new main-navigation item
- **WHEN** the desktop nav rail or mobile bottom navigation is rendered
- **THEN** neither includes a "Settings" entry — the mobile "Mehr" overflow sheet is the only additional discovery path

### Requirement: All existing account settings are preserved by category
Every setting previously available in the flat "Mein Konto" sheet MUST remain reachable and functionally unchanged, organized as: Profil (avatar/photo, name, email), Darstellung & Sprache (color scheme, language), Benachrichtigungen (Web Push toggle, per-team push-category preferences), Datenschutz (data export, account deletion with email confirmation), Rechtliches (legal links), plus a pinned Logout action outside the categories.

#### Scenario: Web Push settings unchanged
- **WHEN** the browser supports Web Push and the user opens the Benachrichtigungen category
- **THEN** the push toggle and, once subscribed, the per-team category preference toggles behave exactly as they did in the old sheet

#### Scenario: Account deletion flow unchanged
- **WHEN** the user opens Datenschutz and starts account deletion
- **THEN** the confirm-by-retyping-email gating and 401-triggers-logout behavior are unchanged

#### Scenario: Logout remains a single tap away
- **WHEN** the user opens Settings
- **THEN** a Logout action is visible without entering any category

### Requirement: Settings route is not RBAC-gated
Since Settings shows only the caller's own personal data (not team-scoped data), it MUST be reachable regardless of the caller's module permissions.

#### Scenario: No module permission required
- **WHEN** an authenticated user with no write/read permission on any RBAC module navigates to `/settings`
- **THEN** the Settings page still renders normally
