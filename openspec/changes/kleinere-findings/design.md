## Context

Thirteen independent findings surfaced from live usage. They don't share a subsystem, so each gets its own capability spec delta and task group under this one change folder rather than either (a) thirteen separate change proposals for what are mostly small diffs, or (b) one undifferentiated task list that would be hard to archive piecemeal. Seven are small enough to implement immediately (frontend-only, no schema/API change); six require a migration and/or a new/changed OpenAPI operation and are left for a follow-up pass, each already scoped below so that pass can start directly from `tasks.md` instead of re-discovering the affected files.

## Goals / Non-Goals

**Goals:**
- Fix the notification "undefined" bug and the mislabeled account entry point — both are visible defects, not preferences.
- Ship the frontend-only, no-schema-change items in one pass: location autocomplete, poll delete-dialog naming, poll voter transparency, news link detection, sport-neutral wording.
- Scope (file paths, complexity) the six deferred items precisely enough that implementing them later doesn't require re-reading the whole codebase.

**Non-Goals:**
- Implementing the six deferred items now (recurrence end date, calendar birthdays, RSVP deadline/countdown, penalty date+note, absence table, image proxy). Each needs a migration and/or an OpenAPI change, and rushing a schema decision to fit inside this batch would be worse than sequencing it as its own follow-up.
- Reworking the mock/demo seed data (`frontend/src/mocks/db.ts`) even though it's dance/formation-themed (item 7's sibling) — it's example content for the demo, not a default a real user is steered toward, and ~35 files (many with hardcoded string assertions) reference it. Only the two i18n strings a real user actually sees by default (event-type label, title placeholder) were reworded.

## Decisions

- **Notification line2** (`NotificationsSheet.tsx`): read `n.eventTitle` before falling back to `n.title`, since `events/service.go`'s `NotificationArgs` sets `EventTitle` for `event_created`/`event_cancelled` and leaves `Title` null — the frontend was reading the wrong field, not a backend bug. Extracted `eventNotifLine2()` as a standalone function (rather than inlining more branches into `notifMeta`) to keep that function's cyclomatic complexity under the ESLint threshold.
- **Location autocomplete**: sourced from `useEventsQuery` (the same query key the events list/calendar already populate) via a native `<datalist>`, not a new endpoint — free-solo text input stays a plain `<input>`, so no new validation or persistence model, and the suggestion list is naturally team-scoped and stays in sync with real usage.
- **Account label**: renamed the i18n key itself (`shell.accountAndRoles` → `shell.myAccount`) rather than only its value, since a `roles`-adjacent key name inaccurately describing a roles-free sheet is exactly the kind of drift that caused this finding.
- **Poll delete dialog**: `removePoll(id)` → `removePoll(id, question)`, threading the poll's question text into the existing `t('polls.deleteConfirmMsg', { question })` interpolation (the `t()` layer already supports named placeholders — no new i18n mechanism needed).
- **Poll voters**: purely additive rendering in `PollsPage.tsx` — `PollOption.voters` was already typed, populated by the mock/MSW layer, and (per the accompanying backend scoping) already returned by the real API; anonymous polls already return `voters: []`, so the existing `anonymous` flag on the poll doubles as the render gate without a second lookup.
- **News link detection**: a dedicated `Linkify` component that renders text nodes and `<a>` elements from `String.split()` on a single-capture-group URL regex, not `dangerouslySetInnerHTML` — avoids any HTML-injection surface since untrusted news body text never becomes raw HTML.
- **Sport-neutral wording**: reworded only the two *default* strings every user sees (`eventType.auftritt`'s label, `events.fieldTitlePlaceholder`), keeping the `auftritt` enum key itself unchanged — renaming the enum would touch the DB check constraint (`00001_init.sql`), the generated OpenAPI types, and every test fixture for a copy-only fix.

## Deferred item scoping (for the follow-up pass)

- **Recurrence end date** (medium): `event_series` gains an end-date alternative to `repeat_weeks` (`backend/internal/db/migrations`, `events/{service,repository}.go`'s `CreateSeries`, `openapi.yaml`'s series body, `eventFormSchema.ts`/`EventFormSheet.tsx`'s repeat UI).
- **Calendar birthdays** (medium): no schema change — `members.birthday` is already returned by the members list. Needs client-side synthesis of yearly recurring pseudo-events merged into `EventCalendar.tsx`'s render range, gated by the same birthday-visibility rule as the member profile (`contactNote`).
- **RSVP deadline + countdown** (large): new `rsvp_deadline` column on `events`/`event_series`, `openapi.yaml` schema fields, an authorization-aware late-response check in the attendance write path (roles with a bypass), and a frontend countdown component that activates under 24h.
- **Penalty date + note** (small-medium): `penalty_assignments.date` already exists but `CreateAssignment` never accepts it; add a `note TEXT` column, thread date+note through `finances/{repository,service,handler}.go`, `openapi.yaml`'s assignment schemas, and `PenaltyAssignSheet.tsx`.
- **Absence table** (large): new repository query joining events × attendance (reusing `attendance.EffectiveStatusExpr`) for a per-row (member, event, date) result, a new `stats` endpoint, and a new tab + table component next to the existing quota view.
- **Image delivery proxy** (large): add a `Get(ctx, key) (io.ReadCloser, contentType, error)` method to the `storage.ObjectStore` interface (both `S3Store` and `FakeStore`), a config flag selecting redirect vs. proxy per deployment, and rewrite `GetTeamPhoto`/`GetTeamLogo`/`GetMemberPhoto` to stream when proxy mode is on.

## Risks / Trade-offs

- Bundling nine capabilities into one change folder is denser than the repo's usual one-capability-per-change pattern; mitigated by giving each its own spec delta folder and task group so `openspec archive` can still fold them cleanly, and by matching the existing precedent (`backend-quality-cleanup` bundled five unrelated fixes under one capability).
- Leaving six items unchecked in `tasks.md` means this change stays un-archivable until a follow-up pass closes them — same open-ended state as `adopt-tanstack-query`'s one remaining task, which is an accepted pattern in this repo.
