## MODIFIED Requirements

### Requirement: Public legal notice (Impressum) page
The system MUST provide a legal-notice page reachable without authentication, disclosing the operator's identity, address, and contact details per `§5 DDG`. Operator-identity fields (name, legal form, address, represented-by, phone, email, register entry, VAT ID) MUST be configurable at deploy time via container environment variables (`OPERATOR_*`), read at container start — not only by editing and rebuilding the frontend source.

#### Scenario: Legal notice reachable without a session
- **WHEN** a visitor with no active session navigates to the legal-notice page
- **THEN** the page renders the operator's legal-notice content
- **AND** no login or team membership is required

#### Scenario: Placeholder content is visibly unfilled
- **WHEN** the app is deployed without setting the always-required `OPERATOR_*` environment variables (name, street, postal code, city, phone, email)
- **THEN** the rendered page shows explicit, human-readable placeholder markers for those fields (not fabricated real-looking operator details) so an unconfigured deployment is obviously incomplete rather than silently non-compliant

#### Scenario: Operator identity configured via environment variables
- **WHEN** the frontend container starts with `OPERATOR_NAME`, `OPERATOR_STREET`, `OPERATOR_POSTAL_CODE`, `OPERATOR_CITY`, `OPERATOR_PHONE`, and `OPERATOR_EMAIL` set
- **THEN** the legal-notice page renders those values in place of the placeholder markers
- **AND** no rebuild of the frontend image is required

#### Scenario: Optional identity sections omitted when not configured
- **WHEN** `OPERATOR_LEGAL_FORM`, `OPERATOR_REPRESENTED_BY`, `OPERATOR_REGISTER_COURT`/`OPERATOR_REGISTER_NUMBER`, or `OPERATOR_VAT_ID` is left unset
- **THEN** the corresponding section ("Vertreten durch"/"Represented by", "Registereintrag"/"Register entry", "Umsatzsteuer-Identifikationsnummer"/"VAT identification number") is omitted from the rendered page entirely, rather than showing a placeholder

### Requirement: Public privacy policy (Datenschutzerklärung) page
The system MUST provide a privacy-policy page, reachable without authentication, disclosing what personal data is processed, the purposes and legal basis, retention, and known processors, consistent with what `docs/gdpr-data-subject-rights.md` and `SECURITY.md` already document as implemented. The controller identity/contact fields and the per-processor disclosure lines (object storage, outbound email, error monitoring, tracing) MUST be configurable at deploy time via the same `OPERATOR_*` container environment variables as the legal notice.

#### Scenario: Privacy policy reachable without a session
- **WHEN** a visitor with no active session navigates to the privacy-policy page
- **THEN** the page renders the operator's privacy-policy content
- **AND** no login or team membership is required

#### Scenario: Processor disclosure lines configured individually
- **WHEN** the frontend container starts with `OPERATOR_S3_PROVIDER` set but `OPERATOR_SMTP_PROVIDER`, `OPERATOR_SENTRY_PROVIDER`, and `OPERATOR_OTEL_PROVIDER` left unset
- **THEN** the "Empfänger und Auftragsverarbeiter"/"Recipients and processors" list shows only the object-storage line, using the configured text
- **AND** the other three processor lines are omitted rather than shown as placeholders

### Requirement: Operator legal-setup checklist
`docs/operations.md` MUST document, as a pre-go-live checklist: which `OPERATOR_*` environment variables to set on the frontend container and what each controls, which optional integrations (S3, SMTP, Sentry, OTel collector) require an Art. 28 GDPR data-processing agreement with that provider when enabled *and* which matching `OPERATOR_*_PROVIDER` variable must also be set so the privacy policy actually discloses it, a pointer to the existing retention/data-subject-rights documentation, and a note on accessibility (BFSG) applicability to assess.

#### Scenario: Operator enables an optional integration
- **WHEN** an operator sets `S3_ENDPOINT`, `SMTP_HOST`, `VITE_SENTRY_DSN`/`SENTRY_DSN`, or `OTEL_EXPORTER_OTLP_ENDPOINT` for their deployment
- **THEN** the checklist identifies that integration as requiring a data-processing agreement with the chosen provider before going live
- **AND** the checklist identifies the matching `OPERATOR_*_PROVIDER` frontend environment variable that must also be set for the privacy policy to disclose that processor
