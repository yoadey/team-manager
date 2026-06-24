package events_test

import (
	"context"
	"testing"
	"time"

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

func seedTeamAndUser(t *testing.T, pool interface {
	Exec(context.Context, string, ...any) (interface{ RowsAffected() int64 }, error)
},
) {
	t.Helper()
}

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
	all, err := repo.ListEvents(ctx, testTeamID, "all", 50, 0)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// List upcoming (both events are today or future).
	upcoming, err := repo.ListEvents(ctx, testTeamID, "upcoming", 50, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(upcoming), 1)

	// List past (no past events seeded).
	past, err := repo.ListEvents(ctx, testTeamID, "past", 50, 0)
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
