## ADDED Requirements

### Requirement: Public legal notice (Impressum) page
The system MUST provide a legal-notice page reachable without authentication, disclosing the operator's identity, address, and contact details per `§5 DDG`.

#### Scenario: Legal notice reachable without a session
- **WHEN** a visitor with no active session navigates to the legal-notice page
- **THEN** the page renders the operator's legal-notice content
- **AND** no login or team membership is required

#### Scenario: Placeholder content is visibly unfilled
- **WHEN** the app is built and deployed without editing the shipped legal-notice content
- **THEN** the rendered page shows explicit, human-readable placeholder markers (not fabricated real-looking operator details) so an unconfigured deployment is obviously incomplete rather than silently non-compliant

### Requirement: Public privacy policy (Datenschutzerklärung) page
The system MUST provide a privacy-policy page, reachable without authentication, disclosing what personal data is processed, the purposes and legal basis, retention, and known processors, consistent with what `docs/gdpr-data-subject-rights.md` and `SECURITY.md` already document as implemented.

#### Scenario: Privacy policy reachable without a session
- **WHEN** a visitor with no active session navigates to the privacy-policy page
- **THEN** the page renders the operator's privacy-policy content
- **AND** no login or team membership is required

### Requirement: Legal pages are reachable from every screen
A footer exposing links to both the legal-notice and privacy-policy pages MUST be visible on the login screen, the self-registration screen, and throughout the authenticated app shell, in both the compact/mobile and desktop layouts.

#### Scenario: Footer visible before login
- **WHEN** the app loads to the login screen (no session)
- **THEN** the footer with legal-notice and privacy-policy links is visible without further navigation

#### Scenario: Footer visible in the compact mobile layout
- **WHEN** the app shell renders below the compact-layout breakpoint
- **THEN** the legal-notice and privacy-policy links remain reachable (not dropped from the compact layout)

### Requirement: Self-registration discloses the privacy policy and gates on a minimum-age confirmation
The self-registration form (`POST /auth/register`) MUST link the privacy-policy page next to the data it collects, and MUST require an explicit, unchecked-by-default confirmation that the registrant is at least 16 years old before the submit action is enabled.

#### Scenario: Registration form links the privacy policy
- **WHEN** the self-registration form is displayed
- **THEN** a link to the privacy-policy page is shown alongside the email/password fields

#### Scenario: Submit is blocked without the age confirmation
- **WHEN** a visitor fills in email and password but leaves the minimum-age checkbox unchecked
- **THEN** the submit control remains disabled and no `POST /auth/register` request is sent

#### Scenario: Submit proceeds once the age confirmation is checked
- **WHEN** a visitor fills in email and password and checks the minimum-age checkbox
- **THEN** the submit control is enabled and registration proceeds as today

### Requirement: Optional third-party monitoring integration has a documented consent/necessity determination
Whether the optional Sentry browser integration requires `§25 TDDDG` consent MUST be verified by inspecting its actual browser storage behavior, and the outcome MUST be recorded — either as a documented "strictly necessary, disclosure only" determination, or implemented as a consent gate before `Sentry.init` is called.

#### Scenario: Sentry disabled by default
- **WHEN** `VITE_SENTRY_DSN` is unset
- **THEN** no third-party monitoring script initializes and no consent mechanism is shown

#### Scenario: Sentry enabled with a storage determination on record
- **WHEN** `VITE_SENTRY_DSN` is set
- **THEN** the deployment's behavior matches the documented determination — initializing without a consent prompt only if the audit found no non-essential storage use, otherwise gated behind the consent decision

### Requirement: Operator legal-setup checklist
`docs/operations.md` MUST document, as a pre-go-live checklist: what to fill into the legal-notice/privacy-policy placeholders, which optional integrations (S3, SMTP, Sentry, OTel collector) require an Art. 28 GDPR data-processing agreement with that provider when enabled, a pointer to the existing retention/data-subject-rights documentation, and a note on accessibility (BFSG) applicability to assess.

#### Scenario: Operator enables an optional integration
- **WHEN** an operator sets `S3_ENDPOINT`, `SMTP_HOST`, `VITE_SENTRY_DSN`, or `OTEL_EXPORTER_OTLP_ENDPOINT` for their deployment
- **THEN** the checklist identifies that integration as requiring a data-processing agreement with the chosen provider before going live
