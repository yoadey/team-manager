## 1. Database
- [ ] 1.1 New migration `00031_calendar_feed_tokens.sql`: `calendar_feed_tokens`
      (`id UUID PK`, `user_id UUID FK -> users ON DELETE CASCADE`,
      `team_id UUID FK -> teams ON DELETE CASCADE`, `token TEXT NOT NULL UNIQUE`,
      `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `revoked_at TIMESTAMPTZ`)
- [ ] 1.2 Unique index on `(user_id, team_id) WHERE revoked_at IS NULL`
      (enforces "one active token per user+team" at the DB layer) and an
      index on `token` for the feed lookup
- [ ] 1.3 `make migrate` locally; confirm `migration-rollback` (up→down→up)
      and `migration-safety` gates pass

## 2. OpenAPI
- [ ] 2.1 `POST /teams/{teamId}/calendar-feed/token` (`x-rbac-module: events`,
      `x-rbac-self-service: true`) returning `{ url: string }`;
      `DELETE /teams/{teamId}/calendar-feed/token` (same RBAC) returning 204
- [ ] 2.2 `GET /calendar-feed/{token}.ics` — `security: []`, response
      `text/calendar`, no `x-rbac-module` (unauthenticated, outside the RBAC
      table entirely, same as `/auth/*`)
- [ ] 2.3 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 2.4 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: calendarfeed package
- [ ] 3.1 `internal/calendarfeed/model.go`: `TokenRow`
- [ ] 3.2 `internal/calendarfeed/repository.go`: `IssueToken(ctx, userID,
      teamID) (token string, err error)` (revokes any existing active row for
      that pair, inserts a fresh 32-byte `crypto/rand`+hex token — mirror
      `teams.generateCode` in `internal/teams/repository.go`), `Revoke(ctx,
      userID, teamID) error`, `FindActiveByToken(ctx, token) (*TokenRow, error)`
- [ ] 3.3 `internal/calendarfeed/service.go`: `IssueToken`/`RevokeToken`
      (thin wrappers over the repo, building the full URL from
      `config.PublicBaseURL`), `ServeFeed(ctx, token) (icsBytes []byte, err
      error)` — resolves the token, re-checks membership + `events` read
      permission via the same `members.Repository` methods
      `middleware.RequireMembership`/`RequirePermission` use, loads that
      team's non-cancelled events, renders ICS; returns a distinct
      not-found-or-unauthorized error (mapped to plain 404, matching the
      design decision not to leak token validity)
- [ ] 3.4 `internal/calendarfeed/ics.go`: Go port of
      `useCalExportActions.ts::buildIcs()` — `BEGIN:VCALENDAR`/`VEVENT`
      structure, line folding at 73 chars, the same escaping rules
      (backslash/comma/semicolon/newline), `UID: {eventID}@teamverwaltung.app`,
      `DTSTAMP`, `Europe/Berlin`-anchored `DTSTART`/`DTEND` (reuse or port the
      `zonedTimeToUtc`-equivalent conversion in Go, keeping `18:00` /
      2-hour-duration fallbacks identical to the frontend's)
- [ ] 3.5 `internal/calendarfeed/handler.go`: three HTTP handlers — the two
      authenticated token-management ones plug into the generated
      `StrictServerInterface` like other features; the unauthenticated feed
      handler is a plain `http.HandlerFunc` (no `gen`/strict-server wrapper,
      matching `strictSrv.Login`'s existing pattern)

## 4. Router wiring
- [ ] 4.1 `cmd/server/main.go`: register
      `r.Get("/calendar-feed/{token}.ics", calendarFeedHandler)` after the
      generated mux inside `r.Route("/api/v1", ...)`, alongside the existing
      `/auth/*` manual registrations, so it overrides whatever the generated
      mux registered for that path and never passes through
      `AuthMiddleware`/`RequireMembership`/`RequirePermission`

## 5. Frontend
- [ ] 5.1 `features/events/hooks/useCalExportActions.ts`: `copyCalUrl` calls
      `POST /teams/{teamId}/calendar-feed/token` and copies the real returned
      URL instead of formatting a placeholder; add a `regenerateCalUrl`
      (or reuse `copyCalUrl` idempotently) for the "renew link" action
- [ ] 5.2 `features/events/components/CalExportSheet.tsx`: replace the
      hardcoded `url` with the fetched one, remove `t('events.calPrototypeNote')`
      once wired
- [ ] 5.3 `services/serviceLayerReal.ts`: `events.issueCalendarFeedToken` /
      `revokeCalendarFeedToken`
- [ ] 5.4 `mocks/handlers.ts` + `mocks/db.ts`: MSW handlers for the two
      token-management routes (the `.ics` feed route itself is
      backend-only and out of scope for the mock service layer, same as
      other server-rendered-content routes)
- [ ] 5.5 `i18n/en.ts` + `i18n/de.ts`: drop or repurpose `calPrototypeNote`;
      add copy for the "renew link" action if introduced

## 6. Tests
- [ ] 6.1 Backend: `calendarfeed.Service.IssueToken` (rotates old token,
      one active per user+team), `RevokeToken`, `ServeFeed` (valid token;
      revoked token → 404; token for a team the user left → 404; token
      whose `events` permission dropped to `none` → 404; ICS output
      excludes cancelled events and matches expected escaping/folding);
      `ics.go` rendering unit tests (line folding at 73 chars, special-char
      escaping, DST-boundary date around a `Europe/Berlin` transition)
- [ ] 6.2 Integration test hitting `GET /calendar-feed/{token}.ics` with no
      cookie at all, confirming it's reachable unauthenticated and returns
      `text/calendar`
- [ ] 6.3 Frontend: `CalExportSheet`/`useCalExportActions` wired to the real
      endpoint; `serviceContract.test.ts` new scenarios

## 7. Verification
- [ ] 7.1 `openspec validate add-calendar-feed-subscription --strict`
- [ ] 7.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [ ] 7.3 `cd backend && make lint`
- [ ] 7.4 `cd backend && make test` (unit + integration)
- [ ] 7.5 `govulncheck`
- [ ] 7.6 `migration-rollback` / `migration-safety` on the new migration
- [ ] 7.7 `backend-openapi-drift`
- [ ] 7.8 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
- [ ] 7.9 Manual: subscribe the issued URL in a real calendar app (e.g.
      Google Calendar "From URL" or Apple Calendar), confirm events appear
      and a later-created event shows up after the client's next poll;
      revoke the token and confirm the app's next poll fails/stops updating
