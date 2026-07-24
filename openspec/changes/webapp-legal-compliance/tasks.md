## 1. Legal content and rendering
- [x] 1.1 Add `frontend/src/features/legal/content.ts` with structured DE/EN content (no Markdown dependency added — see design.md Decision 1) with explicit `[BETREIBER: ...]`/`[OPERATOR: ...]` placeholder markers covering every `§5 DDG` field and every Art. 13 GDPR field (cross-referencing `SECURITY.md`'s retention table and `docs/gdpr-data-subject-rights.md`)
- [x] 1.2 `features/legal/components/LegalSheet.tsx` renders the structured content with plain semantic elements (no Markdown parser needed)
- [x] 1.3 Implemented as a new `legal` sheet type (`SheetState.legalPage`, team-independent `openLegal` action) rather than new pseudo-routes — `SheetHost` already renders regardless of `state.phase`, so this reaches pre-login and post-login alike with no `urlState.ts`/`Root.tsx` changes (see design.md Decision 2 for why this beat the original plan)
- [x] 1.4 `features/legal/components/LegalFooter.tsx` wired into `Login.tsx` (pre-login, covers the register view too since `Register` renders inside `Login`'s card) and into `ProfileSheet` (`features/team/components/NavSheets.tsx`, new "Rechtliches" section — reachable from the always-visible profile button in both compact and desktop shells)
- [x] 1.5 `i18n/de.ts` / `i18n/en.ts`: added `sheet.legalImpressum`/`legalDatenschutz`, `team.legalSection`, `auth.registerPrivacyIntro`/`registerAgeConfirm`/`footerLegalHint`

## 2. Self-registration transparency and age gate
- [x] 2.1 `features/auth/components/Register.tsx`: privacy-policy link next to the age checkbox (opens the `legal`/`datenschutz` sheet)
- [x] 2.2 Required, unchecked-by-default "at least 16 years old" checkbox; submit button disabled and `handleSubmit` early-returns without it (belt-and-suspenders against Enter-key form submit bypassing the disabled button)
- [x] 2.3 `Register.test.tsx`: submit-blocked-without-checkbox, submit-enabled-with-checkbox, privacy-link-opens-datenschutz
- [x] 2.4 `docs/end-user/daten-und-datenschutz.md`: documented that self-registration requires confirming age ≥16 and that younger members are added via the admin-invite flow

## 3. Sentry storage/consent determination
- [x] 3.1 Verified via a jsdom-based Vitest probe (`Sentry.init` with this app's exact `browserTracingIntegration()` config, then `captureMessage`/`setUser`) that no cookies/localStorage/sessionStorage keys are written, corroborated by Sentry's own docs/community answers (sessionStorage use is tied to the Replay integration, which this app does not enable) — see the PR description for the (deleted, throwaway) probe test and its output
- [x] 3.2 Recorded the finding as a comment above `initMonitoring` in `monitoring.ts` and in `docs/operations.md`'s new section; updated the "Cookies und lokale Speicherung"/"Cookies and local storage" content section from a placeholder to the actual determination (still flagging that it must be re-verified if Replay/session-tracking is ever added)
- [x] 3.3 No non-essential storage found → no consent-gate code change needed, per the determination above

## 4. Operator documentation
- [x] 4.1 `docs/operations.md`: new "Legal setup before going public" section — placeholder fields to fill in, per-integration Art. 28 AVV checklist (S3/SMTP/Sentry/OTel), pointer to `SECURITY.md` retention table and `docs/gdpr-data-subject-rights.md`, BFSG/EN 301 549 applicability note (assessment prompt, not a code gate), self-registration age gate
- [x] 4.2 `CLAUDE.md`: pointer to the new `docs/operations.md` section
- [x] 4.3 `docs/end-user/daten-und-datenschutz.md`: cross-linked the new in-app privacy-policy page (Profil → Rechtliches) and the login-screen footer link

## 5. Verification
- [x] 5.1 `npm run typecheck`, `npm run lint`, `npm test` (frontend) green — 1184/1184 tests pass, including new `LegalSheet`/`LegalFooter`/`Register`/`ProfileSheet` tests
- [x] 5.2 `npm run build` + `npm run check:bundle` — 263.7 KB total gzipped, all chunks within the 250 KB/chunk · 600 KB total budget
- [ ] 5.3 Lighthouse CI (`frontend-lighthouse`) — runs in CI only (Chrome-launcher based, not exercised locally in this session); left for CI to confirm
- [x] 5.4 Manually verified via a Playwright smoke check against the dev server: the legal sheet opens pre-login (Login footer, both Impressum and Datenschutzerklärung) and post-login (ProfileSheet "Rechtliches")
