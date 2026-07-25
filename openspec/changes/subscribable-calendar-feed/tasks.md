## 1. Spec
- [ ] 1.1 Add a token-authenticated feed endpoint returning `text/calendar` (`GET /calendar/{token}.ics`), outside session-auth/RBAC, to `openapi.yaml`
- [ ] 1.2 Add endpoints to get + rotate the feed token and to read + update the feed's content selection (event-type subset + include-birthdays), session-authenticated, team-scoped
- [ ] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Migration + backend
- [ ] 2.1 Migration: `calendar_feed_tokens(membership_id, token_hash, created_at, types text[], include_birthdays bool)`, unique per membership; rotate replaces the hash; default selection = all types + birthdays on
- [ ] 2.2 Feed route bypasses session auth: resolve the presented token by hash → (user, team); reject unknown/rotated tokens; rate-limit and keep it off the RBAC table's fast path deliberately
- [ ] 2.3 Server-side `.ics` builder (`internal/calendar`): team events (all types) + member birthdays (yearly all-day), `VTIMEZONE` Europe/Berlin, serialized only for what the resolved member may see, filtered to the feed's selected types / birthday toggle
- [ ] 2.4b Endpoint to read + update the feed's content selection (types subset + include-birthdays), session-authenticated, team-scoped
- [ ] 2.4 Send `Cache-Control`/`ETag`; exclude emails/phones and anything beyond title/type/time/location + birthdays

## 3. Frontend
- [ ] 3.1 `useCalExportActions.ts` + calExport sheet: real feed URL, copy + `webcal://` actions, per-platform instructions (existing i18n keys), a "regenerate link" action, and a content-selection UI (checkboxes per event type + birthdays) persisting via the selection endpoint
- [ ] 3.2 Keep the one-time `.ics` download as a secondary option; remove the dead prototype URL/`calPrototypeNote`
- [ ] 3.3 `de`/`en` string updates; MSW handlers for token get/rotate + a sample feed

## 4. Ops/docs
- [ ] 4.1 Document the feed URL as a secret, revocable bearer link, its refresh/cache behavior, and the Google/Apple/Outlook subscribe steps in `docs/`

## 5. Verification
- [ ] 5.1 openapi-drift green; the feed route is intentionally excluded from RBAC gating and that's covered by a test
- [ ] 5.2 Backend tests: valid token → correct feed (events + birthdays, timezone-correct); unknown/rotated token → rejected; feed exposes no emails/phones; rate-limit applies; visibility matches the member's permissions; content selection filters types + toggles birthdays (default = everything)
- [ ] 5.3 `golangci-lint` + `go test ./...` + govulncheck green; migration-rollback + migration-safety green
- [ ] 5.4 Frontend lint/typecheck/test/build + bundle budget green
- [ ] 5.5 Manual: subscribe the feed in Google Calendar, Apple Calendar, and Outlook; confirm events + birthdays appear at the right times and updates propagate
