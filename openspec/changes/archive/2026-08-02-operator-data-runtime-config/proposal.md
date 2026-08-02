## Why

`frontend/src/features/legal/content.ts` currently bakes every operator identity/contact field (name, address, phone, email, register entry, VAT ID, which optional integrations are in use) into the frontend source as `[BETREIBER: ...]`/`[OPERATOR: ...]` placeholders that must be hand-edited before the production Docker image is built (`openspec/changes/archive/2026-07-24-webapp-legal-compliance/design.md` Decision 1). In practice this means the same operator, deploying the same released image (e.g. `ghcr.io/yoadey/team-manager-frontend:vX.Y.Z`), cannot fill in their own legal-notice data without forking the source and rebuilding — every other per-deployment setting the frontend needs (`API_BASE_URL`, `SENTRY_DSN`, `VAPID_PUBLIC_KEY`) is already a container-start runtime override (`frontend/docker/docker-entrypoint-runtime-config.sh`) precisely so one built image serves any deployment; operator identity data is the odd one out and blocks a straightforward "point the same image at your own instance" deploy.

Decision 1's stated reason for keeping this build-time — "multi-paragraph prose is a poor fit for an env var" — holds for the generic legal boilerplate (Haftung für Inhalte, Streitschlichtung, GDPR rights, retention, etc.), which is not operator-specific and isn't changing here. It does not hold for the operator-identity fields themselves, which are short scalars (a name, a street, a phone number) exactly like the DSN/key values already handled this way.

## What Changes

- Extend the existing runtime-config mechanism (`window.__RUNTIME_CONFIG__`, populated by `frontend/docker/docker-entrypoint-runtime-config.sh` via `envsubst` at container start) with a new `OPERATOR_*` field group: name, legal form, street, postal code, city, represented-by, phone, email, register court/number, VAT ID, data-protection contact email, and free-text descriptions of the S3/SMTP/Sentry/OTel sub-processors in use.
- `frontend/src/features/legal/content.ts` changes from static constants to a function of this operator config: each field renders the deploy-time value when set, and falls back to the existing `[BETREIBER: ...]`/`[OPERATOR: ...]` placeholder text when unset — preserving the "obviously incomplete beats silently non-compliant" behavior from the archived change, now driven by config instead of source edits. Sections that only make sense when a corresponding field is present (Vertreten durch/Represented by, Registereintrag/Register entry, USt-IdNr/VAT ID) are omitted entirely when that field is unset, instead of rendering a placeholder telling the operator to delete the section by hand.
- The generic legal boilerplate paragraphs (liability, dispute resolution, GDPR purposes/rights/retention text) stay hardcoded source, unchanged — only the operator-identity fields move to runtime config.
- `docs/operations.md`'s "Legal setup before going public" checklist is rewritten from "edit `content.ts` before building" to "set `OPERATOR_*` environment variables on the frontend container," alongside the existing `API_BASE_URL`/`SENTRY_DSN`/`VAPID_PUBLIC_KEY` runtime-override documentation.
- **BREAKING** (deploy-config surface, not an API): operators who already forked `content.ts` to fill in their placeholders keep working (their edited file still renders — they simply won't pick up defaults from env vars unless they also revert to the shipped `content.ts`), but the *documented* path changes from "edit source" to "set env vars," so existing deployments following the old checklist should migrate to `OPERATOR_*` vars to stay on the supported path.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `legal-compliance`: "Public legal notice (Impressum) page" and "Public privacy policy (Datenschutzerklärung) page" requirements gain a runtime-configuration mechanism for operator-identity fields, replacing the build-time-source-edit workflow.

## Impact

- `frontend/src/config.ts`: new `operator` config object read from `window.__RUNTIME_CONFIG__.OPERATOR_*`, same pattern as `resolveSentryDsn`/`resolveVapidPublicKey`.
- `frontend/src/features/legal/content.ts`: restructured from static `LegalPageText` constants to a builder function taking the operator config.
- `frontend/docker/config.js.template`, `frontend/docker/docker-entrypoint-runtime-config.sh`: add the `OPERATOR_*` variables to the substituted set.
- `frontend/public/config.js` (local-dev/test defaults): document the new keys as blank defaults (placeholders render, matching today's dev experience).
- `apps/base/team-manager/frontend.yaml` / `apps/base/team-manager-test/frontend.yaml` in the `fluxcd-talos-cluster` deployment repo (out of scope for this change's tasks, but the eventual consumer of this env-var surface).
- `docs/operations.md`: "Legal setup before going public" section rewritten.
- Tests: `content.ts`/`LegalSheet.test.tsx`/`LegalFooter.test.tsx` updated for the new config-driven rendering (default/unset-env-var behavior must still show placeholder markers, per the existing "Placeholder content is visibly unfilled" requirement).
