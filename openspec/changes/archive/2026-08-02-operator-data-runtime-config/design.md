## Context

`openspec/changes/archive/2026-07-24-webapp-legal-compliance/design.md` Decision 1 chose to ship legal-notice/privacy-policy content as build-time source (`frontend/src/features/legal/content.ts`), rejecting a runtime-injected value because "legal-notice/privacy-policy text is multi-paragraph prose with headings and links — a poor fit for a single env var." That reasoning is sound for the boilerplate paragraphs (liability, dispute resolution, GDPR rights/retention text), which are generic and don't vary per operator. It does not hold for the operator-*identity* fields specifically (name, address, phone, email, register entry, VAT ID) — those are short scalars, the same shape as `SENTRY_DSN`/`VAPID_PUBLIC_KEY`, which already use `frontend/docker/docker-entrypoint-runtime-config.sh` to regenerate `window.__RUNTIME_CONFIG__` (served as `/config.js`) from container env vars at startup, so one built image serves any deployment.

Operators today must fork `content.ts` and rebuild to fill in their own identity data — the one config surface in the whole app that doesn't follow the "same image, different env vars" deployment model everything else uses (see `docs/operations.md`'s `API_BASE_URL`/`SENTRY_DSN`/`VAPID_PUBLIC_KEY` container-start override docs). This change closes that gap for the identity fields only.

## Goals / Non-Goals

**Goals:**
- An operator deploying the stock `ghcr.io/yoadey/team-manager-frontend` image can fill in their legal-notice/privacy-policy identity data via container env vars, with no fork/rebuild.
- Preserve the existing "loud, obviously-incomplete placeholder beats a page that looks complete but is legally empty" behavior (`legal-compliance` spec, "Placeholder content is visibly unfilled") for any field left unset.
- Keep the generic legal boilerplate (liability, dispute resolution, GDPR purposes/rights/retention) as static source — it isn't operator data and doesn't belong in an env var either.

**Non-Goals:**
- Moving the boilerplate paragraphs to runtime config too. Still a poor fit per the original Decision 1 reasoning; this change only touches the identity/processor-disclosure fields.
- A mounted-file/JSON-blob config format. A dozen-odd flat scalars fit the existing `envsubst`-templated `config.js.template` pattern directly; a new file-mount mechanism would be a second, inconsistent config path for values one syntax cleaner than the one already in use for everything else in this file.
- Auto-deriving which processors to disclose from *other* env vars being set (e.g. inferring "S3 is in use" from `S3_ENDPOINT` on the backend). Rejected for the same reason the archived design rejected it: the frontend container never sees backend env vars, and even if it did, the actual disclosure sentence (which provider, which region, EU vs. third country) isn't derivable from an endpoint URL — the operator has to write it. Each processor line is its own explicit `OPERATOR_*_PROVIDER` free-text var, opted into individually.

## Decisions

### 1. Sixteen new `OPERATOR_*` runtime env vars, added to the existing `config.js.template` mechanism

Extends `window.__RUNTIME_CONFIG__`/`config.js.template`/`docker-entrypoint-runtime-config.sh` (same files that already carry `API_BASE_URL`/`SENTRY_DSN`/`VAPID_PUBLIC_KEY`) with:

`OPERATOR_NAME`, `OPERATOR_LEGAL_FORM`, `OPERATOR_STREET`, `OPERATOR_POSTAL_CODE`, `OPERATOR_CITY`, `OPERATOR_REPRESENTED_BY`, `OPERATOR_PHONE`, `OPERATOR_EMAIL`, `OPERATOR_REGISTER_COURT`, `OPERATOR_REGISTER_NUMBER`, `OPERATOR_VAT_ID`, `OPERATOR_DATA_PROTECTION_EMAIL`, `OPERATOR_S3_PROVIDER`, `OPERATOR_SMTP_PROVIDER`, `OPERATOR_SENTRY_PROVIDER`, `OPERATOR_OTEL_PROVIDER`.

All optional (same "blank env var is treated as unset" convention `config.ts`'s `runtimeConfig()` already uses). `frontend/src/config.ts` gains a `config.operator` object read via the same `runtimeConfig()` helper, one field per var.

### 2. `content.ts` becomes a builder function over the operator config, with two different "unset" behaviors

- **Always-present fields** (name, street, postal code + city, phone, email — the core `§5 DDG` block and the `Kontakt` section): unset renders the existing `[BETREIBER: ...]`/`[OPERATOR: ...]` placeholder text, same as today. These fields are mandatory for any legal-notice page to be non-empty, so the loud-placeholder behavior stays exactly as-is — the difference is only *where* the value comes from (env var vs. hardcoded source).
- **Optional, section-gated fields** (legal form, represented-by, register court/number, VAT ID, and all four processor-disclosure lines): unset omits the whole section/list item, rather than rendering a placeholder instructing the operator to delete it by hand. This is a behavior change from the archived version, which always rendered these with a "fill in if applicable, else remove this section" placeholder. Justification: these fields are conditionally applicable (a private individual has no register entry; an operator with no VAT ID has no VAT section; a disabled integration has nothing to disclose) — with build-time source editing, "remove this section" was a one-time manual edit; with runtime config, there is no equivalent one-time edit available, so the marker would otherwise persist forever on deployments where it's genuinely not applicable. Trade-off accepted and documented below.

`frontend/src/features/legal/content.ts` keeps its existing structured-data shape (`LegalPageText { title, sections: { heading, paragraphs, list? }[] }`) and rendering path (`LegalSheet`, plain semantic elements) unchanged — only how the `paragraphs`/`list` arrays for the identity/processor fields are populated changes, from static literals to values pulled from `config.operator`.

### 3. `docs/operations.md` documents the new vars as a `OPERATOR_*` table, alongside the existing runtime-override table

Same location/format as the existing `SENTRY_DSN`/`VAPID_PUBLIC_KEY` container-start documentation, not a new section — this is the same mechanism, just more variables. The "before going public" checklist item that said "edit `content.ts`" becomes "set the `OPERATOR_*` env vars on the frontend container (or leave any of them unset to keep the placeholder for that field)."

## Risks / Trade-offs

- **Silent omission risk for the optional fields**: unlike the always-present fields, an operator who enables S3/SMTP/Sentry/OTel but forgets to also set the matching `OPERATOR_*_PROVIDER` var gets a privacy policy that silently doesn't mention that processor, instead of a loud placeholder demanding attention (see Decision 2). Mitigated by calling this out explicitly in `docs/operations.md`'s checklist ("if you enable integration X, also set `OPERATOR_X_PROVIDER` or the disclosure is silently omitted") — but it is a real behavior change from the archived version's stricter "always visible, always loud" default, accepted as the cost of the sections being removable at all without a rebuild.
- **Legacy deployments that already forked `content.ts`**: keep working unchanged (their fork still renders whatever they wrote), but no longer receive the "flip an env var" story unless they revert to the shipped file — documented in the proposal as the deploy-config migration path, not a breaking API/data change.
- **Not a legal opinion**, same caveat as the archived change: this is engineering's best-effort mechanism for operators to supply their own data, not legal review of what that data should say.
