package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
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

	params := makeCreateParams("Match Day", time.Now().UTC())
	ev, err := repo.CreateEvent(ctx, teamID, &params)
	require.NoError(t, err)

	eventID := ev.Id.String()

	// First upsert: yes.
	status := "yes"
	rec, err := repo.SetAttendance(ctx, eventID, userID, &status, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "yes", rec.Status)
	firstID := rec.Id

	// Second upsert: change to no — should update, not insert.
	status = "no"
	reason := "sick"
	rec2, err := repo.SetAttendance(ctx, eventID, userID, &status, &reason, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, rec2)
	assert.Equal(t, "no", rec2.Status)
	assert.Equal(t, &reason, rec2.Reason)
	// Verify same logical row (UNIQUE constraint upheld).
	assert.Equal(t, firstID, rec2.Id, "upsert should update existing row, not insert new one")

	// Verify GetMyAttendance returns the latest record.
	myRec, err := repo.GetMyAttendance(ctx, eventID, userID)
	require.NoError(t, err)
	require.NotNil(t, myRec)
	assert.Equal(t, "no", myRec.Status)
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
	_, err = repo.SetAttendance(ctx, e1.Id.String(), userID.String(), &yes, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, e1.Id.String(), otherUserID.String(), &no, nil, nil, nil)
	require.NoError(t, err)
	_, err = repo.SetAttendance(ctx, e2.Id.String(), userID.String(), &no, nil, nil, nil)
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
	singleSummary, err := repo.GetAttendanceSummary(ctx, e1.Id.String())
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
