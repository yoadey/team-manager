## 1. Database
- [x] 1.1 New migration `00003_calendar_feed_tokens.sql` (renumbered from
      `00031` after `main` squashed all prior migrations into a single
      `00001_init.sql`): `calendar_feed_tokens`
      (`id UUID PK`, `user_id UUID FK -> users ON DELETE CASCADE`,
      `team_id UUID FK -> teams ON DELETE CASCADE`, `token TEXT NOT NULL UNIQUE`,
      `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `revoked_at TIMESTAMPTZ`)
- [x] 1.2 Unique index on `(user_id, team_id) WHERE revoked_at IS NULL`
      and an index on `token` (the latter comes free from the
      `UNIQUE(token)` column constraint, no separate index needed)
- [x] 1.3 `make migrate` locally against a real Postgres; confirmed
      up→down→up round-trips cleanly

## 2. OpenAPI
- [x] 2.1 `POST /teams/{teamId}/calendar-feed/token` (`x-rbac-module: events`,
      `x-rbac-self-service: true`) returning `{ url: string }`;
      `DELETE /teams/{teamId}/calendar-feed/token` (same RBAC) returning 204
- [x] 2.2 `GET /calendar-feed/{token}.ics` — `security: []`, response
      `text/calendar`, no `x-rbac-module`. oapi-codegen generated the
      expected raw-bytes strict-response type
      (`GetCalendarFeed200TextcalendarResponse{Body io.Reader, ContentLength
      int64}`) automatically for the non-JSON content type — no custom
      codegen config needed.
- [x] 2.3 `cd backend && make generate`
- [x] 2.4 repo-root `make generate-ts`

## 3. Backend: calendarfeed package
- [x] 3.1 `internal/calendarfeed/model.go`: `TokenRow`
- [x] 3.2 `internal/calendarfeed/repository.go`: `IssueToken` (transactional
      revoke-then-insert, 32-byte `crypto/rand`+hex token mirroring
      `teams.generateCode`), `Revoke`, `FindActiveByToken`
- [x] 3.3 `internal/calendarfeed/service.go`: `IssueToken`/`RevokeToken`
      (URL built from `config.PublicBaseURL`), `ServeFeed` — re-checks
      membership + `events` read permission on every call via the same
      `members.Repository` (passed in as `membershipChecker`/`permsChecker`
      interfaces), reuses `notifications.HasReadAccess` for the permission
      check itself, returns `ErrFeedUnavailable` uniformly (token unknown,
      revoked, team left, or permission dropped to `none` all look
      identical to the caller)
- [x] 3.4 `internal/calendarfeed/ics.go`: Go port of `buildIcs()` — line
      folding at 73 octets, the same escaping rules, stable
      `UID: {eventID}@teamverwaltung.app`, `DTSTAMP`, `Europe/Berlin`-anchored
      `DTSTART`/`DTEND` via `time.LoadLocation` (simpler and more correct
      than porting the frontend's manual offset-math `zonedTimeToUtc` —
      Go's stdlib does IANA-zone-aware wall-clock-to-UTC conversion
      natively), same `18:00`/2-hour-duration fallbacks
- [x] 3.5 `internal/calendarfeed/handler.go`: `IssueCalendarFeedToken`/
      `RevokeCalendarFeedToken`/`GetCalendarFeed`, all three as ordinary
      `gen.StrictServerInterface` methods (not a bespoke `http.HandlerFunc`
      as originally planned) — the strict-server adapter already generated
      a working type for the `text/calendar` response, so reusing it end to
      end (same request/response marshaling as every other route) turned
      out simpler than a hand-rolled handler.

## 4. Router wiring
- [x] 4.1 `cmd/server/main.go`: registered
      `r.Get("/calendar-feed/{token}.ics", func(w, req) {
      strictSrv.GetCalendarFeed(w, req, chi.URLParam(req, "token")) })`
      after the generated mux inside `r.Route("/api/v1", ...)`, alongside
      the `/auth/*` manual overrides. Verified live (see 7.9) that chi's
      `{token}.ics` compound path segment correctly extracts just the
      token, and that the route is reachable with zero cookies.

## 5. Frontend
- [x] 5.1 `features/events/hooks/useCalExportActions.ts`: `copyCalUrl` now
      takes the URL as a parameter (fetched by the sheet via a new
      `useCalendarFeedUrlQuery`, `staleTime: Infinity` so reopening the
      sheet doesn't silently rotate the token); added `regenerateCalUrl`
      for the explicit "renew link" action
- [x] 5.2 `features/events/components/CalExportSheet.tsx`: shows the fetched
      URL (loading/error states while the query resolves), removed
      `t('events.calPrototypeNote')`, added a "renew link" button
- [x] 5.3 `services/serviceLayerReal.ts`: `events.issueCalendarFeedToken` /
      `revokeCalendarFeedToken`
- [x] 5.4 `mocks/handlers.ts` + `mocks/db.ts`: MSW handlers for the two
      token-management routes
- [x] 5.5 `i18n/en.ts` + `i18n/de.ts`: dropped `calPrototypeNote`, added
      `calLoading`/`calLoadFailed`/`calRenew`/`calRenewFailed`/
      `toastCalLinkRenewed`, updated `calSubscribeDesc` to drop the
      "not active in this preview" caveat now that it's real

## 6. Tests
- [x] 6.1 Backend: `calendarfeed.Service.IssueToken`/`RevokeToken`/`ServeFeed`
      (unknown/revoked token, left-team, permission-dropped-to-none, team
      gone, happy path) with mocked deps; `ics.go` rendering tests (cancelled
      exclusion, VCALENDAR structure, DST-boundary CET/CEST date pair,
      escaping, line folding); `calendarfeed.Repository` integration tests
      (rotate-invalidates-old-token, revoke, unknown-token) — need Docker,
      skip in this sandbox, additionally verified via a manual smoke script
      against a real local Postgres (issue → find → rotate → old token
      404s → revoke → 404s)
- [x] 6.2 Not an automated Go integration test (no existing precedent in
      `cmd/server` for booting the full assembled router in a test — only
      small pure-function unit tests exist there today); instead verified
      manually end-to-end against a real running server + real Postgres:
      logged in, issued a token, then fetched
      `GET /calendar-feed/{token}.ics` with `curl` sending **no cookie at
      all**, got back `200 text/calendar` with the correct ICS body; then
      revoked the token and confirmed the same URL now 404s. A follow-up
      change could add a proper `httptest`-based router-boot test harness
      for this class of regression (nothing currently exercises the
      "manual override route bypasses AuthMiddleware" wiring in an
      automated way, for `/auth/*` either).
- [x] 6.3 Frontend: `CalExportSheet.test.tsx`/`useCalExportActions.test.ts`
      updated for the real query-based flow (loading state, renew button,
      copy-with-URL); `serviceLayerReal.test.ts` new
      `issueCalendarFeedToken`/`revokeCalendarFeedToken` cases (not
      `serviceContract.test.ts`, which is scoped to cross-implementation
      drift scenarios from the pre-MSW mock era — see the sibling
      `add-web-push-notifications` proposal's tasks.md 8.2 for the same
      reasoning)

## 7. Verification
- [x] 7.1 `openspec validate add-calendar-feed-subscription --strict`
- [x] 7.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [x] 7.3 `cd backend && make lint` (golangci-lint: 0 issues)
- [x] 7.4 `cd backend && make test` — unit tests pass; integration tests
      skip cleanly (no Docker in this sandbox)
- [ ] 7.5 `govulncheck` — could not run in this sandbox (outbound proxy
      returns 403 for `vuln.go.dev`); needs to run in CI
- [x] 7.6 `migration-rollback` / `migration-safety` — exercised manually
      (`goose up`/`down`/`up` against a real local Postgres 16)
- [x] 7.7 `backend-openapi-drift` — confirmed no diff after regenerating
- [x] 7.8 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
      — all pass (1168 tests, 0 lint issues, bundle within budget)
- [ ] 7.9 Manual real-calendar-app subscribe walkthrough (Google Calendar
      "From URL" / Apple Calendar) — not performed: this sandbox has no
      such app to test against. What *was* verified end-to-end with `curl`
      against a real running server + real Postgres: issuing a token,
      fetching the feed with zero cookies and getting back a correct
      `text/calendar` document containing the seeded event, and confirming
      revocation immediately 404s the old URL. A real calendar-app pass is
      still needed before shipping (client-side ICS parsing quirks — e.g.
      how strictly a given app validates `DTSTAMP`/line-folding — can't be
      verified without one).
