## 1. Runtime config plumbing
- [x] 1.1 `frontend/docker/config.js.template`: add the 16 `OPERATOR_*` keys to `window.__RUNTIME_CONFIG__`
- [x] 1.2 `frontend/docker/docker-entrypoint-runtime-config.sh`: default each new var to `""` and add them to the `envsubst` variable list
- [x] 1.3 `frontend/public/config.js` (local-dev/test defaults): add the same keys, blank, with a comment pointing at `docs/operations.md`
- [x] 1.4 `frontend/src/config.ts`: extend the `__RUNTIME_CONFIG__` type and `runtimeConfig()` key union with the new keys; add a `config.operator` object (one field per var, using the existing blank-is-unset convention)

## 2. Content builder
- [x] 2.1 `frontend/src/features/legal/content.ts`: replace the static `impressumDe`/`datenschutzDe`/`impressumEn`/`datenschutzEn` constants with builder functions taking `config.operator`, applying the two fallback behaviors from design.md Decision 2 (placeholder for always-present fields, section omission for optional fields)
- [x] 2.2 Keep the boilerplate paragraphs (liability, dispute resolution, GDPR purposes/rights/retention) as static literals inside the builder, unchanged in wording
- [x] 2.3 `LegalPageText`/`LegalSection` types and `LegalSheet` rendering path unchanged (only the exported entry point changed, from a `LEGAL_CONTENT` record to a `getLegalContent(locale, page)` function, updated at both call sites: `LegalSheet.tsx`, `features/legal/index.ts`)

## 3. Tests
- [x] 3.1 New `frontend/src/features/legal/content.test.ts`: unset operator config renders placeholder markers and omits all optional sections; fully-set config renders real values and all optional sections; partially-set config (only `OPERATOR_S3_PROVIDER`) renders exactly that one processor line; Sentry cookie-disclosure paragraph switches on `OPERATOR_SENTRY_PROVIDER`; `OPERATOR_DATA_PROTECTION_EMAIL` falls back to `OPERATOR_EMAIL`
- [x] 3.2 `config.test.ts`: added `config.operator` coverage (unset/all-fields/blank-is-unset). `LegalSheet.test.tsx` left exercising only the already-covered unset-config path (its top-level `LegalSheet` import binds before any test body runs, so testing a configured operator there would need `vi.resetModules()` + dynamic re-import of a React component tree, which risks a "multiple React instances" failure mixing that fresh instance with the file's already-imported `render`/`screen` — the configured-operator behavior is fully covered at the data layer by 3.1 instead)
- [x] 3.3 `npm run typecheck` / `npm run lint` / `npm test` pass (1252/1252 frontend tests green)

## 4. Documentation
- [x] 4.1 `docs/operations.md`: rewrote "Legal setup before going public" step 1 from "edit `content.ts`" to the `OPERATOR_*` env var list (always-required vs. optional/section-omitted), placed alongside the existing `API_BASE_URL`/`SENTRY_DSN`/`VAPID_PUBLIC_KEY` runtime-override docs; updated "Frontend image: pointing it at a backend" with an `OPERATOR_*` example
- [x] 4.2 Step 2 (Art. 28 DPA checklist): cross-referenced the matching `OPERATOR_*_PROVIDER` var for each optional integration
- [x] 4.3 `CLAUDE.md`: updated the "Legal compliance for a public deployment" pointer to name the `OPERATOR_*` container env vars instead of "filling in ... content.ts"

## 5. Verification
- [x] 5.1 `npm run typecheck`, `npm run lint`, `npm test` (frontend) green
- [x] 5.2 `npm run build` + `npm run check:bundle` — 266.1 KB total gzipped, all chunks within the 250 KB/chunk · 600 KB total budget (no regression)
- [x] 5.3 `openspec validate operator-data-runtime-config --strict` passes
