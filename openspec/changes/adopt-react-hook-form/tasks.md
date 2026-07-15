## 1. Dependencies & generation
- [ ] 1.1 Add `react-hook-form@^7`, `zod@^3`, `@hookform/resolvers@^3`; dev tool `openapi-zod-client` (or orval)
- [ ] 1.2 Generate `src/api/zod.gen.ts` from `openapi.yaml`; wire into `make generate-ts` and the drift check

## 2. Schemas
- [ ] 2.1 Per form, add `<feature>/<form>Schema.ts`: generated base schema + `.superRefine()` for cross-field rules ported from `validation.ts`, reusing i18n messages
- [ ] 2.2 Keep money comma coercion as `z.preprocess`/`.transform`

## 3. Field binding
- [ ] 3.1 Add RHF-bound field components (or extend `Field`/`FormText`) using `register`/`Controller`; keep the visual `Field` wrapper (label, error, aria)

## 4. Migrate sheets
- [ ] 4.1 Migrate `EventFormSheet` first (`useForm` + `zodResolver`; submit disabled on `isSubmitting`; per-field errors); update `useEventFormActions.ts` to take validated values
- [ ] 4.2 Migrate the other six: `AbsenceFormSheet`, `PenaltyFormSheet`, `ContribFormSheet`, `TxFormSheet`, `NewsFormSheet`, `PollFormSheet`

## 5. Remove global form state
- [ ] 5.1 Remove `state.form`, `busy`, `onFormInput`, `setFormVal`/`setFormValues` and related actions from `AppContext.tsx`
- [ ] 5.2 Delete `utils/forms.ts` (`formValues`, `clearBusyIfOwned`); trim `validation.ts` to remaining coercion helpers

## 6. Verification
- [ ] 6.1 `npm run typecheck` + `npm run lint` (incl. `exhaustive-deps`) green
- [ ] 6.2 `npm run test` green; the 7 `*FormSheet.test.tsx` updated to RHF; new tests for `.superRefine()` rules
- [ ] 6.3 `npm run build` + `check:bundle` under budget (RHF + Zod)
- [ ] 6.4 `make generate-ts` produces `zod.gen.ts` with no diff
- [ ] 6.5 Manual smoke per form: empty required → field error; invalid time/money/date → correct localized field error; double-click Save → single submit
