# Security Policy

## Supported Versions

This project is under active development. Security fixes are applied to the
`main` branch and the latest release only.

## Reporting a Vulnerability

**Please do not open public issues for security vulnerabilities.**

Report suspected vulnerabilities privately via GitHub's
[private vulnerability reporting](https://github.com/yoadey/team-manager/security/advisories/new)
("Report a vulnerability" on the repository's Security tab).

Please include:

- A description of the issue and its potential impact
- Steps to reproduce (proof-of-concept where possible)
- Affected version / commit
- Any suggested remediation

We aim to acknowledge reports within **5 business days** and to provide a
remediation timeline after triage. Please allow a reasonable disclosure window
before any public discussion.

## Scope

This repository is a monorepo with a React/TypeScript **frontend** (`frontend/`)
and a Go **backend** (`backend/`). The frontend can also run against a
localStorage mock (`frontend/src/services/serviceLayer.ts`); reports about mock
data behaviour are out of scope. In-scope concerns include, for example: XSS,
CSP gaps, authentication/authorization bypass, RBAC flaws, leakage of
secrets/PII, SQL injection, and dependency vulnerabilities — on either tier.

## Data Protection (GDPR)

The application stores personal data of club members (name, email, phone,
birthday, address, photo). The following data-subject rights are implemented:

- **Right of access (Art. 15):** a member can export all personal data held
  about them as a JSON document via `GET /api/v1/auth/me/data-export`
  ("Meine Daten exportieren" in the profile sheet).
- **Right to erasure (Art. 17) — by anonymization:** `DELETE /api/v1/auth/me`
  overwrites the member's identifying fields (name, email, phone, birthday,
  address, photo, password hash), strips free-text PII from their comments and
  absence/attendance reasons, and deletes their sessions. Records that are
  **shared** (attendance, event history) or must be **retained** for accounting
  (transactions, penalty assignments, contributions) are kept in anonymized
  form so they no longer resolve to an identifiable person. Erasure is
  irreversible.
- **Confirmation without a password:** because accounts may authenticate via
  OIDC and have no password, erasure is authorized by the active session and
  confirmed by retyping the account email (verified server-side).
- Security-sensitive actions (login, logout, account erasure) emit structured
  **audit-log** events.
- **Storage limitation (Art. 5(1)(e)):** a daily retention job deletes rows
  once they age past a configurable window: notifications after
  `RETENTION_NOTIFICATIONS_DAYS` (default 90), expired sessions after
  `RETENTION_SESSIONS_DAYS` past expiry (default 30), and `audit_log` rows
  after `RETENTION_AUDIT_LOG_DAYS` (default 365) — raise the audit-log window
  if your organization's retention policy requires longer.

## Hardening Already in Place

- Content-Security-Policy and security headers (dev server + `index.html`;
  production server must emit `frame-ancestors`/`X-Frame-Options`).
- `dangerouslySetInnerHTML` is blocked via ESLint; HTML must be sanitised.
- PII (email/IP) is stripped before events are sent to Sentry.
- `npm audit` (high/critical, production deps) runs in CI and blocks merges.
- Dependabot keeps dependencies and GitHub Actions up to date.
