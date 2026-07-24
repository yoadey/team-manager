## 1. Notifications: fix "undefined" on event notifications
- [x] 1.1 `NotificationsSheet.tsx`: extracted `eventNotifLine2()`, reading `n.eventTitle` (populated by the backend for `event_created`/`event_cancelled`) before `n.title` (never populated for events), plus event date/note/actor, each segment omitted when absent
- [x] 1.2 Regression test: `event_created` notification with `eventTitle`/`eventDate` set and no `title` renders the event title + date, never the literal string "undefined"

## 2. Account navigation: label matches sheet content
- [x] 2.1 Renamed `shell.accountAndRoles` → `shell.myAccount` ("Konto & Rollen"/"Account & Roles" → "Mein Konto"/"My Account") in `de.ts`/`en.ts`; updated `AppShell.tsx`'s reference
- [x] 2.2 Reworded `sheet.profile` (the sheet title) to match — the sheet shows photo/name/email/color-scheme/language/delete-account, never roles

## 3. Sport-neutral default wording
- [x] 3.1 `eventType.auftritt` label: "Auftritt / Turnier" → "Wettkampf / Auftritt" (de), "Performance / Tournament" → "Competition / Performance" (en); `events.typeAuftritt` short label: "Auftritt" → "Wettkampf" (de), "Performance" → "Competition" (en)
- [x] 3.2 `events.fieldTitlePlaceholder`: "z. B. Lateinformation – Training" → "z. B. Wöchentliches Training" (de), "e.g. Latin formation – Training" → "e.g. Weekly training" (en)
- [x] 3.3 Updated the two `useCalExportActions.test.ts` assertions pinned to the old label strings

## 4. Events: location autocomplete
- [x] 4.1 `EventFormSheet.tsx`: `useEventsQuery` (same query key already used by the events list/calendar, so no extra request in the common case) feeds a deduplicated (case-insensitive), order-preserving list of past locations into a native `<datalist>` wired to the location `<input>` via `list=`
- [x] 4.2 Tests: datalist renders de-duplicated, non-empty past locations; `list` attribute wired on the input

## 5. Polls: delete dialog names the poll
- [x] 5.1 `removePoll(id)` → `removePoll(id, question)` (`usePollActions.ts`, `AppContext.tsx`'s type, `PollsPage.tsx`'s call site); `polls.deleteConfirmMsg` now interpolates `{question}`
- [x] 5.2 Updated existing tests for the new signature; added an assertion that the confirm message contains the poll's question

## 6. Polls: voter transparency for non-anonymous polls
- [x] 6.1 `PollsPage.tsx` renders each option's voter names (already typed/populated via `PollOption.voters`, already `[]` for anonymous polls) beneath the option list, gated on `!p.anonymous`
- [x] 6.2 Tests: voter names render for a non-anonymous poll's options; no voter names render for an anonymous poll

## 7. News: hyperlink auto-detection
- [x] 7.1 New `Linkify` component (`components/Linkify.tsx`): splits body text on a URL regex and renders `<a target="_blank" rel="noopener noreferrer">` for matches (no `dangerouslySetInnerHTML`); `www.`-prefixed matches get `https://` prepended in `href` only
- [x] 7.2 Wired into `NewsCard` (`components/cards.tsx`), used by both the News page and Home's compact card
- [x] 7.3 Tests: plain text unchanged, bare `https://` URL linked, `www.`-prefixed URL linked with `https://` href, multiple URLs in one body each linked

## 8. Events: recurrence end date (deferred — needs a migration + OpenAPI change)
- [ ] 8.1 `event_series`: add an end-date alternative to `repeat_weeks` (migration; `events/{service,repository}.go`'s `CreateSeries` branches on whichever is set)
- [ ] 8.2 `openapi.yaml`: sibling `endDate` field on the series-creation body; regenerate `internal/gen` + `frontend/src/api/*`
- [ ] 8.3 `eventFormSchema.ts`/`EventFormSheet.tsx`: toggle between "N weeks" and "until date" recurrence input

## 9. Events: birthdays in the calendar (deferred)
- [ ] 9.1 `EventCalendar.tsx`: synthesize yearly recurring pseudo-events from the members list's existing `birthday` field for the visible range
- [ ] 9.2 Gate visibility with the same rule as the member profile's birthday field (`contactNote`: visible to the Trainerteam only)

## 10. Events: configurable RSVP deadline + countdown (deferred — needs a migration + OpenAPI change)
- [ ] 10.1 Migration: `rsvp_deadline` (nullable timestamptz) on `events` and `event_series`
- [ ] 10.2 `openapi.yaml`: request/response schema fields; regenerate `internal/gen` + `frontend/src/api/*`
- [ ] 10.3 Backend: attendance write path rejects a response after the deadline unless the caller's role bypasses it (admin-equivalent); `EventFormSheet.tsx` gains a deadline field
- [ ] 10.4 Frontend: event detail shows a countdown once under 24h remain until the deadline

## 11. Finances: penalty earned-date + note (deferred — needs a migration + OpenAPI change)
- [ ] 11.1 Migration: `note TEXT` on `penalty_assignments` (the `date` column already exists but `CreateAssignment` never accepts a caller-supplied value)
- [ ] 11.2 `finances/{repository,service,handler}.go`: `CreateAssignment` accepts an explicit date + optional note; `openapi.yaml`'s assignment create/response schemas; regenerate clients
- [ ] 11.3 `PenaltyAssignSheet.tsx`: date picker + optional note field

## 12. Statistics: absence table tab (deferred — needs a new endpoint)
- [ ] 12.1 New `stats` repository query: per-row (member, event, date) for events×attendance where the effective status is absent (reuses `attendance.EffectiveStatusExpr`)
- [ ] 12.2 New `openapi.yaml` operation + `stats/{service,handler}.go`; regenerate clients
- [ ] 12.3 `Stats.tsx`: a second tab with the absence table, alongside the existing per-person quota view

## 13. Image delivery: backend proxy option (deferred — needs an interface + config change)
- [ ] 13.1 `storage.ObjectStore`: add `Get(ctx, key) (io.ReadCloser, contentType string, err error)` on both `S3Store` and `FakeStore`
- [ ] 13.2 Config flag selecting redirect (current default) vs. proxy delivery per deployment
- [ ] 13.3 `teams.GetTeamPhoto`/`GetTeamLogo`, member photo handler: stream via `Get` instead of a 302 redirect when proxy mode is on; `openapi.yaml`'s 200-binary response alternative to the 302

## 14. Verification
- [x] 14.1 Frontend: `npm run typecheck`, `npm test` (1169 tests), `npm run lint` (0 warnings), `npm run build` + `npm run check:bundle` (256.5 KB / 600 KB budget) all green for groups 1–7
- [ ] 14.2 Once groups 8–13 are implemented: `make generate`/`make generate-ts` produce only the intended diff; backend `go test ./... -short` + `golangci-lint run ./...`; migration-safety + migration-rollback CI gates; frontend gates re-run for any touched component
