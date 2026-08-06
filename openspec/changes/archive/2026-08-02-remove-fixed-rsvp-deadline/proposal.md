## Why

Events currently expose two overlapping ways to cut off attendance changes: an absolute-timestamp **RSVP deadline** (`rsvpDeadline`, a `datetime-local` field) and a **cancellation lead time** (`cancelLeadMinutes`, hours+minutes before start). They enforce the same thing — after which point a member can no longer change their own attendance — and the lead-time variant already supersedes the fixed-date one (it survives series occurrences and doesn't need per-occurrence recomputation). Keeping both confuses organizers filling out the event form and doubles the enforcement/UI surface for no benefit. The lead-time field's own input (two bare number boxes) is also hard to read as a duration.

## What Changes

- Remove the fixed-date RSVP deadline (`rsvpDeadline`) entirely: the field on `Event`/`EventSeries` and their create/update requests, the `rsvp_deadline` DB columns, server-side enforcement, and the event form's datetime-local input.
- `cancelLeadMinutes` becomes the only cutoff mechanism; `effectiveRsvpCutoff`/`isRsvpCutoffPassed` and the backend's equivalent drop the `rsvpDeadline` branch.
- **BREAKING:** existing events with an `rsvpDeadline` set lose that cutoff on migration (silently — no `cancelLeadMinutes` backfill, since a lead time can't be derived from a single past absolute date for series correctness). Acceptable since the feature is unreleased-enough in practice / superseded; called out for reviewers.
- Restyle the cancellation lead-time input in the event form: clearly labeled, unit-suffixed hours/minutes fields instead of two unlabeled number boxes.

## Capabilities

### Removed Capabilities
- None removed outright — `event-cancellation` (lead-time based) remains; only the fixed-date requirement under `event-experience` is dropped.

### Modified Capabilities
- `event-experience`: remove the "Configurable RSVP deadline with a countdown" requirement (fixed-date deadline + its countdown).

## Impact

- Spec/backend: `openapi.yaml` (`Event`, `EventSeries`, create/update request schemas — drop `rsvpDeadline`); regenerate `internal/gen` + `frontend/src/api/*`. `internal/events/{model,service,repository,handler}.go` (drop `RsvpDeadline` field, `ErrRsvpDeadlinePassed`, its enforcement branch and repository wiring). New migration dropping `events.rsvp_deadline` / `event_series.rsvp_deadline`.
- Frontend: `EventFormSheet.tsx` (drop the RSVP-deadline `Field`, restyle the cancel-lead-time `Field`), `eventFormSchema.ts`, `types.ts` (`EventFormValues`/`EventDto`), `rsvpCutoff.ts`, `useEventFormActions.ts`, `useEventMutations.ts`, `EventDetailSheet.tsx`, `mocks/db.ts`/`mocks/handlers.ts`, `api/map.ts`, `i18n/{de,en}.ts`.
- Tests: `backend/internal/events/{service,repository}_test.go`; `frontend/src/features/events/**/*.test.{ts,tsx}`, `services/mappers.test.ts`.
- CI: openapi-drift, migration-rollback/safety, backend + frontend gates. **API + migration-affecting.**
