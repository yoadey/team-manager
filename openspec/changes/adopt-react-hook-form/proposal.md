## Why

Forms use a single global, untyped `state.form` bag in the God-context: every field writes via `onFormInput` into `state.form`, and a single shared `busy` flag (with a hand-written `clearBusyIfOwned` race guard) gates all Save buttons. Validation is hand-written in `utils/validation.ts`, returns a single message at a time, and duplicates backend constraints — already a third source of truth beside `openapi.yaml` and `validate.go`. Adopting React Hook Form gives each form its own local state (dissolving the global form bag and `busy` flag) and per-field errors that fit the existing `Field` API.

## What Changes

- Add React Hook Form; each of the 7 form sheets gets its own `useForm`.
- Add Zod for runtime validation, but **generate the base schemas from `openapi.yaml`** (not hand-written) so validation shares the spec as source of truth; cross-field rules stay as hand-written `.superRefine()`s.
- Replace the global `busy` flag with per-form `isSubmitting`.
- Remove the global `state.form` bag, `formValues`, `clearBusyIfOwned`, and reduce `validation.ts` to the coercion helpers still needed.

## Capabilities

### New Capabilities
- `form-handling`: how forms manage local state, validate input at runtime, and surface per-field errors, without a shared global form buffer.

### Modified Capabilities
<!-- none -->

## Impact

- Frontend: the 7 `*FormSheet.tsx` + tests, per-form `<form>Schema.ts`, RHF-bound field components in `ui.tsx`, `AppContext.tsx` (form state removed), `utils/validation.ts` (trimmed), `utils/forms.ts` (removed), generated `src/api/zod.gen.ts`, generation pipeline, `package.json`.
- Overlaps `AppContext.tsx` with the tanstack-query change — sequence merges.
