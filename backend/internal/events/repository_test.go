package events_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

const (
	testTeamID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	testUserID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func makeCreateParams(title string, date time.Time) events.CreateEventParams {
	return events.CreateEventParams{
		Type:  "training",
		Title: title,
		Date:  date,
	}
}

// findAttendanceRow returns the roster row for userID, or nil if absent --
// ListAttendance is roster-based (one row per current team member), so
// tests that need a specific member's row out of a multi-member roster use
// this instead of assuming row order or list length alone.
func findAttendanceRow(rows []events.AttendanceEnriched, userID uuid.UUID) *events.AttendanceEnriched {
	for i := range rows {
		if rows[i].UserId == userID {
			return &rows[i]
		}
	}
	return nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestEventRepository_CreateAndListEvents(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	// Seed required team and user rows.
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Test User', 'repo-test@example.com', '#ff0000')
	`, testUserID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO teams (id, name) VALUES ($1, 'Test Team')
	`, testTeamID)
	require.NoError(t, err)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	params1 := makeCreateParams("Training A", today)
	params2 := makeCreateParams("Training B", today.AddDate(0, 0, 7))

	e1, err := repo.CreateEvent(ctx, testTeamID, &params1)
	require.NoError(t, err)
	require.NotNil(t, e1)
	assert.Equal(t, "Training A", e1.Title)
	assert.Equal(t, "training", e1.Type)
	assert.Equal(t, "active", e1.Status)

	e2, err := repo.CreateEvent(ctx, testTeamID, &params2)
	require.NoError(t, err)
	require.NotNil(t, e2)
	assert.Equal(t, "Training B", e2.Title)

	// List all.
	all, err := repo.ListEvents(ctx, testTeamID, gen.All, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// List upcoming: one event is dated today, one a week out. BOTH must be
	// upcoming -- a today-dated event must not fall out of "upcoming" (regression
	// guard for the DATE-vs-timestamp boundary bug). Asserted exactly, not `>= 1`.
	upcoming, err := repo.ListEvents(ctx, testTeamID, gen.Upcoming, 50, nil)
	require.NoError(t, err)
	assert.Len(t, upcoming, 2, "today's event and next week's event must both be upcoming")

	// List past: today's event must NOT be past, and no earlier events exist.
	past, err := repo.ListEvents(ctx, testTeamID, gen.Past, 50, nil)
	require.NoError(t, err)
	assert.Len(t, past, 0, "a today-dated event must not be classified as past")
}

func TestEventRepository_CreateRecurringEvent(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Recurring User', 'recurring@example.com', '#00ff00')
	`, "cccccccc-cccc-cccc-cccc-cccccccccccc")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO teams (id, name) VALUES ($1, 'Recurring Team')
	`, "dddddddd-dddd-dddd-dddd-dddddddddddd")
	require.NoError(t, err)

	teamID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	startDate := time.Now().UTC().Truncate(24 * time.Hour)
	params := events.CreateEventParams{
		Type:        "training",
		Title:       "Weekly Training",
		Date:        startDate,
		Recurring:   true,
		RepeatWeeks: 4,
	}

	eventRows, err := repo.CreateSeries(ctx, teamID, &params)
	require.NoError(t, err)
	require.Len(t, eventRows, 4, "should have created 4 event instances")

	// Verify dates are spaced one week apart.
	for i, e := range eventRows {
		expectedDate := startDate.AddDate(0, 0, i*7)
		assert.Equal(t, expectedDate.Format("2006-01-02"), e.Date.Format("2006-01-02"),
			"event %d should be on week %d", i, i)
		assert.NotNil(t, e.SeriesId, "event %d should have a series_id", i)
	}

	// All events share the same series_id.
	assert.Equal(t, *eventRows[0].SeriesId, *eventRows[1].SeriesId)
	assert.Equal(t, *eventRows[0].SeriesId, *eventRows[3].SeriesId)
}

// Regression test: CreateEvent/CreateSeries/UpdateEvent must validate
// nominated_role_ids against the roles table inside their own transaction
// (holding the same pg_advisory_xact_lock key roles.DeleteRole uses), not
// just via the service layer's separate, lock-free pre-check -- otherwise a
// role deleted concurrently between that pre-check and this write could
// leave a dangling ID in nominated_role_ids that DeleteRole's scrub (which
// only runs once, at deletion time) would never clean up.
func TestEventRepository_CreateEvent_RejectsForeignTeamNominatedRole(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New().String()
	otherTeamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Nom Role Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Nom Role Team')`, otherTeamID)
	require.NoError(t, err)

	var foreignRoleID uuid.UUID
	err = pool.QueryRow(
		ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Foreign Role', '{}') RETURNING id`,
		otherTeamID,
	).Scan(&foreignRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Bad Nomination", time.Now().UTC())
	params.NominatedRoleIds = []uuid.UUID{foreignRoleID}
	_, err = repo.CreateEvent(ctx, teamID, &params)
	require.ErrorIs(t, err, events.ErrInvalidNominatedRoleIDs)

	// A role that genuinely belongs to the team is accepted.
	var ownRoleID uuid.UUID
	err = pool.QueryRow(
		ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Own Role', '{}') RETURNING id`,
		teamID,
	).Scan(&ownRoleID)
	require.NoError(t, err)
	params.NominatedRoleIds = []uuid.UUID{ownRoleID}
	ev, err := repo.CreateEvent(ctx, teamID, &params)
	require.NoError(t, err)
	require.NotNil(t, ev)
}

// Regression test: CreateEvent must not take the team's
// pg_advisory_xact_lock(hashtextextended(teamID, 0)) at all when the event has
// no nominated roles to validate -- otherwise it needlessly serializes against
// every other privilege-relevant mutation on the team (role deletion, role
// assignment, team updates), even though there's nothing here that could race
// with them.
func TestEventRepository_CreateEvent_NoNominatedRoles_DoesNotBlockOnAdvisoryLock(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Lock Team')`, teamID)
	require.NoError(t, err)

	lockHeld := make(chan struct{})
	lockReleased := make(chan struct{})
	go func() {
		defer close(lockReleased)
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID)
		require.NoError(t, err)
		close(lockHeld)
		time.Sleep(2 * time.Second)
		_ = tx.Rollback(ctx)
	}()

	<-lockHeld

	params := makeCreateParams("No Roles Event", time.Now().UTC())
	// params.NominatedRoleIds left nil -- nothing to validate.

	start := time.Now()
	ev, err := repo.CreateEvent(ctx, teamID, &params)
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.NotNil(t, ev)
	assert.Less(t, elapsed, 500*time.Millisecond,
		"CreateEvent with no nominated roles should not block on the team's advisory lock; took %v", elapsed)

	<-lockReleased
}

// A series-scope update with a new title AND a new date must apply the title
// to every occurrence but NOT collapse every occurrence onto the same date —
// date is what makes each occurrence distinct; only the specific event the
// update was invoked on should get the new date.
func TestEventRepository_UpdateEvent_Series_DoesNotCollapseDates(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	userID := "12121212-1212-1212-1212-121212121212"
	teamID := "34343434-3434-3434-3434-343434343434"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Series Update User', 'series-update@example.com', '#654321')
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Series Update Team')`, teamID)
	require.NoError(t, err)

	startDate := time.Now().UTC().Truncate(24 * time.Hour)
	params := events.CreateEventParams{
		Type:        "training",
		Title:       "Weekly Training",
		Date:        startDate,
		Recurring:   true,
		RepeatWeeks: 3,
	}
	eventRows, err := repo.CreateSeries(ctx, teamID, &params)
	require.NoError(t, err)
	require.Len(t, eventRows, 3)

	newTitle := "Renamed Training"
	newDate := startDate.AddDate(0, 0, 1) // shift only the targeted event by one day
	updated, err := repo.UpdateEvent(ctx, eventRows[0].Id.String(), teamID, &events.UpdateEventParams{
		Title: &newTitle,
		Date:  &newDate,
	}, "series")
	require.NoError(t, err)
	assert.Equal(t, newTitle, updated.Title)
	assert.Equal(t, newDate.Format("2006-01-02"), updated.Date.Format("2006-01-02"))

	all, err := repo.ListEvents(ctx, teamID, gen.All, 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 3)

	dates := make(map[string]bool)
	for _, e := range all {
		assert.Equal(t, newTitle, e.Title, "title must apply to every occurrence in the series")
		dates[e.Date.Format("2006-01-02")] = true
	}
	assert.Len(t, dates, 3, "each occurrence must keep its own distinct date, not collapse onto the updated event's date")
}

// Cancelling the remainder of a series must not retroactively cancel already-
// held (past) occurrences. Only today's and future instances flip to cancelled;
// a past occurrence keeps its status so team history and stats are unchanged.
func TestEventRepository_SetStatus_Series_DoesNotCancelPastInstances(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	userID := "56565656-5656-5656-5656-565656565656"
	teamID := "78787878-7878-7878-7878-787878787878"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Series Cancel User', 'series-cancel@example.com', '#abcdef')
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Series Cancel Team')`, teamID)
	require.NoError(t, err)

	startDate := time.Now().UTC().Truncate(24 * time.Hour)
	params := events.CreateEventParams{
		Type:        "training",
		Title:       "Weekly Training",
		Date:        startDate,
		Recurring:   true,
		RepeatWeeks: 3,
	}
	rows, err := repo.CreateSeries(ctx, teamID, &params)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	// Backdate the first occurrence into the past (yesterday), keeping its series_id.
	pastID := rows[0].Id
	_, err = pool.Exec(ctx, `UPDATE events SET date = $1 WHERE id = $2`, startDate.AddDate(0, 0, -1), pastID)
	require.NoError(t, err)

	// Cancel the remainder of the series, invoked on a future occurrence.
	_, err = repo.SetStatus(ctx, rows[2].Id.String(), teamID, "cancelled", "series")
	require.NoError(t, err)

	all, err := repo.ListEvents(ctx, teamID, gen.All, 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 3)
	for _, e := range all {
		if e.Id == pastID {
			assert.Equal(t, "active", e.Status, "a past occurrence must keep its status when the series is cancelled")
		} else {
			assert.Equal(t, "cancelled", e.Status, "today/future occurrences must be cancelled")
		}
	}
}

// Regression test: the handler's endTime>startTime check only runs when both
// fields are present in the same PATCH request (see events.validateEventFields);
// a partial update sending only one of the two fields skipped that check
// entirely and could persist end_time <= start_time. The
// events_end_after_start_time CHECK constraint (mirroring absences' identical
// from_date/to_date pattern) must catch this at the DB layer regardless, and
// UpdateEvent must map the violation to the same ErrEndTimeBeforeStartTime
// the handler-level check returns.
func TestEventRepository_UpdateEvent_PartialUpdate_RejectsEndBeforeStart(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	userID := "23232323-2323-2323-2323-232323232323"
	teamID := "45454545-4545-4545-4545-454545454545"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Partial Update User', 'partial-update@example.com', '#123123')
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Partial Update Team')`, teamID)
	require.NoError(t, err)

	startTime, endTime := "09:00", "10:00"
	params := makeCreateParams("Training", time.Now().UTC().Truncate(24*time.Hour))
	params.StartTime = &startTime
	params.EndTime = &endTime
	created, err := repo.CreateEvent(ctx, teamID, &params)
	require.NoError(t, err)

	// Partial update: only endTime, set before the stored startTime (09:00).
	newEndTime := "08:00"
	_, err = repo.UpdateEvent(ctx, created.Id.String(), teamID, &events.UpdateEventParams{
		EndTime: &newEndTime,
	}, "single")
	require.ErrorIs(t, err, events.ErrEndTimeBeforeStartTime)

	// Partial update: only startTime, set after the stored endTime (10:00).
	newStartTime := "11:00"
	_, err = repo.UpdateEvent(ctx, created.Id.String(), teamID, &events.UpdateEventParams{
		StartTime: &newStartTime,
	}, "single")
	require.ErrorIs(t, err, events.ErrEndTimeBeforeStartTime)
}

// A series-scope update with ONLY Date set (no other field) — the primary
// use case the fix above exists to support — must not corrupt the
// series-wide UPDATE: with every other field nil, buildUpdateSets' "nothing
// to update" no-op fallback ("SET id = $1") combined with the stripped
// eventID arg previously produced "UPDATE events SET id = $1 WHERE
// series_id = $1", overwriting every event's primary key with the series
// ID. Only the single targeted event's date should change; every other
// occurrence and the events table's row identities must be untouched.
func TestEventRepository_UpdateEvent_Series_OnlyDateSet_DoesNotCorruptSQL(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	userID := "56565656-5656-5656-5656-565656565656"
	teamID := "78787878-7878-7878-7878-787878787878"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Only Date User', 'only-date@example.com', '#abcdef')
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Only Date Team')`, teamID)
	require.NoError(t, err)

	startDate := time.Now().UTC().Truncate(24 * time.Hour)
	params := events.CreateEventParams{
		Type:        "training",
		Title:       "Weekly Training",
		Date:        startDate,
		Recurring:   true,
		RepeatWeeks: 3,
	}
	eventRows, err := repo.CreateSeries(ctx, teamID, &params)
	require.NoError(t, err)
	require.Len(t, eventRows, 3)
	originalTitle := eventRows[0].Title
	targetID := eventRows[0].Id
	otherIDs := []uuid.UUID{eventRows[1].Id, eventRows[2].Id}

	newDate := startDate.AddDate(0, 0, 1)
	updated, err := repo.UpdateEvent(ctx, targetID.String(), teamID, &events.UpdateEventParams{
		Date: &newDate,
	}, "series")
	require.NoError(t, err, "must not error — the degenerate all-nil-except-Date series update must be a safe no-op for the series-wide UPDATE")
	assert.Equal(t, newDate.Format("2006-01-02"), updated.Date.Format("2006-01-02"))
	assert.Equal(t, targetID, updated.Id, "the targeted event's own identity must be unchanged")
	assert.Equal(t, originalTitle, updated.Title, "title must be untouched since it wasn't in the patch")

	all, err := repo.ListEvents(ctx, teamID, gen.All, 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 3, "no rows must be lost or merged")

	seenIDs := make(map[uuid.UUID]bool)
	for _, e := range all {
		seenIDs[e.Id] = true
		assert.Equal(t, originalTitle, e.Title)
	}
	assert.True(t, seenIDs[targetID])
	for _, id := range otherIDs {
		assert.True(t, seenIDs[id], "every event's primary key must survive unchanged")
	}
}

// Deleting a series must remove every occurrence (past and future), not just
// the event_series definition row — events.series_id is ON DELETE SET NULL,
// so leaving DeleteEvent to rely on FK cascade alone would silently detach
// the events instead of removing them, contradicting the UI's "all events in
// this series ... will be permanently removed" confirmation copy.
func TestEventRepository_DeleteEvent_Series_RemovesAllOccurrences(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	userID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	teamID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Series Delete User', 'series-delete@example.com', '#123456')
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Series Delete Team')`, teamID)
	require.NoError(t, err)

	startDate := time.Now().UTC().Truncate(24 * time.Hour)
	params := events.CreateEventParams{
		Type:        "training",
		Title:       "Weekly Training",
		Date:        startDate,
		Recurring:   true,
		RepeatWeeks: 3,
	}
	eventRows, err := repo.CreateSeries(ctx, teamID, &params)
	require.NoError(t, err)
	require.Len(t, eventRows, 3)

	err = repo.DeleteEvent(ctx, eventRows[0].Id.String(), teamID, "series")
	require.NoError(t, err)

	all, err := repo.ListEvents(ctx, teamID, gen.All, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, all, "all occurrences of the deleted series must be gone, not just detached")

	var seriesCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM event_series WHERE id = $1`, *eventRows[0].SeriesId).Scan(&seriesCount))
	assert.Equal(t, 0, seriesCount)
}

func TestEventRepository_SetAttendance(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color)
		VALUES ($1, 'Attend User', 'attend@example.com', '#0000ff')
	`, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO teams (id, name) VALUES ($1, 'Attend Team')
	`, "ffffffff-ffff-ffff-ffff-ffffffffffff")
	require.NoError(t, err)

	teamID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	userID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"

	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, userID)
	require.NoError(t, err)

	params := makeCreateParams("Match Day", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID, &params)
	require.NoError(t, err)

	eventID := ev.Id.String()

	// First upsert: yes.
	status := "yes"
	rec, err := repo.SetAttendance(ctx, eventID, userID, userID, teamID, &status, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "yes", rec.Status)
	firstID := rec.Id

	// Second upsert: change to no — should update, not insert.
	status = "no"
	reason := "sick"
	rec2, err := repo.SetAttendance(ctx, eventID, userID, userID, teamID, &status, &reason, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, rec2)
	assert.Equal(t, "no", rec2.Status)
	assert.Equal(t, &reason, rec2.Reason)
	// Verify same logical row (UNIQUE constraint upheld).
	assert.Equal(t, firstID, rec2.Id, "upsert should update existing row, not insert new one")

	// Verify GetMyAttendance returns the latest record.
	myRec, err := repo.GetMyAttendance(ctx, eventID, userID, teamID)
	require.NoError(t, err)
	require.NotNil(t, myRec)
	assert.Equal(t, "no", myRec.Status)
}

func TestEventRepository_SetAttendance_RejectsNonMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	nonMemberID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'No Membership Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Non Member', 'nonmember@example.com', '#123123')
	`, nonMemberID)
	require.NoError(t, err)

	params := makeCreateParams("Non-Member Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)

	status := "yes"
	_, err = repo.SetAttendance(ctx, ev.Id.String(), nonMemberID.String(), nonMemberID.String(), teamID.String(), &status, nil, nil, nil)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetAttendance must reject a userID that is not a member of teamID")
}

// Regression test: SetAttendance's cross-member write path used to trust the
// service layer's earlier, unlocked permChecker.GetPermissions read as the
// SOLE enforcement of events:write -- a concurrent SetRoles/DeleteRole/
// UpdateRole revoking the caller's events:write between that check and this
// write could still let the write through. The write itself must now
// re-verify callerID currently holds events:write via its own WHERE clause,
// so even calling this repository method directly (bypassing the service
// layer's permission check entirely, as this test does) rejects a caller
// with no events:write role.
func TestEventRepository_SetAttendance_RejectsCallerWithoutEventsWrite(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'No Write Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
			($1, 'Caller', 'caller-nowrite@example.com', '#111111'),
			($2, 'Target', 'target-nowrite@example.com', '#222222')
	`, callerID, targetID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)`, teamID, callerID, targetID)
	require.NoError(t, err)

	// Caller has a role, but it grants events:read, not events:write.
	var readOnlyRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Read Only', '{"events":"read"}') RETURNING id`,
		teamID,
	).Scan(&readOnlyRoleID)
	require.NoError(t, err)
	var callerMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, callerID).Scan(&callerMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, callerMembershipID, readOnlyRoleID)
	require.NoError(t, err)

	params := makeCreateParams("No Write Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)

	status := "yes"
	_, err = repo.SetAttendance(ctx, ev.Id.String(), callerID.String(), targetID.String(), teamID.String(), &status, nil, nil, nil)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetAttendance must reject a cross-member write from a caller without events:write, even called directly")

	// The rejected write must not have created a row at all.
	rec, err := repo.GetMyAttendance(ctx, ev.Id.String(), targetID.String(), teamID.String())
	require.NoError(t, err)
	assert.Nil(t, rec, "no attendance row should exist after the rejected cross-member write")
}

// Companion positive test: a caller WITH events:write can set another
// member's attendance, confirming the new WHERE-clause predicate isn't
// overly restrictive.
func TestEventRepository_SetAttendance_AllowsCallerWithEventsWrite(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Write Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
			($1, 'Organizer', 'caller-write@example.com', '#111111'),
			($2, 'Target', 'target-write@example.com', '#222222')
	`, callerID, targetID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)`, teamID, callerID, targetID)
	require.NoError(t, err)

	var writeRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Organizer', '{"events":"write"}') RETURNING id`,
		teamID,
	).Scan(&writeRoleID)
	require.NoError(t, err)
	var callerMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, callerID).Scan(&callerMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, callerMembershipID, writeRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Write Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)

	status := "yes"
	rec, err := repo.SetAttendance(ctx, ev.Id.String(), callerID.String(), targetID.String(), teamID.String(), &status, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "yes", rec.Status)
	assert.Equal(t, targetID.String(), rec.UserId.String())
}

func TestEventRepository_GetReasonVisibilityContext(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	viewerID := uuid.New()
	trainerRoleID := uuid.New()
	otherRoleID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name, reason_visibility_role_ids) VALUES ($1, 'Reason Vis Team', $2)`,
		teamID, []uuid.UUID{trainerRoleID})
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Viewer', 'viewer-reasonvis@example.com', '#123123')`,
		viewerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO roles (id, team_id, name) VALUES ($1, $2, 'Trainer'), ($3, $2, 'Other')
	`, trainerRoleID, teamID, otherRoleID)
	require.NoError(t, err)

	var membershipID uuid.UUID
	err = pool.QueryRow(
		ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, teamID, viewerID,
	).Scan(&membershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, membershipID, otherRoleID)
	require.NoError(t, err)

	teamRoleIDs, viewerRoleIDs, err := repo.GetReasonVisibilityContext(ctx, teamID.String(), viewerID.String())
	require.NoError(t, err)
	assert.Equal(t, []string{trainerRoleID.String()}, teamRoleIDs)
	assert.Equal(t, []string{otherRoleID.String()}, viewerRoleIDs)
}

// TestEventRepository_GetReasonVisibilityContext_ExcludesCrossTeamRole is a
// defense-in-depth regression test: the viewer-roles query joined roles to
// membership_roles with no r.team_id check, unlike the established pattern
// elsewhere (members.getMembershipEffectivePermissionsQ,
// teams.Repository.GetRolesForMembership). Every current INSERT INTO
// membership_roles call site always inserts a role already validated as
// belonging to the target team, so this can't happen through normal API use
// today -- but this feeds the decline-reason redaction decision in
// ListAttendance, so a future change that broke that insert-side invariant
// would silently let a cross-team role slip a viewer into the reason-
// visibility whitelist instead of failing safe.
func TestEventRepository_GetReasonVisibilityContext_ExcludesCrossTeamRole(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	otherTeamID := uuid.New()
	viewerID := uuid.New()
	ownRoleID := uuid.New()
	foreignRoleID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Reason Vis Own Team'), ($2, 'Reason Vis Other Team')`,
		teamID, otherTeamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Cross Team Viewer', 'viewer-crossteam-reasonvis@example.com', '#456456')`,
		viewerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO roles (id, team_id, name) VALUES ($1, $2, 'Own'), ($3, $4, 'Foreign')`,
		ownRoleID, teamID, foreignRoleID, otherTeamID)
	require.NoError(t, err)

	var membershipID uuid.UUID
	err = pool.QueryRow(
		ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, teamID, viewerID,
	).Scan(&membershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2), ($1, $3)`,
		membershipID, ownRoleID, foreignRoleID)
	require.NoError(t, err)

	_, viewerRoleIDs, err := repo.GetReasonVisibilityContext(ctx, teamID.String(), viewerID.String())
	require.NoError(t, err)
	assert.Contains(t, viewerRoleIDs, ownRoleID.String(), "the viewer's own-team role must still be returned")
	assert.NotContains(t, viewerRoleIDs, foreignRoleID.String(), "a role belonging to a different team must never be returned, even if membership_roles points at it")
}

func TestEventRepository_BatchedAttendanceLookups(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	otherUserID := uuid.New()
	teamID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
		($1, 'Batch User', 'batch-user@example.com', '#123456'),
		($2, 'Other User', 'batch-other@example.com', '#654321')
	`, userID, otherUserID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Batch Team')`, teamID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)
	`, teamID, userID, otherUserID)
	require.NoError(t, err)

	params1 := makeCreateParams("Batch Event 1", time.Now().UTC())
	e1, err := repo.CreateEvent(ctx, teamID.String(), &params1)
	require.NoError(t, err)
	params2 := makeCreateParams("Batch Event 2", time.Now().UTC())
	e2, err := repo.CreateEvent(ctx, teamID.String(), &params2)
	require.NoError(t, err)
	params3 := makeCreateParams("Batch Event 3 (no attendance)", time.Now().UTC())
	e3, err := repo.CreateEvent(ctx, teamID.String(), &params3)
	require.NoError(t, err)

	yes := "yes"
	no := "no"
	_, err = repo.SetAttendance(ctx, e1.Id.String(), userID.String(), userID.String(), teamID.String(), &yes, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, e1.Id.String(), otherUserID.String(), otherUserID.String(), teamID.String(), &no, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, e2.Id.String(), userID.String(), userID.String(), teamID.String(), &no, nil, nil, nil)
	require.NoError(t, err)

	eventIDs := []uuid.UUID{e1.Id, e2.Id, e3.Id}

	summaries, err := repo.GetAttendanceSummaries(ctx, eventIDs)
	require.NoError(t, err)
	assert.Equal(t, 1, summaries[e1.Id].Yes)
	assert.Equal(t, 1, summaries[e1.Id].No)
	assert.Equal(t, 2, summaries[e1.Id].Total)
	// e2: otherUserID never explicitly responded -- roster-based summaries
	// count them as a "pending" non-responder rather than dropping them
	// from Total entirely (the exact bug this roster-based rewrite fixes).
	assert.Equal(t, 1, summaries[e2.Id].No)
	assert.Equal(t, 1, summaries[e2.Id].Pending)
	assert.Equal(t, 2, summaries[e2.Id].Total)
	// e3: nobody has responded at all -- still present in the map with the
	// full 2-member roster as "pending", not absent (the batch-query
	// equivalent of the same fix).
	require.Contains(t, summaries, e3.Id, "event with no attendance rows must still report its full roster")
	assert.Equal(t, 0, summaries[e3.Id].Yes)
	assert.Equal(t, 2, summaries[e3.Id].Pending)
	assert.Equal(t, 2, summaries[e3.Id].Total)

	// Cross-check against the single-event methods to prove the batched
	// queries return identical results, not just plausible-looking ones.
	singleSummary, err := repo.GetAttendanceSummary(ctx, e1.Id.String(), teamID.String())
	require.NoError(t, err)
	assert.Equal(t, singleSummary, summaries[e1.Id])

	myAttendances, err := repo.GetMyAttendances(ctx, eventIDs, userID.String())
	require.NoError(t, err)
	require.Contains(t, myAttendances, e1.Id)
	assert.Equal(t, "yes", myAttendances[e1.Id].Status)
	require.Contains(t, myAttendances, e2.Id)
	assert.Equal(t, "no", myAttendances[e2.Id].Status)
	_, e3HasMyAttendance := myAttendances[e3.Id]
	assert.False(t, e3HasMyAttendance, "event with no attendance from this user should be absent from the map")

	// Empty ID list should short-circuit without a query and return an empty map.
	emptySummaries, err := repo.GetAttendanceSummaries(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, emptySummaries)
}

// Regression test: members.Repository.RemoveMember only deletes the
// memberships row -- attendance is keyed by user_id/team_id, not
// membership_id, so it's deliberately left in place as history. Without a
// membership join, GetAttendanceSummary/GetAttendanceSummaries/ListAttendance
// kept counting and naming a departed member forever, silently inflating an
// event's headcount with no way for an admin to detect the discrepancy from
// the API response alone.
func TestEventRepository_AttendanceExcludesDepartedMembers(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	stayingUserID := uuid.New()
	departedUserID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
		($1, 'Staying Member', 'staying@example.com', '#111111'),
		($2, 'Departed Member', 'departed@example.com', '#222222')
	`, stayingUserID, departedUserID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Departure Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)
	`, teamID, stayingUserID, departedUserID)
	require.NoError(t, err)

	params := makeCreateParams("Departure Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)

	yes := "yes"
	_, err = repo.SetAttendance(ctx, ev.Id.String(), stayingUserID.String(), stayingUserID.String(), teamID.String(), &yes, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, ev.Id.String(), departedUserID.String(), departedUserID.String(), teamID.String(), &yes, nil, nil, nil)
	require.NoError(t, err)

	// Sanity check: both count before the departure.
	before, err := repo.GetAttendanceSummary(ctx, ev.Id.String(), teamID.String())
	require.NoError(t, err)
	assert.Equal(t, 2, before.Total)
	assert.Equal(t, 2, before.Yes)

	attendeesBefore, err := repo.ListAttendance(ctx, ev.Id.String(), teamID.String())
	require.NoError(t, err)
	assert.Len(t, attendeesBefore, 2)

	// Departure: only the memberships row is removed, mirroring
	// members.Repository.RemoveMember -- the attendance row for
	// departedUserID is deliberately left in place as history.
	_, err = pool.Exec(ctx, `DELETE FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, departedUserID)
	require.NoError(t, err)

	after, err := repo.GetAttendanceSummary(ctx, ev.Id.String(), teamID.String())
	require.NoError(t, err)
	assert.Equal(t, 1, after.Total, "departed member must not count toward the event's total")
	assert.Equal(t, 1, after.Yes)

	attendeesAfter, err := repo.ListAttendance(ctx, ev.Id.String(), teamID.String())
	require.NoError(t, err)
	require.Len(t, attendeesAfter, 1)
	assert.Equal(t, stayingUserID, attendeesAfter[0].UserId)

	summaries, err := repo.GetAttendanceSummaries(ctx, []uuid.UUID{ev.Id})
	require.NoError(t, err)
	assert.Equal(t, 1, summaries[ev.Id].Total, "batched summary must also exclude the departed member")
}

// Regression test for the round-79 finding: response_mode="opt_out" and
// absence-based auto-attendance were validated and stored but never
// actually applied anywhere -- ListAttendance/GetAttendanceSummary only
// counted rows that physically existed in the attendance table, so a
// non-responder to an opt_out event silently showed as "pending" forever
// instead of the intended "auto yes", and a member with a planned absence
// covering the event date was never auto-marked "no". This test drives the
// full precedence chain: explicit response wins; otherwise a covering
// absence defaults to auto "no"; otherwise opt_out defaults to auto "yes";
// otherwise "pending".
func TestEventRepository_ListAttendance_OptOutAndAbsenceDefaulting(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	respondedUserID := uuid.New()  // explicitly declines
	absentUserID := uuid.New()     // never responds, has a covering absence
	silentUserID := uuid.New()     // never responds, no absence
	explicitAbsentID := uuid.New() // explicitly accepts despite a covering absence

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
		($1, 'Responded User', 'opt-out-responded@example.com', '#111111'),
		($2, 'Absent User', 'opt-out-absent@example.com', '#222222'),
		($3, 'Silent User', 'opt-out-silent@example.com', '#333333'),
		($4, 'Explicit Despite Absence', 'opt-out-explicit-absent@example.com', '#444444')
	`, respondedUserID, absentUserID, silentUserID, explicitAbsentID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Opt Out Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3), ($1, $4), ($1, $5)
	`, teamID, respondedUserID, absentUserID, silentUserID, explicitAbsentID)
	require.NoError(t, err)

	eventDate := time.Now().UTC().Truncate(24 * time.Hour)
	params := makeCreateParams("Opt-out Training", eventDate)
	optOut := "opt_out"
	params.ResponseMode = &optOut
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	// absentUserID and explicitAbsentID both have a planned absence covering
	// the event date.
	_, err = pool.Exec(ctx, `
		INSERT INTO absences (user_id, team_id, from_date, to_date)
		VALUES ($1, $2, $3, $3), ($4, $2, $3, $3)
	`, absentUserID, teamID, eventDate, explicitAbsentID)
	require.NoError(t, err)

	no := "no"
	_, err = repo.SetAttendance(ctx, eventID, respondedUserID.String(), respondedUserID.String(), teamID.String(), &no, nil, nil, nil)
	require.NoError(t, err)
	yes := "yes"
	_, err = repo.SetAttendance(ctx, eventID, explicitAbsentID.String(), explicitAbsentID.String(), teamID.String(), &yes, nil, nil, nil)
	require.NoError(t, err)

	rows, err := repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, rows, 4)

	responded := findAttendanceRow(rows, respondedUserID)
	require.NotNil(t, responded)
	assert.Equal(t, "no", responded.Status, "explicit response must win over opt_out defaulting")
	assert.False(t, responded.Auto)
	assert.False(t, responded.Absent, "no covering absence for this member")

	absent := findAttendanceRow(rows, absentUserID)
	require.NotNil(t, absent)
	assert.Equal(t, "no", absent.Status, "a covering absence must default a non-responder to auto no")
	assert.True(t, absent.Auto)
	assert.True(t, absent.Absent)

	silent := findAttendanceRow(rows, silentUserID)
	require.NotNil(t, silent)
	assert.Equal(t, "yes", silent.Status, "opt_out must default a non-responder with no absence to auto yes")
	assert.True(t, silent.Auto)
	assert.False(t, silent.Absent)

	explicitDespiteAbsence := findAttendanceRow(rows, explicitAbsentID)
	require.NotNil(t, explicitDespiteAbsence)
	assert.Equal(t, "yes", explicitDespiteAbsence.Status, "explicit response must win over the absence default too")
	assert.False(t, explicitDespiteAbsence.Auto)
	assert.True(t, explicitDespiteAbsence.Absent, "Absent still reflects the live absence overlap even though the member responded")

	summary, err := repo.GetAttendanceSummary(ctx, eventID, teamID.String())
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Yes, "silentUser (auto) + explicitAbsentUser (explicit)")
	assert.Equal(t, 2, summary.No, "respondedUser (explicit) + absentUser (auto)")
	assert.Equal(t, 0, summary.Pending)
	assert.Equal(t, 4, summary.Total)

	summaries, err := repo.GetAttendanceSummaries(ctx, []uuid.UUID{ev.Id})
	require.NoError(t, err)
	assert.Equal(t, summary, summaries[ev.Id], "batched summary must agree with the single-event summary")
}

func TestEventRepository_SetStatus_CrossTeamBlocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	otherTeamID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Status Team A')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Status Team B')`, otherTeamID)
	require.NoError(t, err)

	params := makeCreateParams("Status Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	assert.Equal(t, "active", ev.Status)

	// A member of another team must not be able to change this event's status.
	_, err = repo.SetStatus(ctx, ev.Id.String(), otherTeamID.String(), "cancelled", "single")
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	// The event must remain unaffected.
	unchanged, err := repo.GetEvent(ctx, ev.Id.String(), teamID.String())
	require.NoError(t, err)
	assert.Equal(t, "active", unchanged.Status)

	// Scoped to the correct team, the update succeeds.
	updated, err := repo.SetStatus(ctx, ev.Id.String(), teamID.String(), "cancelled", "single")
	require.NoError(t, err)
	assert.Equal(t, "cancelled", updated.Status)
}

// TestEventRepository_CrossTenantIDOR verifies that every event-scoped
// read/write repository method rejects an eventID that belongs to a
// different team than the one supplied — regression test for the
// cross-tenant IDOR where these methods used to filter only by eventID.
func TestEventRepository_CrossTenantIDOR(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamA := uuid.New()
	teamB := uuid.New()
	user := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'IDOR Team A'), ($2, 'IDOR Team B')`, teamA, teamB)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'IDOR User', 'idor@example.com', '#abcdef')`, user)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamA, user)
	require.NoError(t, err)

	// user needs events:write in teamA for the SetNomination calls below,
	// which are never self-service and now re-verify events:write atomically.
	var writeRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'IDOR Organizer', '{"events":"write"}') RETURNING id`,
		teamA,
	).Scan(&writeRoleID)
	require.NoError(t, err)
	var userMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamA, user).Scan(&userMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, userMembershipID, writeRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Team A Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamA.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	// A member of Team B must not be able to read Team A's event via any
	// event-scoped method, even though the eventID itself is valid.
	_, err = repo.GetEvent(ctx, eventID, teamB.String())
	assert.ErrorIs(t, err, pgx.ErrNoRows, "GetEvent must reject cross-team eventID")

	// Aggregate query, never errors — assert it reports zero for a cross-team event instead.
	summary, err := repo.GetAttendanceSummary(ctx, eventID, teamB.String())
	require.NoError(t, err)
	assert.Equal(t, 0, summary.Total, "GetAttendanceSummary must not see cross-team event's attendance")

	myAtt, err := repo.GetMyAttendance(ctx, eventID, user.String(), teamB.String())
	require.NoError(t, err)
	assert.Nil(t, myAtt, "GetMyAttendance must not see cross-team event's attendance")

	attendanceList, err := repo.ListAttendance(ctx, eventID, teamB.String())
	require.NoError(t, err)
	assert.Empty(t, attendanceList, "ListAttendance must not see cross-team event's attendance")

	comments, err := repo.ListComments(ctx, eventID, teamB.String(), 50, nil)
	require.NoError(t, err)
	assert.Empty(t, comments, "ListComments must not see cross-team event's comments")

	// A member of Team B must not be able to write to Team A's event either.
	status := "yes"
	_, err = repo.SetAttendance(ctx, eventID, user.String(), user.String(), teamB.String(), &status, nil, nil, nil)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetAttendance must reject cross-team eventID")

	err = repo.SetNomination(ctx, eventID, user.String(), user.String(), teamB.String(), false)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(false) must reject cross-team eventID")

	err = repo.SetNomination(ctx, eventID, user.String(), user.String(), teamB.String(), true)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(true) must reject cross-team eventID")

	_, err = repo.AddComment(ctx, eventID, user.String(), teamB.String(), "cross-team comment")
	assert.ErrorIs(t, err, pgx.ErrNoRows, "AddComment must reject cross-team eventID")

	// Verify no attendance/comment rows leaked through despite the rejected
	// calls. ListAttendance is roster-based (every current member of teamA
	// appears, whether or not they've explicitly responded), so a plain
	// emptiness check no longer distinguishes "no row was created" -- use
	// GetMyAttendance (still explicit-row-only) directly instead.
	noRow, err := repo.GetMyAttendance(ctx, eventID, user.String(), teamA.String())
	require.NoError(t, err)
	assert.Nil(t, noRow, "no attendance row should have been created by the rejected cross-team SetAttendance call")

	comments, err = repo.ListComments(ctx, eventID, teamA.String(), 50, nil)
	require.NoError(t, err)
	assert.Empty(t, comments, "no comment should have been created by the rejected cross-team AddComment call")

	// Scoped to the correct team, all the same operations succeed.
	rec, err := repo.SetAttendance(ctx, eventID, user.String(), user.String(), teamA.String(), &status, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "yes", rec.Status)

	comment, err := repo.AddComment(ctx, eventID, user.String(), teamA.String(), "same-team comment")
	require.NoError(t, err)
	assert.Equal(t, "same-team comment", comment.Text)

	// DeleteComment stays scoped by (commentID, userID) for self-ownership,
	// but must also reject a mismatched teamID.
	err = repo.DeleteComment(ctx, comment.Id.String(), user.String(), teamB.String())
	assert.ErrorIs(t, err, pgx.ErrNoRows, "DeleteComment must reject cross-team teamID")

	err = repo.DeleteComment(ctx, comment.Id.String(), user.String(), teamA.String())
	require.NoError(t, err, "DeleteComment scoped to the correct team must succeed")
}

// TestEventRepository_SetNomination_RejectsNonMemberUser regression-tests a bug
// where SetNomination(false) checked that eventID belonged to teamID but never
// checked that userID was actually a member of that team (unlike the
// equivalent SetAttendance query) — a caller with events:write could nominate
// an arbitrary, unrelated user UUID onto the event, leaking that user's
// identity via ListAttendance to every member of a team they have no
// relationship to.
func TestEventRepository_SetNomination_RejectsNonMemberUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	member := uuid.New()
	outsider := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Nomination Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES
		($1, 'Member', 'member@example.com', '#abcdef'),
		($2, 'Outsider', 'outsider@example.com', '#123456')`, member, outsider)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, member)
	require.NoError(t, err)

	// member needs events:write to act as the caller below (SetNomination is
	// never self-service and now re-verifies events:write atomically).
	var writeRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Nomination Organizer', '{"events":"write"}') RETURNING id`,
		teamID,
	).Scan(&writeRoleID)
	require.NoError(t, err)
	var memberMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, member).Scan(&memberMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, memberMembershipID, writeRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Nomination Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	err = repo.SetNomination(ctx, eventID, member.String(), outsider.String(), teamID.String(), false)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(false) must reject a userID that is not a member of teamID")

	// ListAttendance is roster-based (every current team member appears,
	// whether or not they've responded), so the real member shows up
	// regardless -- the actual regression to guard is that the non-member
	// outsider never leaks into the list, which structurally can't happen
	// via a roster join (outsider was never a memberships row to begin with).
	attendanceList, err := repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, attendanceList, 1, "only the real member should appear")
	assert.Equal(t, member, attendanceList[0].UserId)
	assert.NotEqual(t, "not_nominated", attendanceList[0].Status, "no attendance row should have been created for the non-member user")

	// A real member of the team is unaffected.
	err = repo.SetNomination(ctx, eventID, member.String(), member.String(), teamID.String(), false)
	require.NoError(t, err, "SetNomination(false) scoped to an actual team member must succeed")
}

// TestEventRepository_AddComment_RejectsNonMemberUser regression-tests a bug
// where AddComment checked that eventID belonged to teamID but never checked
// that userID was actually a member of that team, unlike SetAttendance/
// SetNomination's equivalent self-service writes. events/comments is
// self-service (see authz.go), so RequireMembership only checks membership
// once at the start of the request -- a membership removal racing a
// concurrent AddComment call could otherwise attach a permanently visible
// comment to an event from someone no longer on the team.
func TestEventRepository_AddComment_RejectsNonMemberUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	member := uuid.New()
	outsider := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Comment Membership Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES
		($1, 'Member', 'comment-member@example.com', '#abcdef'),
		($2, 'Outsider', 'comment-outsider@example.com', '#123456')`, member, outsider)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, member)
	require.NoError(t, err)

	params := makeCreateParams("Comment Membership Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	_, err = repo.AddComment(ctx, eventID, outsider.String(), teamID.String(), "should be rejected")
	assert.ErrorIs(t, err, pgx.ErrNoRows, "AddComment must reject a userID that is not a member of teamID")

	comments, err := repo.ListComments(ctx, eventID, teamID.String(), 50, nil)
	require.NoError(t, err)
	assert.Empty(t, comments, "no comment should have been created for the non-member user")

	// A real member of the team is unaffected.
	comment, err := repo.AddComment(ctx, eventID, member.String(), teamID.String(), "from an actual member")
	require.NoError(t, err, "AddComment scoped to an actual team member must succeed")
	assert.Equal(t, "from an actual member", comment.Text)
}

// TestEventRepository_ListComments_KeysetPaginatesWholeThread verifies the
// keyset pagination reaches every comment, oldest-first, with no row skipped or
// repeated across pages -- and stays scoped to the event's team.
func TestEventRepository_ListComments_KeysetPaginatesWholeThread(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	member := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Keyset Comment Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Member', 'keyset-comment@example.com', '#abcdef')`, member)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, member)
	require.NoError(t, err)

	params := makeCreateParams("Keyset Comment Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	// Insert comments with explicit, ascending timestamps so the chronological
	// order is deterministic to assert against.
	const total = 5
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < total; i++ {
		_, err = pool.Exec(ctx,
			`INSERT INTO event_comments (id, event_id, user_id, text, created_at) VALUES ($1, $2, $3, $4, $5)`,
			uuid.New(), ev.Id, member, fmt.Sprintf("comment-%d", i), base.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	var seen []string
	var lastCreated time.Time
	var cur *events.CommentCursor
	pages := 0
	for {
		page, err := repo.ListComments(ctx, eventID, teamID.String(), 2, cur)
		require.NoError(t, err)
		if len(page) == 0 {
			break
		}
		pages++
		for _, c := range page {
			if !lastCreated.IsZero() {
				assert.False(t, c.CreatedAt.Before(lastCreated), "comments must be returned oldest-first across pages")
			}
			lastCreated = c.CreatedAt
			seen = append(seen, c.Id.String())
		}
		if len(page) < 2 {
			break
		}
		last := page[len(page)-1]
		cur = &events.CommentCursor{CreatedAt: last.CreatedAt, ID: last.Id}
		require.LessOrEqual(t, pages, total+1, "pagination must terminate")
	}

	assert.Len(t, seen, total, "every comment must be reachable by paging")
	unique := map[string]struct{}{}
	for _, id := range seen {
		_, dup := unique[id]
		assert.False(t, dup, "keyset pagination must not repeat a comment across pages")
		unique[id] = struct{}{}
	}

	// A different team sees none of this event's comments.
	other, err := repo.ListComments(ctx, eventID, uuid.New().String(), 50, nil)
	require.NoError(t, err)
	assert.Empty(t, other, "ListComments must stay scoped to the event's team")
}

func TestEventRepository_CountComments(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	member := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Count Comments Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Member', 'count-comments-member@example.com', '#abcdef')`, member)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, member)
	require.NoError(t, err)

	params := makeCreateParams("Count Comments Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	count, err := repo.CountComments(ctx, eventID, teamID.String())
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	_, err = repo.AddComment(ctx, eventID, member.String(), teamID.String(), "one")
	require.NoError(t, err)
	_, err = repo.AddComment(ctx, eventID, member.String(), teamID.String(), "two")
	require.NoError(t, err)

	count, err = repo.CountComments(ctx, eventID, teamID.String())
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// A cross-team teamID must not see this event's comments.
	otherTeamID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Count Comments Other Team')`, otherTeamID)
	require.NoError(t, err)
	count, err = repo.CountComments(ctx, eventID, otherTeamID.String())
	require.NoError(t, err)
	assert.Equal(t, 0, count, "CountComments must not see a cross-team event's comments")
}

// TestEventRepository_SetNomination_ClearsStaleReason regression-tests a bug
// where SetNomination(false)'s ON CONFLICT branch only updated status/at,
// leaving a prior "no" row's reason/reason_id/reason_visibility untouched.
// ListAttendance's confidentiality redaction is gated strictly on
// status=="no", so a stale reason surviving under status='not_nominated'
// would leak a member's private decline reason to every team member.
func TestEventRepository_SetNomination_ClearsStaleReason(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	member := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Nomination Reason Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES
		($1, 'Member', 'member-reason@example.com', '#abcdef')`, member)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, member)
	require.NoError(t, err)

	// member needs events:write to call SetNomination (never self-service,
	// now re-verified atomically).
	var writeRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Reason Organizer', '{"events":"write"}') RETURNING id`,
		teamID,
	).Scan(&writeRoleID)
	require.NoError(t, err)
	var memberMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, member).Scan(&memberMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, memberMembershipID, writeRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Nomination Reason Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	status := "no"
	reason := "private medical reason"
	_, err = repo.SetAttendance(ctx, eventID, member.String(), member.String(), teamID.String(), &status, &reason, nil, nil)
	require.NoError(t, err)

	err = repo.SetNomination(ctx, eventID, member.String(), member.String(), teamID.String(), false)
	require.NoError(t, err)

	attendanceList, err := repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, attendanceList, 1)
	assert.Equal(t, "not_nominated", attendanceList[0].Status)
	assert.Nil(t, attendanceList[0].Reason, "reason must be cleared once nomination is revoked, not just left stale under a new status")
}

// TestEventRepository_SetNomination_RejectsCallerWithoutEventsWrite regression-
// tests the TOCTOU race deferred in round 68 alongside SetAttendance's
// equivalent fix: SetNomination is never self-service (events.Service.
// SetNomination requires events:write unconditionally, with no caller-equals-
// target bypass), so a concurrent SetRoles/DeleteRole/UpdateRole revoking the
// caller's events:write between the service layer's unlocked permission read
// and this write must not let either the upsert (nominated=false) or delete
// (nominated=true) branch through. Calls the repository directly, bypassing
// the service layer's own (separately racy) check, to prove the write itself
// is guarded.
func TestEventRepository_SetNomination_RejectsCallerWithoutEventsWrite(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Nomination No Write Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
			($1, 'Caller', 'nom-caller-nowrite@example.com', '#111111'),
			($2, 'Target', 'nom-target-nowrite@example.com', '#222222')
	`, callerID, targetID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)`, teamID, callerID, targetID)
	require.NoError(t, err)

	// Caller has a role, but it grants events:read, not events:write.
	var readOnlyRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Nomination Read Only', '{"events":"read"}') RETURNING id`,
		teamID,
	).Scan(&readOnlyRoleID)
	require.NoError(t, err)
	var callerMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, callerID).Scan(&callerMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, callerMembershipID, readOnlyRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Nomination No Write Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	// nominated=false (upsert branch) must be rejected.
	err = repo.SetNomination(ctx, eventID, callerID.String(), targetID.String(), teamID.String(), false)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(false) must reject a caller without events:write, even called directly")

	// ListAttendance is roster-based (both callerID and targetID are
	// current team members, so both always appear); the regression to
	// guard is that targetID's row wasn't flipped to not_nominated by the
	// rejected write.
	attendanceList, err := repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, attendanceList, 2)
	targetRow := findAttendanceRow(attendanceList, targetID)
	require.NotNil(t, targetRow)
	assert.NotEqual(t, "not_nominated", targetRow.Status, "no attendance row should have been created by the rejected upsert")

	// Seed a not_nominated row directly (bypassing the guard) so the delete
	// branch below has something to try to delete.
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status, at) VALUES ($1, $2, 'not_nominated', now())`, eventID, targetID)
	require.NoError(t, err)

	// nominated=true (delete branch) must also be rejected.
	err = repo.SetNomination(ctx, eventID, callerID.String(), targetID.String(), teamID.String(), true)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(true) must reject a caller without events:write, even called directly")

	attendanceList, err = repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, attendanceList, 2)
	targetRow = findAttendanceRow(attendanceList, targetID)
	require.NotNil(t, targetRow)
	assert.Equal(t, "not_nominated", targetRow.Status, "the seeded not_nominated row must survive the rejected delete")
}

// Companion positive test: a caller WITH events:write can set/clear another
// member's nomination, confirming the new predicate isn't overly restrictive.
func TestEventRepository_SetNomination_AllowsCallerWithEventsWrite(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := events.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Nomination Write Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color) VALUES
			($1, 'Organizer', 'nom-caller-write@example.com', '#111111'),
			($2, 'Target', 'nom-target-write@example.com', '#222222')
	`, callerID, targetID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)`, teamID, callerID, targetID)
	require.NoError(t, err)

	var writeRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Nomination Organizer', '{"events":"write"}') RETURNING id`,
		teamID,
	).Scan(&writeRoleID)
	require.NoError(t, err)
	var callerMembershipID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, callerID).Scan(&callerMembershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, callerMembershipID, writeRoleID)
	require.NoError(t, err)

	params := makeCreateParams("Nomination Write Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	err = repo.SetNomination(ctx, eventID, callerID.String(), targetID.String(), teamID.String(), false)
	require.NoError(t, err, "SetNomination(false) with events:write must succeed")

	// ListAttendance is roster-based: both callerID and targetID are
	// current team members and always appear; only targetID's status
	// should flip to not_nominated.
	attendanceList, err := repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, attendanceList, 2)
	targetRow := findAttendanceRow(attendanceList, targetID)
	require.NotNil(t, targetRow)
	assert.Equal(t, "not_nominated", targetRow.Status)

	err = repo.SetNomination(ctx, eventID, callerID.String(), targetID.String(), teamID.String(), true)
	require.NoError(t, err, "SetNomination(true) with events:write must succeed")

	attendanceList, err = repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	require.Len(t, attendanceList, 2)
	targetRow = findAttendanceRow(attendanceList, targetID)
	require.NotNil(t, targetRow)
	assert.NotEqual(t, "not_nominated", targetRow.Status, "not_nominated row must be removed")
}
