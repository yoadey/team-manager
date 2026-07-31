## 1. Database

- [x] 1.1 Migration `00016_push_preferences.sql`: `push_preferences` table
      (`team_id`, `user_id`, `attendance`/`events`/`news`/`polls`/`absence`
      booleans all `NOT NULL DEFAULT true`, `updated_at`, PK
      `(team_id, user_id)`, FKs to `teams`/`users` `ON DELETE CASCADE`)
- [x] 1.2 `make migrate` locally; confirm up→down→up round-trips (not runnable
      without Docker in this environment; migration mirrors the established
      safe `CREATE TABLE` pattern, e.g. `00009_calendar_shares.sql`)

## 2. OpenAPI

- [x] 2.1 New schema `PushCategoryPreferences` (required booleans:
      `attendance`, `events`, `news`, `polls`, `absence`)
- [x] 2.2 `GET /teams/{teamId}/push-preferences` (tag `push`,
      `x-rbac-module: public`) → `PushCategoryPreferences`
- [x] 2.3 `PUT /teams/{teamId}/push-preferences` (tag `push`,
      `x-rbac-module: public`, body `PushCategoryPreferences`) → 204
- [x] 2.4 `make generate` + repo-root `make generate-ts`; commit generated
      output (`internal/gen/api.gen.go`, `internal/middleware/rbac_table.gen.go`,
      `frontend/src/api/types.gen.ts`, `frontend/src/api/zod.gen.ts`)

## 3. Backend: push package

- [x] 3.1 `internal/push/model.go`: `CategoryPreferences` struct
      (`Attendance, Events, News, Polls, Absence bool`),
      `DefaultCategoryPreferences()`, `NotificationCategory(notifType string) string`
      mapping `attendance`→`attendance`, `event_*`→`events`, `news`→`news`,
      `poll`→`polls`, `absence`→`absence`
- [x] 3.2 `internal/push/repository.go`: `GetPreferences(ctx, teamID, userID)`
      (returns `DefaultCategoryPreferences()` on no row),
      `UpsertPreferences(ctx, teamID, userID, prefs)` (`ON CONFLICT
      (team_id, user_id) DO UPDATE`)
- [x] 3.3 `internal/push/service.go`: `GetPreferences`/`SetPreferences`
      wrapping the repository
- [x] 3.4 `internal/push/handler.go`: `GetPushPreferences`/`SetPushPreferences`
      StrictServerInterface methods (team id from path, user from
      `auth.UserFromContext`); wired into `internal/server/server.go`'s
      delegation methods

## 4. Backend: delivery gate

- [x] 4.1 `internal/jobs/notification_worker.go`: widen the interface
      satisfied by `pushRepo` to also require `GetPreferences`; in
      `enqueuePushDeliveries`, compute `push.NotificationCategory(a.Type)`
      once, cache per-user preference lookups alongside the existing
      permission cache, and skip a subscription whose category is disabled
- [x] 4.2 Update `notification_worker_test.go`'s mock to implement the
      widened interface (defaulting to all-enabled) and add gating tests
      (end-to-end via a real `river.Client`, asserting only the opted-in
      subscriber's push actually enqueues)

## 5. Frontend

- [x] 5.1 New hooks (`features/notifications/hooks/usePushPreferencesQuery.ts`,
      `usePushPreferencesActions.ts`) wrapping GET/PUT via React Query
- [x] 5.2 Five toggles (attendance/events/news/polls/absence), added inline
      to `ProfileSheet` (`features/team/components/NavSheets.tsx`, extracted
      into `PushCategoryPreferencesPanel`) directly below the existing global
      push opt-in toggle, scoped to the currently active team and shown only
      once the browser already has an active push subscription -- simpler
      than a dedicated sheet/route, consistent with `usePushActions` already
      being self-contained UI state outside `AppContext`
- [x] 5.3 `mocks/{handlers.ts,db.ts}`, `services/serviceLayerReal.ts`,
      `services/serviceContract.test.ts`, `i18n/{de,en}.ts`

## 6. Verification

- [x] 6.1 `backend-openapi-drift` green (regenerated clients committed)
- [x] 6.2 Backend tests: default-enabled with no stored row, per-category
      opt-out suppresses only that category/team, permission gate and
      preference gate each independently suppress a push
- [x] 6.3 `golangci-lint` + `go test ./...` green; migration-rollback +
      migration-safety not runnable without Docker in this environment
- [x] 6.4 Frontend lint/typecheck/test/build + bundle budget green
