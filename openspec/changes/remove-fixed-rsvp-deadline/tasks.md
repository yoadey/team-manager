## 1. Spec
- [x] 1.1 Remove `rsvpDeadline` from `Event`, `EventSeries`, and their create/update request schemas in `openapi.yaml`
- [x] 1.2 Remove the RSVP-deadline-passed problem+json documentation that's now solely about `cancelLeadMinutes`
- [x] 1.3 Run `make generate` + repo-root `make generate-ts`; commit generated output

## 2. Migration + backend
- [x] 2.1 New migration: drop `rsvp_deadline` column from `events` and `event_series`
- [x] 2.2 Remove `RsvpDeadline` from `internal/events/model.go` (EventRow, EventSeriesRow, Create/UpdateParams, etc.)
- [x] 2.3 Remove `rsvp_deadline` reads/writes from `internal/events/repository.go` (select columns, insert/update params, cutoff-enforcement query)
- [x] 2.4 Collapse `internal/events/service.go`'s deadline check to `cancelLeadMinutes` only; drop the `rsvpDeadline` branch and the now-redundant `ErrRsvpDeadlinePassed` sentinel (`ErrCancelLeadTimePassed` already covers this cutoff)
- [x] 2.5 `internal/events/handler.go`: drop the `ErrRsvpDeadlinePassed` → 409 mapping

## 3. Frontend
- [x] 3.1 Remove the RSVP-deadline `Field` (datetime-local input + hint) from `EventFormSheet.tsx`
- [x] 3.2 Restyle the cancellation lead-time hours/minutes inputs: visible per-field labels with unit suffix instead of placeholder-only boxes
- [x] 3.3 Drop `rsvpDeadline` from `eventFormSchema.ts`, `types.ts` (`EventFormValues`, `EventDto`), `useEventFormActions.ts`, `useEventMutations.ts`
- [x] 3.4 `rsvpCutoff.ts`: `effectiveRsvpCutoff`/`isRsvpCutoffPassed` drop the `rsvpDeadline` candidate, `cancelLeadMinutes` only
- [x] 3.5 `EventDetailSheet.tsx`, `api/map.ts`, `mocks/db.ts`/`mocks/handlers.ts`: drop `rsvpDeadline` references
- [x] 3.6 `de`/`en` i18n: remove now-unused `fieldRsvpDeadline*` strings, add labels for the restyled hours/minutes inputs if needed

## 4. Verification
- [x] 4.1 openapi-drift green (regenerated clients committed)
- [x] 4.2 Backend tests updated: cutoff enforcement tests exercise only `cancelLeadMinutes`; `golangci-lint` + `go test ./...` green; migration-rollback (up→down→up) + migration-safety green (single `DROP COLUMN`, no unsafe DDL)
- [x] 4.3 Frontend lint/typecheck/test/build + bundle budget green
