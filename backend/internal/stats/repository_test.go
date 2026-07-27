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

func TestStatsRepository_AttendanceMatrix(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	alice, bob := uuid.New().String(), uuid.New().String()

	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Matrix Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Matrix Alice', 'malice@example.com', '#a1a1a1'), ($2, 'Matrix Bob', 'mbob@example.com', '#b2b2b2')`,
		alice, bob)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)`, tid, alice, bob)
	require.NoError(t, err)

	// Two events on different dates; the earlier one is an opt_out training.
	d1 := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")
	d2 := time.Now().UTC().Format("2006-01-02")
	var e1, e2 string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status, response_mode) VALUES ($1, 'training', 'Training One', $2, 'active', 'opt_out') RETURNING id`,
		tid, d1).Scan(&e1)
	require.NoError(t, err)
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'auftritt', 'Match Two', $2, 'active') RETURNING id`,
		tid, d2).Scan(&e2)
	require.NoError(t, err)

	// Alice: explicit 'no' on e2 (e1 opt_out with no response defaults to yes).
	// Bob: explicit 'yes' on e2 (e1 opt_out defaults to yes) → Bob attends both.
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'no')`, e2, alice)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, e2, bob)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -3).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	cols, cells, err := repo.AttendanceMatrix(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)

	// Columns ordered by date ascending.
	require.Len(t, cols, 2)
	assert.Equal(t, e1, cols[0].EventID.String())
	assert.Equal(t, "training", cols[0].Type)
	assert.Equal(t, e2, cols[1].EventID.String())

	// Cells: 2 members × 2 events = 4 rows. Resolve to a lookup for assertions.
	type key struct{ user, event string }
	got := map[key]string{}
	for _, c := range cells {
		require.NotNil(t, c.EventID)
		got[key{c.UserID.String(), c.EventID.String()}] = c.Eff
	}
	require.Len(t, got, 4)
	assert.Equal(t, "yes", got[key{alice, e1}], "opt_out with no response → yes")
	assert.Equal(t, "no", got[key{alice, e2}])
	assert.Equal(t, "yes", got[key{bob, e1}])
	assert.Equal(t, "yes", got[key{bob, e2}])
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

func TestStatsRepository_AbsenceStats_ExplicitNo(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Absent Explicit', 'absentexp@example.com', '#112233')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Absence Stats Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Missed Training', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'no')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.AbsenceStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Absent Explicit", rows[0].Name)
	assert.Equal(t, "Missed Training", rows[0].EventTitle)
	assert.Equal(t, today, rows[0].Date)
	assert.Equal(t, eid, rows[0].EventID.String())
}

// A covering planned absence must also surface as a row here, matching the
// "no" default MemberStats already applies via attendance.EffectiveStatusExpr.
func TestStatsRepository_AbsenceStats_CoveringPlannedAbsence(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Absent Planned', 'absentplanned@example.com', '#223344')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Absence Planned Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	_, err = pool.Exec(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Covered Training', $2, 'active')`,
		tid, today)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date) VALUES ($1, $2, $3, $3)`, tid, uid, today)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.AbsenceStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Covered Training", rows[0].EventTitle)
}

// A member who attended ('yes') must not show up in the absence table.
func TestStatsRepository_AbsenceStats_ExcludesAttended(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Attended User', 'attended@example.com', '#334455')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Attended Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Attended Training', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.AbsenceStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// An event outside the [from, to] date range must not contribute a row, even
// though it would otherwise be an absence.
func TestStatsRepository_AbsenceStats_RespectsDateRange(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Out Of Range', 'outofrange@example.com', '#445566')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Out Of Range Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	farPast := time.Now().UTC().AddDate(-1, 0, 0).Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Old Training', $2, 'active') RETURNING id`,
		tid, farPast).Scan(&eid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'no')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.AbsenceStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	assert.Empty(t, rows, "an absence for an event outside the date range must not be returned")
}

// A team-scoping guard: absences for a different team's member/event must
// never leak into this team's absence table.
func TestStatsRepository_AbsenceStats_TeamScoped(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Other Team Member', 'otherteam@example.com', '#556677')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Scoped Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Scoped Team')`, otherTid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, otherTid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	_, err = pool.Exec(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Other Team Training', $2, 'active')`,
		otherTid, today)
	require.NoError(t, err)
	// No attendance row -> would default per opt_out/pending rules, but since
	// the member isn't in tid's roster at all, tid's absence table must be
	// empty regardless.

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	rows, err := repo.AbsenceStats(ctx, uuid.MustParse(tid), from, to)
	require.NoError(t, err)
	assert.Empty(t, rows, "another team's events/members must not leak into this team's absence table")
}
