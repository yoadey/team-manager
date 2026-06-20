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

This repository is the **frontend** application. The backend is currently a
mock (`src/services/serviceLayer.ts`); reports about mock data behaviour are out
of scope. Relevant frontend concerns include, for example: XSS, CSP gaps,
client-side authorization bypass, leakage of secrets/PII, and dependency
vulnerabilities.

## Hardening Already in Place

- Content-Security-Policy and security headers (dev server + `index.html`;
  production server must emit `frame-ancestors`/`X-Frame-Options`).
- `dangerouslySetInnerHTML` is blocked via ESLint; HTML must be sanitised.
- PII (email/IP) is stripped before events are sent to Sentry.
- `npm audit` (high/critical, production deps) runs in CI and blocks merges.
- Dependabot keeps dependencies and GitHub Actions up to date.
