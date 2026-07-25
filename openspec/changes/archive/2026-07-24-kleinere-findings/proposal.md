## Why

A batch of smaller usability/correctness findings accumulated from real usage (screenshots + notes), spanning events, notifications, account navigation, wording, finances, statistics, image delivery, and polls/news. None individually justifies its own change, but several are outright bugs (a notification literally rendering the string "undefined") and the rest are small-to-medium UX gaps. Bundling them avoids thirteen near-empty change folders while still tracking each as its own numbered task group with its own spec delta, per capability.

Concretely:

1. Event recurrence only takes an occurrence count (`repeatWeeks`), not an end date.
2. Member birthdays (`members.birthday`, already stored) are never surfaced in the calendar.
3. There is no configurable RSVP deadline for events, so there's also no "less than 24h left" countdown.
4. There is no location autocomplete when creating/editing an event — every location is retyped from scratch.
5. The bottom-left profile button and its sheet title read "Konto & Rollen" ("Account & Roles"), but that sheet only shows account settings (photo, name, email, color scheme, language, delete account) — no roles.
6. `NotificationsSheet`'s event_created/event_cancelled line renders the literal string `"undefined"` instead of the event title, because the backend enqueues `EventTitle` while the frontend read `n.title` (never populated for event notifications) — see the attached screenshot.
7. Several user-facing strings (event-type label "Auftritt / Turnier", the title-field placeholder "Lateinformation – Training") assume a dance/formation club, not a generic sports club.
8. Penalty assignments (`penalty_assignments`) already have a `date` column but no way to set it or attach a note explaining what happened — every penalty is silently dated "today" with no context.
9. `Stats` only ever shows the per-person attendance quota; there is no table of who missed which training and when.
10. Team/user/member photos are delivered as a 302 redirect straight to a presigned S3 URL (`move-images-to-object-storage`'s design), which breaks when the object store is only reachable inside the cluster network, not from the browser.
11. The poll-delete confirm dialog ("Umfrage löschen?" / "Diese Umfrage … wird entfernt.") never names which poll is about to be deleted.
12. Non-anonymous polls already compute and ship per-option voter lists (`PollOption.voters`) but the frontend never renders them — nobody can actually see who voted for what.
13. News body text is rendered as plain text; a pasted URL is not clickable.

## What Changes

- **Events — recurrence end date** (item 1): let a recurring series be defined by an end date as an alternative to a repeat count.
- **Events — birthdays in the calendar** (item 2): synthesize yearly recurring birthday entries into the calendar view from existing member data, respecting the same birthday-visibility rule as the member profile.
- **Events — RSVP deadline + countdown** (item 3): add an optional per-event response deadline; block non-privileged responses after it passes; show a countdown once under 24h remaining.
- **Events — location autocomplete** (item 4, done in this change): suggest previously used event locations (deduplicated) via a native `<datalist>` sourced from the already-loaded events list — no new backend endpoint.
- **Account navigation label** (item 5, done in this change): rename "Konto & Rollen"/"Account & Roles" to "Mein Konto"/"My Account" (button subtitle + sheet title), matching what the sheet actually contains.
- **Notification clarity** (item 6, done in this change): event notifications now show the event title + date instead of the literal string "undefined".
- **Sport-neutral wording** (item 7, done in this change): reword the "Auftritt" event-type label and title-field placeholder so they don't presuppose a dance/formation club.
- **Penalty audit detail** (item 8): let a penalty assignment carry an explicit earned-date (already a DB column, currently unsettable) and an optional note.
- **Attendance absence table** (item 9): add a "Fehlzeiten" tab next to the per-person quota view, listing who missed which event and when.
- **Image delivery proxy** (item 10): add a backend-streaming delivery mode for team/user photos and logos as a configurable alternative to the current presigned-redirect, for cluster-internal object stores.
- **Poll delete dialog** (item 11, done in this change): the confirm dialog now names the poll being deleted.
- **Poll voter transparency** (item 12, done in this change): render each option's voter names for non-anonymous polls.
- **News link detection** (item 13, done in this change): auto-linkify bare URLs in news body text on render.

## Capabilities

### New Capabilities
- `event-experience`: recurrence-by-end-date, calendar birthdays, RSVP deadlines with a countdown, and location autocomplete for event creation/editing.
- `notification-clarity`: notification lines always show a name/date, never a literal "undefined".
- `account-navigation-labels`: the profile entry point's label matches what it actually contains.
- `sport-neutral-content`: default/example UI copy doesn't presuppose one sport (dance/formation).
- `penalty-audit-detail`: penalty assignments can carry an explicit earned-date and an optional note.
- `attendance-absence-table`: a per-event, per-member absence table alongside the existing quota view.
- `image-delivery-proxy`: team/user photo and logo delivery can be proxied through the backend instead of redirecting to a presigned object-store URL.
- `poll-transparency`: the delete-confirmation dialog names the poll, and non-anonymous polls show voter names per option.
- `news-link-detection`: bare URLs in news body text render as clickable links.

### Modified Capabilities
<!-- none -->

## Impact

- **Frontend-only, done in this change**: `frontend/src/features/events/components/EventFormSheet.tsx` (+`.test.tsx`, location `<datalist>`), `frontend/src/i18n/{de,en}.ts` (account label, event-type wording), `frontend/src/layouts/AppShell.tsx`, `frontend/src/features/notifications/components/NotificationsSheet.tsx` (+`.test.tsx`), `frontend/src/features/polls/{PollsPage.tsx,hooks/usePollActions.ts}` (+tests), `frontend/src/context/AppContext.tsx` (`removePoll` signature), `frontend/src/components/{Linkify.tsx (new),cards.tsx}` (+test), `frontend/src/features/events/hooks/useCalExportActions.test.ts` (updated fixture strings).
- **Deferred, larger — backend + spec + frontend**: `backend/openapi/openapi.yaml` + regenerated `internal/gen`/`frontend/src/api/*` (items 1, 3, 8, 9, 10), new migrations (items 1 or 8 optionally, 3, 9's read path reuses existing tables, 8's `note` column), `backend/internal/{events,finances,stats,storage,teams,members}/*`, corresponding frontend feature components/hooks.
- CI: the frontend-only items stay within the existing frontend gates (lint, typecheck, test, build, check:bundle) — no backend/openapi touch, no `make generate`/`make generate-ts` needed. The deferred items are **API-affecting** and will need their own `make generate`/`make generate-ts` pass plus migration-safety/migration-rollback CI gates when implemented.
