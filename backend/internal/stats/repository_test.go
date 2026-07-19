package stats_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/stats"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestStatsRepository_MemberStats(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Stats User', 'stats@example.com', '#112233')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Stats Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	// Seed an active event in the date range.
	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Training', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)

	// Seed a 'yes' attendance.
	_, err = pool.Exec(ctx,
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	teamID := uuid.MustParse(tid)
	rows, err := repo.MemberStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Stats User", rows[0].Name)
	assert.Equal(t, 1, rows[0].Yes)
	assert.Equal(t, 1, rows[0].Counted)
}

func TestStatsRepository_EventStats(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Evt Stats User', 'evtstats@example.com', '#334455')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Evt Stats Team')`, tid)
	require.NoError(t, err)
	// EventStats is roster-driven: only current members are scored. The user
	// must be a member for their attendance to count.
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Game Day', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	teamID := uuid.MustParse(tid)
	rows, err := repo.EventStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Game Day", rows[0].Title)
	// Regression: the query used to omit e.type entirely, leaving
	// EventStatRow.Type at its Go zero value ("") even though
	// gen.EventStat.Type is a required field of the API contract.
	assert.Equal(t, "training", rows[0].Type)
	assert.Equal(t, 1, rows[0].Yes)
}

// An opt_out event with no explicit response must count the member as
// attending in the statistics, matching the event summary. Explicit-only
// counting previously reported 0% here while the event showed the member as
// attending.
func TestStatsRepository_MemberStats_OptOutDefaultsToAttending(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'OptOut User', 'optout@example.com', '#010203')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'OptOut Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	_, err = pool.Exec(ctx,
		`INSERT INTO events (team_id, type, title, date, status, response_mode) VALUES ($1, 'training', 'Opt-Out Training', $2, 'active', 'opt_out')`,
		tid, today)
	require.NoError(t, err)
	// Deliberately NO attendance row: the member never responded.

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.MemberStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].Yes, "opt_out with no response must default to attending")
	assert.Equal(t, 1, rows[0].Counted)
}

// A planned absence covering the event date defaults the member to "no"
// (counted, not attending), matching the event summary.
func TestStatsRepository_MemberStats_AbsenceDefaultsToNotAttending(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Absent User', 'absent@example.com', '#040506')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Absence Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	_, err = pool.Exec(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Covered Training', $2, 'active')`,
		tid, today)
	require.NoError(t, err)
	// A covering planned absence, no explicit attendance.
	_, err = pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date) VALUES ($1, $2, $3, $3)`, tid, uid, today)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.MemberStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 0, rows[0].Yes, "a covering absence must not count as attending")
	assert.Equal(t, 1, rows[0].Counted, "a covering absence is a counted 'no'")
}

func TestStatsRepository_SingleMemberStats(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Single Stats User', 'single@example.com', '#aabbcc')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Single Stats Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Solo Training', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'maybe')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	s, err := repo.SingleMemberStats(ctx, uuid.MustParse(tid), uuid.MustParse(uid), from, to)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, "Single Stats User", s.Name)
	assert.Equal(t, 0, s.Yes)
	assert.Equal(t, 1, s.Counted)
}

func TestStatsRepository_SingleMemberStats_NonMemberBlocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Outsider User', 'outsider@example.com', '#aabbcc')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Home Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Other Team')`, otherTid)
	require.NoError(t, err)
	// The user is a member of tid, but NOT of otherTid.
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	// Querying this user's stats scoped to a team they don't belong to must fail.
	_, err = repo.SingleMemberStats(ctx, uuid.MustParse(otherTid), uuid.MustParse(uid), from, to)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
