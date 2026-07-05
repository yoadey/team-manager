package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/events"
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
	all, err := repo.ListEvents(ctx, testTeamID, "all", 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// List upcoming (both events are today or future).
	upcoming, err := repo.ListEvents(ctx, testTeamID, "upcoming", 50, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(upcoming), 1)

	// List past (no past events seeded).
	past, err := repo.ListEvents(ctx, testTeamID, "past", 50, nil)
	require.NoError(t, err)
	assert.Len(t, past, 0)
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
	err = pool.QueryRow(ctx,
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
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Own Role', '{}') RETURNING id`,
		teamID,
	).Scan(&ownRoleID)
	require.NoError(t, err)
	params.NominatedRoleIds = []uuid.UUID{ownRoleID}
	ev, err := repo.CreateEvent(ctx, teamID, &params)
	require.NoError(t, err)
	require.NotNil(t, ev)
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

	all, err := repo.ListEvents(ctx, teamID, "all", 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 3)

	dates := make(map[string]bool)
	for _, e := range all {
		assert.Equal(t, newTitle, e.Title, "title must apply to every occurrence in the series")
		dates[e.Date.Format("2006-01-02")] = true
	}
	assert.Len(t, dates, 3, "each occurrence must keep its own distinct date, not collapse onto the updated event's date")
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

	all, err := repo.ListEvents(ctx, teamID, "all", 50, nil)
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

	all, err := repo.ListEvents(ctx, teamID, "all", 50, nil)
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
	rec, err := repo.SetAttendance(ctx, eventID, userID, teamID, &status, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "yes", rec.Status)
	firstID := rec.Id

	// Second upsert: change to no — should update, not insert.
	status = "no"
	reason := "sick"
	rec2, err := repo.SetAttendance(ctx, eventID, userID, teamID, &status, &reason, nil, nil)
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
	_, err = repo.SetAttendance(ctx, ev.Id.String(), nonMemberID.String(), teamID.String(), &status, nil, nil, nil)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetAttendance must reject a userID that is not a member of teamID")
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
	err = pool.QueryRow(ctx,
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
	_, err = repo.SetAttendance(ctx, e1.Id.String(), userID.String(), teamID.String(), &yes, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, e1.Id.String(), otherUserID.String(), teamID.String(), &no, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, e2.Id.String(), userID.String(), teamID.String(), &no, nil, nil, nil)
	require.NoError(t, err)

	eventIDs := []uuid.UUID{e1.Id, e2.Id, e3.Id}

	summaries, err := repo.GetAttendanceSummaries(ctx, eventIDs)
	require.NoError(t, err)
	assert.Equal(t, 1, summaries[e1.Id].Yes)
	assert.Equal(t, 1, summaries[e1.Id].No)
	assert.Equal(t, 2, summaries[e1.Id].Total)
	assert.Equal(t, 1, summaries[e2.Id].No)
	assert.Equal(t, 1, summaries[e2.Id].Total)
	_, e3HasSummary := summaries[e3.Id]
	assert.False(t, e3HasSummary, "event with no attendance rows should be absent from the map")

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

	comments, err := repo.ListComments(ctx, eventID, teamB.String(), 50, 0)
	require.NoError(t, err)
	assert.Empty(t, comments, "ListComments must not see cross-team event's comments")

	// A member of Team B must not be able to write to Team A's event either.
	status := "yes"
	_, err = repo.SetAttendance(ctx, eventID, user.String(), teamB.String(), &status, nil, nil, nil)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetAttendance must reject cross-team eventID")

	err = repo.SetNomination(ctx, eventID, user.String(), teamB.String(), false)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(false) must reject cross-team eventID")

	err = repo.SetNomination(ctx, eventID, user.String(), teamB.String(), true)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(true) must reject cross-team eventID")

	_, err = repo.AddComment(ctx, eventID, user.String(), teamB.String(), "cross-team comment")
	assert.ErrorIs(t, err, pgx.ErrNoRows, "AddComment must reject cross-team eventID")

	// Verify no attendance/comment rows leaked through despite the rejected calls.
	attendanceList, err = repo.ListAttendance(ctx, eventID, teamA.String())
	require.NoError(t, err)
	assert.Empty(t, attendanceList, "no attendance row should have been created by the rejected cross-team SetAttendance call")

	comments, err = repo.ListComments(ctx, eventID, teamA.String(), 50, 0)
	require.NoError(t, err)
	assert.Empty(t, comments, "no comment should have been created by the rejected cross-team AddComment call")

	// Scoped to the correct team, all the same operations succeed.
	rec, err := repo.SetAttendance(ctx, eventID, user.String(), teamA.String(), &status, nil, nil, nil)
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

	params := makeCreateParams("Nomination Event", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID.String(), &params)
	require.NoError(t, err)
	eventID := ev.Id.String()

	err = repo.SetNomination(ctx, eventID, outsider.String(), teamID.String(), false)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "SetNomination(false) must reject a userID that is not a member of teamID")

	attendanceList, err := repo.ListAttendance(ctx, eventID, teamID.String())
	require.NoError(t, err)
	assert.Empty(t, attendanceList, "no attendance row should have been created for the non-member user")

	// A real member of the team is unaffected.
	err = repo.SetNomination(ctx, eventID, member.String(), teamID.String(), false)
	require.NoError(t, err, "SetNomination(false) scoped to an actual team member must succeed")
}
