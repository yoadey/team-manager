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
- [x] 8.1 Migration `00006_event_series_end_date.sql`: `event_series.repeat_end_date DATE`; `events/{service,repository}.go`'s `CreateSeries`/new `seriesDates()` helper branches on whichever of `repeat_weeks`/`repeat_end_date` is set (capped at `maxRepeatWeeks`)
- [x] 8.2 `openapi.yaml`: sibling `endDate` field on the series-creation body; regenerated `internal/gen` + `frontend/src/api/*`
- [x] 8.3 `eventFormSchema.ts`/`EventFormSheet.tsx`: `RepeatModeSelector` toggle between "N weeks" and "until date" recurrence input

## 9. Events: birthdays in the calendar (deferred)
- [x] 9.1 `EventCalendar.tsx`: synthesize yearly recurring pseudo-events from the members list's existing `birthday` field for the visible range (extracted as pure `synthesizeBirthdayEvents.ts` for testability)
- [x] 9.2 Gate visibility — `contactNote` turned out to be i18n copy only, not backed by an actual read-time gate anywhere in the codebase (birthday is otherwise unconditionally shown in `MemberDetailSheet`); gated calendar birthday entries on `app.can('members', 'write')`, the permission level the copy's "Trainerteam only" promise conceptually maps to. This is a new, calendar-specific gate — flagged here for awareness in case the underlying `contactNote` promise should also be enforced on the member detail view itself, which is out of this change's scope.

## 10. Events: configurable RSVP deadline + countdown (deferred — needs a migration + OpenAPI change)
- [x] 10.1 Migration `00007_event_rsvp_deadline.sql`: nullable `rsvp_deadline timestamptz` on `events` and `event_series`
- [x] 10.2 `openapi.yaml`: `rsvpDeadline` request/response schema fields; regenerated `internal/gen` + `frontend/src/api/*`
- [x] 10.3 Backend: `SetAttendance` rejects a late response (`ErrRsvpDeadlinePassed`, mapped to 409) unless the caller holds `events:write` (re-checked atomically inside the write via a `caller_write` CTE, closing the same TOCTOU window the cancelled-status check already closes); `EventFormSheet.tsx` gains a `datetime-local` deadline field
- [x] 10.4 Frontend: new `RsvpCountdown.tsx`, wired into `EventDetailSheet.tsx`, shown once under 24h remain until the deadline

## 11. Finances: penalty earned-date + note (deferred — needs a migration + OpenAPI change)
- [x] 11.1 Migration `00008_penalty_assignment_note.sql`: nullable `note TEXT` on `penalty_assignments` (the `date` column already existed but `CreateAssignment` never accepted a caller-supplied value)
- [x] 11.2 `finances/{repository,service,handler}.go`: `CreateAssignment` accepts an explicit date + optional note; `openapi.yaml`'s assignment create/response schemas; regenerated clients
- [x] 11.3 `PenaltyAssignSheet.tsx`: date picker (defaulting to today, editable to the past) + optional note field

## 12. Statistics: absence table tab (deferred — needs a new endpoint)
- [x] 12.1 New `stats` repository query: per-row (member, event, date) for events×attendance where the effective status is absent (reuses `attendance.EffectiveStatusExpr`)
- [x] 12.2 New `GET /teams/{teamId}/stats/absences` operation (`x-rbac-module: events`) + `stats/{service,handler}.go`; regenerated clients
- [x] 12.3 `Stats.tsx`: a second tab ("Fehlzeiten") with the absence table, alongside the existing per-person quota view; empty state when no absences in range

## 13. Image delivery: backend proxy option (deferred — needs an interface + config change)
- [x] 13.1 `storage.ObjectStore`: added `Get(ctx, key) (io.ReadCloser, contentType string, err error)` on both `S3Store` and `FakeStore`
- [x] 13.2 `IMAGE_DELIVERY_PROXY_ENABLED` config flag (default `false`) selecting redirect (current default) vs. proxy delivery per deployment
- [x] 13.3 `teams.GetTeamPhoto`/`GetTeamLogo`, member photo handler: stream via `Get` instead of a 302 redirect when proxy mode is on; `openapi.yaml`'s 200-binary response alternative added to the 302; access-control check still runs before any bytes are streamed

## 14. Verification
- [x] 14.1 Frontend: `npm run typecheck`, `npm test` (1169 tests), `npm run lint` (0 warnings), `npm run build` + `npm run check:bundle` (256.5 KB / 600 KB budget) all green for groups 1–7
- [x] 14.2 Groups 8–13 implemented and integrated: `make generate`/`make generate-ts` produce no diff beyond the intended changes (verified after merging all six agents' work); backend `go build ./...` clean, `go test ./... -short -race` all packages `ok`, `golangci-lint run ./...` 0 issues; frontend `npm run typecheck`/`npm test -- --run` (1239 tests)/`npm run lint` (0 warnings)/`npm run build`/`npm run check:bundle` (266.0 KB / 600 KB budget) all green. Migration-safety + migration-rollback CI gates could not be exercised locally — no Docker daemon in this sandbox for testcontainers/`make test-integration`; left for CI to confirm on migrations `00006`–`00008`.
