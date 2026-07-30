## 1. Spec
- [x] 1.1 Add a token-authenticated feed endpoint returning `text/calendar` (`GET /calendar/{token}.ics`), outside session-auth/RBAC, to `openapi.yaml` (already shipped by `add-calendar-feed-subscription`)
- [x] 1.2 Add endpoints to get + rotate the feed token (already shipped) and to read + update the feed's content selection (event-type subset + include-birthdays), session-authenticated, team-scoped (`GET`/`PUT /teams/{teamId}/calendar-feed/settings`, added here)
- [x] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Migration + backend
- [x] 2.1 Migration: added `types text[]` + `include_birthdays bool` columns to the existing `calendar_feed_tokens` table (rather than a new table -- the row already exists per-membership); rotate carries the previous selection forward; default selection = all types + birthdays on
- [x] 2.2 Feed route bypasses session auth (already shipped by `add-calendar-feed-subscription`)
- [x] 2.3 Server-side `.ics` builder: added member birthdays (yearly all-day `VEVENT`, `DTSTART;VALUE=DATE` + `RRULE:FREQ=YEARLY`), filtered to the feed's selected types / birthday toggle and to what the resolved member may see (birthdays additionally gated on `members` read access, since that's the module they live behind in-app). **Not done**: a named `VTIMEZONE` component -- the existing builder already emits UTC (`Z`-suffixed) timestamps for timed events, which every calendar client interprets unambiguously; birthdays are date-only (`VALUE=DATE`) and don't need one either. Adding a real `VTIMEZONE` block is an orthogonal pre-existing gap, out of scope for this content-configurability change.
- [x] 2.4b Endpoint to read + update the feed's content selection (types subset + include-birthdays), session-authenticated, team-scoped
- [ ] 2.4 Send `Cache-Control`/`ETag` -- pre-existing gap, out of scope here. Exclusion of emails/phones holds for the new birthday entries (name + date only).

## 3. Frontend
- [x] 3.1 `useCalExportActions.ts` + calExport sheet: added a content-selection UI (toggle rows per event type + birthdays) persisting immediately via the selection endpoint (optimistic update, reverts + toasts on failure). Real feed URL, copy action, per-platform instructions, and "regenerate link" were already shipped. **Not done**: a `webcal://` deep-link action -- pre-existing gap, out of scope here.
- [ ] 3.2 Remove the dead prototype URL/`calPrototypeNote` -- not found in the current codebase (already removed by `add-calendar-feed-subscription`); no `.ics` download removal was in scope for this change.
- [x] 3.3 `de`/`en` string updates for the content-selection UI; MSW handlers added for the settings get/put endpoints (token get/rotate handlers already existed)

## 4. Ops/docs
- [ ] 4.1 Document the feed URL's secret/revocable nature and subscribe steps -- pre-existing gap (not written by `add-calendar-feed-subscription` either); out of scope for this change, which only adds content selection.

## 5. Verification
- [x] 5.1 `make generate`/`make generate-ts` re-run clean (no drift); the feed route's RBAC exclusion is pre-existing and already covered
- [x] 5.2 Backend tests: content selection filters types + toggles birthdays (default = everything on no active token); birthdays gated on `members` read access separately from `events` read access; rotate carries the previous selection forward; invalid type rejected
- [x] 5.3 `golangci-lint` + `go test ./...` green (govulncheck not run in this sandbox); migration is additive/reversible (`ADD COLUMN` / `DROP COLUMN`)
- [x] 5.4 Frontend lint/typecheck/test/build green
- [ ] 5.5 Manual: subscribe the feed in a real calendar app -- not feasible in this sandboxed environment; unit/integration coverage substitutes
