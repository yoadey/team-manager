## 1. Legal content and rendering
- [ ] 1.1 Add `frontend/src/legal/impressum.de.md`, `impressum.en.md`, `datenschutz.de.md`, `datenschutz.en.md` with explicit `[BETREIBER: ...]`/`[OPERATOR: ...]` placeholder markers covering every `§5 DDG` field (name/legal form, address, contact, register entry if any, VAT ID if any) and every Art. 13 GDPR field (categories of data, purposes, legal basis, retention — cross-reference `SECURITY.md`'s retention table, recipients/processors, data-subject rights — cross-reference `docs/gdpr-data-subject-rights.md`, right to lodge a complaint with a supervisory authority)
- [ ] 1.2 Add a small Markdown-rendering component (or reuse an existing lightweight renderer if one already exists in the dependency tree — check before adding a new dependency, per `CLAUDE.md`'s "justify new runtime deps") for the legal pages
- [ ] 1.3 `context/urlState.ts` / `components/Root.tsx`: add unauthenticated legal pseudo-routes (e.g. `/impressum`, `/datenschutz`) rendered before the `phase === 'login' | 'app'` switch, independent of `Route`/`ALL_ROUTES`/RBAC
- [ ] 1.4 New `Footer` component with legal-notice + privacy-policy links; wire into `Login.tsx`, `Register.tsx`, and `layouts/AppShell.tsx` (both compact and desktop layouts)
- [ ] 1.5 `i18n/de.ts` / `i18n/en.ts`: add strings for the footer links and legal page chrome (back-to-login navigation, page titles)

## 2. Self-registration transparency and age gate
- [ ] 2.1 `features/auth/components/Register.tsx`: add a privacy-policy link near the email/password fields
- [ ] 2.2 Add a required, unchecked-by-default "at least 16 years old" checkbox; disable submit until checked; no `doRegister` call fires without it
- [ ] 2.3 `Register.test.tsx`: cover submit-blocked-without-checkbox and submit-enabled-with-checkbox
- [ ] 2.4 `docs/end-user/daten-und-datenschutz.md`: document that self-registration is for members ≥16 and that younger members are added via the existing admin-invite flow

## 3. Sentry storage/consent determination
- [ ] 3.1 With `VITE_SENTRY_DSN` set locally, inspect actual `document.cookie`/`localStorage`/`sessionStorage` writes from `Sentry.browserTracingIntegration()` in a real browser session (not just source-reading the SDK, since behavior is version-specific)
- [ ] 3.2 Record the finding as a comment in `monitoring.ts` and in the new `docs/operations.md` section
- [ ] 3.3 If non-essential storage is written: implement a minimal consent gate before `Sentry.init` (or drop `browserTracingIntegration`, keeping only error capture) per design.md Decision 4; if not: no code change needed beyond the recorded determination

## 4. Operator documentation
- [ ] 4.1 `docs/operations.md`: new "Legal setup before going public" section — placeholder fields to fill in, per-integration Art. 28 AVV checklist (S3/SMTP/Sentry/OTel), pointer to `SECURITY.md` retention table and `docs/gdpr-data-subject-rights.md`, BFSG/EN 301 549 applicability note (assessment prompt, not a code gate)
- [ ] 4.2 `CLAUDE.md`: short pointer to the new `docs/operations.md` section (mirrors how other operational runbooks are referenced)
- [ ] 4.3 `docs/end-user/daten-und-datenschutz.md`: cross-link the new in-app privacy-policy page

## 5. Verification
- [ ] 5.1 `npm run typecheck`, `npm run lint`, `npm test` (frontend) green, including new `Footer`/`Register` tests
- [ ] 5.2 `npm run build` (bundle-size budget: 250 KB/chunk, 600 KB total gzipped) still passes with the new Markdown content + renderer
- [ ] 5.3 Lighthouse CI (`frontend-lighthouse`) passes on the new legal pages (accessibility score, since they're new public-facing routes)
- [ ] 5.4 Manually verify `/impressum` and `/datenschutz` render pre-login, and that the footer is reachable in both compact and desktop layouts
