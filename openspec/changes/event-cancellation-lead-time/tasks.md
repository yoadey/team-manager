## 1. Spec
- [ ] 1.1 Add optional `cancelLeadMinutes` (integer, minutes) to `Event`, `EventSeries`, and their create/update request schemas in `openapi.yaml`
- [ ] 1.2 Document the cutoff-enforcement error on the attendance endpoint (problem+json, e.g. 409)
- [ ] 1.3 Run `make generate` + repo-root `make generate-ts`; commit generated output

## 2. Migration + backend
- [ ] 2.1 Migration: add nullable `cancel_lead_minutes` column to events (and series) tables
- [ ] 2.2 Persist/read the field in `events` repository (create/update/get/list), preserving team-scoping
- [ ] 2.3 Enforce cutoff in `events.Service` attendance mutation: compute `start − cancelLeadMinutes`; reject self-service changes past it; allow `write`-on-`events` override
- [ ] 2.4 Series occurrences inherit the field and compute their own cutoff

## 3. Frontend
- [ ] 3.1 Event form: hours + minutes inputs mapped to a single `cancelLeadMinutes`
- [ ] 3.2 `EventDetailSheet.tsx`: show the effective cutoff; disable RSVP controls once passed; surface the server rejection as a toast
- [ ] 3.3 `de`/`en` strings; MSW handlers honor and enforce the field

## 4. Verification
- [ ] 4.1 openapi-drift green (regenerated clients committed)
- [ ] 4.2 Backend tests: cutoff passed → self-service rejected, organizer override allowed, series per-occurrence cutoff, timezone-correct boundary
- [ ] 4.3 `golangci-lint` + `go test ./...` green; migration-rollback (up→down→up) + migration-safety green
- [ ] 4.4 Frontend lint/typecheck/test/build + bundle budget green
