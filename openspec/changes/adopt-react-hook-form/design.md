## Context

Fields bind via `useAppSelector(s => s.form[name])` + `onFormInput` into one shared `state.form` object; `formValues<T>()` casts it per sheet. Validators in `validation.ts` return one error at a time and mirror backend limits (comments literally cite `openapi.yaml` and `validate.go`). The interesting rules are cross-field/semantic (meet≤start, end>start, repeat-weeks range, duplicate poll options, date-range order) and coercion (German decimal comma). There are 7 form sheets, each with tests.

## Goals / Non-Goals

**Goals:**
- Per-form local state via RHF; remove the global form bag and `busy` flag from the context.
- Runtime validation derived from the spec, not a new hand-written schema source.
- Per-field error display using the existing `Field` API.

**Non-Goals:**
- Making client validation a security boundary — the server (`validate.go`) stays authoritative.
- Rewriting the coercion logic (German comma money parsing stays custom).

## Decisions

- **Do NOT hand-write Zod base schemas** — generate `src/api/zod.gen.ts` from `openapi.yaml` (orval / openapi-zod-client), wired into `make generate-ts` and drift-checked.
- Cross-field rules live as `.superRefine()` on the generated base schema, one place per form, reusing existing i18n messages.
- Money comma coercion stays as `z.preprocess`/`.transform`.
- Migrate `EventFormSheet` first (richest validation), then the other six.
- The visual `Field` wrapper (label, error, aria) is kept; only the data binding changes (`register`/`Controller`).

## Risks / Trade-offs

- Zod adds runtime weight; verify against the bundle budget.
- Generated schemas capture only structural spec constraints; semantic rules remain hand-written — Zod relocates that logic, it does not eliminate it.
- Concurrent edits with the tanstack-query change in `AppContext.tsx`.
