## Context

Teamverwaltung is self-hosted per deployment (Docker Compose or the Helm chart) — each instance has exactly one operator (the club or whoever runs it for a club). That shapes every decision below: this is not a multi-tenant SaaS where "the operator" varies per team inside one running instance, so legal-notice/privacy-policy content is an **instance-level** concern, not a per-team database row.

Legal grounding used throughout (Germany/EU, since the app is German-first — `i18n` defaults to `de`, `README.md` is German, "Verein" terminology throughout):

- **`§5 DDG`** (Digitale-Dienste-Gesetz, formerly `§5 TMG`): Impressumspflicht — legal notice must be easily recognizable, directly reachable (conventionally ≤2 clicks, footer link), and permanently available, on every "geschäftsmäßig" operated site. Club/association sites are treated as in scope by prevailing case law even when non-commercial.
- **Art. 13 GDPR**: information must be provided *at the point personal data is collected*, not merely be available somewhere.
- **Art. 8 GDPR**: for information-society services offered directly to a child, processing based on consent requires the child be ≥16 (Germany did not lower the national threshold) or that a holder of parental responsibility consent/authorize it.
- **`§25 TDDDG`** (formerly TTDSG, ePrivacy transposition): consent is required to store or access information on the end device unless it is *strictly necessary* for a service explicitly requested by the user — broader than "cookies with PII," and applies to `localStorage`/`sessionStorage` access too.
- **Art. 28 GDPR**: a written data-processing agreement is required with every processor (hosting/S3/SMTP/error-tracking/telemetry vendor) that touches personal data on the controller's behalf.

## Goals / Non-Goals

**Goals:**
- Every screen a visitor can reach — including pre-login — has a reachable path to the legal notice and privacy policy.
- Self-registration discloses the privacy policy at collection time and does not let an under-16 registrant self-provision an account.
- The optional Sentry integration's consent status is a documented, verified decision, not an assumption.
- An operator following `docs/operations.md` end-to-end ends up with a legally defensible deployment, including the parts that are *their* responsibility (DPAs, filling in the legal-notice placeholders) rather than something code can solve for them.

**Non-Goals:**
- An admin-editable / multi-language-CMS legal-content feature. One operator per deployment; content changes are rare and belong with the other pre-deploy configuration (`.env`, `README.md`, `CLAUDE.md`), not a new DB table + admin UI + RBAC surface for a document that changes maybe once a year.
- AGB/Nutzungsbedingungen (terms of service). Not legally required here: the software itself isn't sold to end users under a consumer contract (club fees are a club/member relationship the app doesn't intermediate financially — `finances` module records transactions, it doesn't process payments between the operator and members). Revisit if the product ever adds paid hosting/subscriptions.
- A binding BFSG/EN 301 549 accessibility *requirement*. BFSG's mandatory scope (Anlage 2) centers on consumer e-commerce, banking, and audiovisual-media-access services; a club-internal management tool's fit is genuinely unclear without a real legal opinion, and the existing UI already leans on MUI's accessible primitives plus `vitest-axe` and Lighthouse CI. Document it as a note for the operator to assess for their jurisdiction/use case, not a code gate.
- Server-side rendering of the legal pages for no-JS/crawler reachability. The entire app is a client-rendered SPA already; carving out SSR for two pages would be inconsistent and disproportionate. Accepted limitation, same as the rest of the app.

## Decisions

### 1. Legal content ships as build-time Markdown, not a runtime-configurable field

Two options were weighed:
- **(a) Static Markdown checked into `frontend/src/legal/`**, edited by the operator before building their image (like `README.md`).
- **(b) A runtime-injected value** (mounted file or env var), following the pattern the frontend already uses for `SENTRY_DSN`/`VAPID_PUBLIC_KEY` (build-time `VITE_*` default, overridable by a runtime env var injected at container start).

**Chosen: (a).** Legal-notice/privacy-policy text is multi-paragraph prose with headings and links — a poor fit for a single env var, and the existing runtime-override mechanism exists specifically for short scalar values (a DSN, a public key), not documents. A mounted-file scheme would work but adds a new content-loading path (fetch + Markdown parse + failure handling for a missing mount) for a document that, per the Non-Goals above, changes rarely and is set once at deploy time — the same moment `.env` is written. Ship committed placeholder Markdown (`frontend/src/legal/impressum.de.md`, `.en.md`, `datenschutz.de.md`, `.en.md`) with explicit `[BETREIBER: ...]` / `[OPERATOR: ...]` markers for every field `§5 DDG`/Art. 13 requires, plus a Markdown renderer component. `docs/operations.md` documents that these files must be edited before a production build ships — the same category of "must configure before going live" step as `JWT_PRIVATE_KEY` or `COOKIE_ENCRYPTION_KEY`.

Rejected alternative considered and dropped: generating the privacy-policy processor list dynamically from which env vars are set (`S3_ENDPOINT`, `SMTP_HOST`, `SENTRY_DSN`, `OTEL_EXPORTER_OTLP_ENDPOINT`). Attractive in principle (self-updating disclosure), but the actual sub-processor (which S3-compatible provider, which SMTP relay, physical hosting location, EU vs. third-country) isn't derivable from the env var value alone in a way that produces legally sufficient prose — it would still need the operator to write the actual sentence. Instead, the operator checklist in `docs/operations.md` enumerates each optional integration and tells the operator what to add to the privacy-policy placeholder if they enable it.

### 2. Legal pages are new unauthenticated pseudo-routes, not part of `Route`/`ALL_ROUTES`

`frontend/src/context/urlState.ts`'s `Route` union and `ALL_ROUTES` are all post-login, RBAC-relevant, `ensureRouteData`-backed routes. Legal pages must render **before** authentication (`state.phase === 'login'`) and carry no data-fetch/RBAC concern at all. Rather than overload `Route`, add a small, separate check in `Root.tsx` (e.g. a `legalPage` derived from `location.pathname`/hash, checked before the existing `phase === 'login' ? <Login /> : ...` switch) so `/impressum` and `/datenschutz` render regardless of `phase`. This keeps the RBAC-relevant routing model in `urlState.ts` untouched and matches how `NoTeam` already sits outside the module-permission system as a phase-level concern.

### 3. Self-registration age gate is a self-asserted checkbox, not date-of-birth verification

Real age verification (ID upload, credit-card check) is disproportionate for a club-management tool and out of scope. A required, unchecked-by-default "Ich bin mindestens 16 Jahre alt" checkbox before the submit button enables is the same self-assertion pattern used by essentially every consumer service for this exact GDPR Art. 8 threshold. It does not *prove* age, but it discharges the disclosure/documentation obligation and gives the operator a defensible position; true parental-consent verification, if ever required, is a much larger feature explicitly deferred. Document in `docs/end-user/daten-und-datenschutz.md` and the registration copy that self-registration is for members ≥16; younger members are added via the existing admin-invite flow, which already routes account creation through an adult.

### 4. Sentry consent: audit first, decide after

`Sentry.browserTracingIntegration()` (`monitoring.ts`) is the only optional third-party script in the frontend (Fonts/Material Symbols are already self-hosted via `@fontsource`/`material-symbols`, avoiding the classic Google-Fonts-CDN GDPR issue entirely). Whether it needs `§25 TDDDG` consent turns on whether it writes to `document.cookie`/`sessionStorage`/`localStorage` for anything beyond the current page load — a factual question about the installed `@sentry/react` version's default behavior, not a legal judgment call this document should guess at. Task work must inspect (or instrument and observe) actual browser storage with `VITE_SENTRY_DSN` set, then record the finding directly in `monitoring.ts` as a comment and in the new `docs/operations.md` section:
- If no non-essential storage is written → document "no consent required, disclosure only" and add the Sentry sub-processor line to the privacy-policy placeholder.
- If it does write non-essential storage → gate `initMonitoring()` behind a minimal consent decision (e.g. a `localStorage` opt-in flag set by a small footer/banner prompt) before calling `Sentry.init`, and disable `browserTracingIntegration` (keep only error capture, which is more defensibly "strictly necessary" for operating the service) as the lower-effort alternative to building a full consent-banner UI.

## Risks / Trade-offs

- **Placeholder content risk**: if an operator deploys without editing the placeholders, the app is *more* visibly non-compliant (obvious `[BETREIBER: ...]` text) than if it shipped no page at all — but that's the intended trade-off: a loud gap an operator will notice and fix beats a page that looks complete but is legally empty (e.g., a fabricated address). `docs/operations.md` calls this out as a go-live blocker, not an optional nice-to-have.
- **Not a legal opinion**: this proposal and its implementation are engineering's best-effort translation of publicly known German/EU requirements into shipped defaults and operator documentation. It is not a substitute for the operator obtaining actual legal review before a real public launch, and the docs added here say so explicitly rather than implying certified compliance.
- **Scope creep risk**: legal/compliance work invites "while we're at it" scope (cookie-banner framework, full accessibility audit, AGB). Non-Goals above are deliberately narrow to keep this change reviewable; follow-up changes can pick up anything that turns out to be needed once the Sentry storage audit (Decision 4) lands.
