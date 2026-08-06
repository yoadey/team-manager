## 1. OpenAPI contract

- [x] 1.1 `openapi.yaml`: add `eventReminderEnabled` (boolean) and
      `eventReminderHoursBefore` (integer, minimum 1, maximum 72) to
      `PushCategoryPreferences`, both required, matching the existing
      boolean fields' style
- [x] 1.2 `make generate` (backend) and `make generate-ts` (frontend),
      commit the regenerated output

## 2. Database

- [x] 2.1 New migration: `push_preferences` gets
      `event_reminder_enabled BOOLEAN NOT NULL DEFAULT true` and
      `event_reminder_hours_before SMALLINT NOT NULL DEFAULT 6 CHECK
      (event_reminder_hours_before BETWEEN 1 AND 72)`
- [x] 2.2 Same migration: new `event_reminders_sent` table
      (`event_id UUID REFERENCES events(id) ON DELETE CASCADE`,
      `user_id UUID REFERENCES users(id) ON DELETE CASCADE`, `sent_at
      TIMESTAMPTZ NOT NULL DEFAULT now()`, `PRIMARY KEY (event_id,
      user_id)`)
- [x] 2.3 Migration rollback (`Down`) drops both cleanly

## 3. Backend — preferences

- [x] 3.1 `push.CategoryPreferences`: add `EventReminderEnabled bool`,
      `EventReminderHoursBefore int16`; `DefaultCategoryPreferences`
      defaults them to `true`/`6`
- [x] 3.2 `push.Repository.GetPreferences`/`UpsertPreferences`: read/write
      the two new columns
- [x] 3.3 `push.Repository`: new `ListForTeam(ctx, teamID)` (current
      members' subscriptions, no exclusion — mirrors
      `ListForTeamExcludingUser` minus the exclude clause)
- [x] 3.4 `push.Handler`: `toGenPreferences` and `SetPushPreferences` carry
      the two new fields through
- [x] 3.5 Reject an out-of-range `eventReminderHoursBefore` (not 1–72) with
      400 in the handler, ahead of the DB `CHECK` constraint

## 4. Backend — reminder job

- [x] 4.1 `events.Repository`: new method listing non-cancelled events
      whose `date` falls within a bounded forward window (now-1d .. now+4d)
      across all teams
- [x] 4.2 `jobs.EventReminderWorker` (new `event_reminder_worker.go`):
      for each candidate event, compute `events.EventStartInstant`; for
      each team member (via `push.Repository.ListForTeam`) check
      `events` module read permission, then `EventReminderEnabled` +
      whether `now` has just entered `[start-hoursBefore, start)`; insert
      into `event_reminders_sent` (`ON CONFLICT DO NOTHING`); on an actual
      insert, enqueue a `PushDeliveryArgs` job
- [x] 4.3 `jobs.NewClient`: register `EventReminderWorker` as a
      `river.PeriodicJob` on a 5-minute interval (`RunOnStart: false`,
      matching `RetentionArgs`' pattern)
- [x] 4.4 `cmd/server/main.go`: construct `EventReminderWorker` with the
      already-available `pool`/`membersRepo`/`pushRepo`/`eventsRepo`
- [x] 4.5 Unit tests: due/not-due boundary, disabled preference, `none`
      permission, already-sent idempotency, cancelled event excluded

## 5. Frontend

- [x] 5.1 `features/notifications/types.ts`: extend
      `PushCategoryPreferences` with the two new fields
- [x] 5.2 `usePushPreferencesActions.ts`: extend `DEFAULT_PREFERENCES`;
      add a setter for the hours value (distinct from the boolean
      `setCategory` toggle)
- [x] 5.3 `NotificationsPanel.tsx`: new toggle + bounded (1–72) numeric
      input under the existing category list, shown only when push is
      subscribed (same gate as the category panel)
- [x] 5.4 i18n (`de`/`en`): new strings for the toggle label, description,
      and hours input
- [x] 5.5 `mocks/db.ts` / `mocks/handlers.ts`: mock parity so
      `serviceContract.test.ts` keeps both implementations in sync

## 6. Verification

- [x] 6.1 `cd backend && make generate` — no diff (drift check)
- [x] 6.2 `cd backend && make test` green
- [x] 6.3 `cd backend && make lint` green
- [x] 6.4 `cd frontend && npm run generate` (or repo-root `make
      generate-ts`) — no diff
- [x] 6.5 `cd frontend && npm run lint && npm run typecheck && npm test`
      green
- [x] 6.6 `openspec validate event-reminder-push-notifications --strict`
      passes
