package stats_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

// An event flagged exclude_from_stats must be completely absent from every
// statistics view (member quotas, per-event stats, the attendance matrix) --
// as if it didn't exist for statistics purposes -- while a normal event in
// the same range is still counted, confirming the new predicate doesn't
// accidentally over-filter.
func TestStatsRepository_ExcludesEventFlaggedExcludeFromStats(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Excl Stats User', 'exclstats@example.com', '#778899')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Excl Stats Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var counted, excluded string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status, exclude_from_stats)
		 VALUES ($1, 'training', 'Counted Training', $2, 'active', false) RETURNING id`,
		tid, today).Scan(&counted)
	require.NoError(t, err)
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status, exclude_from_stats)
		 VALUES ($1, 'training', 'GL Training', $2, 'active', true) RETURNING id`,
		tid, today).Scan(&excluded)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, counted, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, excluded, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	teamID := uuid.MustParse(tid)

	memberRows, err := repo.MemberStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, memberRows, 1)
	assert.Equal(t, 1, memberRows[0].Yes, "excluded event must not inflate the member's yes count")
	assert.Equal(t, 1, memberRows[0].Counted, "excluded event must not inflate the member's counted total")

	eventRows, err := repo.EventStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, eventRows, 1, "excluded event must not appear in per-event stats")
	assert.Equal(t, "Counted Training", eventRows[0].Title)

	cols, cells, err := repo.AttendanceMatrix(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, cols, 1, "excluded event must not appear as a matrix column")
	assert.Equal(t, "Counted Training", cols[0].Title)
	require.Len(t, cells, 1, "one member x one non-excluded event")
	require.NotNil(t, cells[0].EventID)
	assert.Equal(t, uuid.MustParse(counted), *cells[0].EventID)

	single, err := repo.SingleMemberStats(ctx, teamID, uuid.MustParse(uid), from, to)
	require.NoError(t, err)
	assert.Equal(t, 1, single.Yes)
	assert.Equal(t, 1, single.Counted)
}

// An absence flagged not_relevant_for_stats must exclude its covered event
// date entirely from statistics (neither attending nor absent), across every
// stats view -- as opposed to a normal absence in the same range, which
// still counts as "no" (the pre-existing default). The matrix cell for the
// excluded date reads 'pending' (not a new wire value -- see
// attendance/sql.go's NotRelevantAbsenceCoversExpr doc comment), but is
// still excluded from the row's Yes/Counted the same way.
func TestStatsRepository_NotRelevantAbsence_ExcludedFromStats(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	excludedUID := uuid.New().String()
	normalUID := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Not Relevant User', 'notrelevant@example.com', '#0011ff')`, excludedUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Normal Absence User', 'normalabsence@example.com', '#ff1100')`, normalUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Not Relevant Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, excludedUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, normalUID)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Covered Both Ways', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date, not_relevant_for_stats) VALUES ($1, $2, $3, $3, true)`,
		tid, excludedUID, today)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date, not_relevant_for_stats) VALUES ($1, $2, $3, $3, false)`,
		tid, normalUID, today)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	teamID := uuid.MustParse(tid)

	memberRows, err := repo.MemberStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, memberRows, 2)
	for _, m := range memberRows {
		if m.UserID.String() == excludedUID {
			assert.Zero(t, m.Counted, "not-relevant absence must not count toward this member's quote at all")
		} else {
			assert.Equal(t, 1, m.Counted, "a normal absence must still count as one 'no'")
			assert.Zero(t, m.Yes)
		}
	}

	single, err := repo.SingleMemberStats(ctx, teamID, uuid.MustParse(excludedUID), from, to)
	require.NoError(t, err)
	assert.Zero(t, single.Counted)

	eventRows, err := repo.EventStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, eventRows, 1)
	assert.Equal(t, 1, eventRows[0].Counted, "only the normal absence counts toward this event's turnout")

	_, cells, err := repo.AttendanceMatrix(ctx, teamID, from, to)
	require.NoError(t, err)
	byUser := map[string]stats.MatrixCellRow{}
	for _, c := range cells {
		byUser[c.UserID.String()] = c
	}
	require.Contains(t, byUser, excludedUID)
	assert.Equal(t, "pending", byUser[excludedUID].Eff, "excluded cell reads as pending, not a new wire value")
	require.Contains(t, byUser, normalUID)
	assert.Equal(t, "no", byUser[normalUID].Eff)
}

// An explicit attendance response for the same date a not-relevant absence
// covers must win -- the CASE in every stats query only falls into the
// 'excluded' branch when a.status IS NULL, so a member who is flagged
// not-relevant but still explicitly responded (e.g. showed up anyway) is
// counted on their real response, not silently dropped.
func TestStatsRepository_NotRelevantAbsence_ExplicitResponseOverrides(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Overrides User', 'overrides@example.com', '#00ffaa')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Overrides Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Showed Up Anyway', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date, not_relevant_for_stats) VALUES ($1, $2, $3, $3, true)`,
		tid, uid, today)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, eid, uid)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	teamID := uuid.MustParse(tid)

	memberRows, err := repo.MemberStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, memberRows, 1)
	assert.Equal(t, 1, memberRows[0].Counted, "an explicit 'yes' must count even though the date is covered by a not-relevant absence")
	assert.Equal(t, 1, memberRows[0].Yes)

	_, cells, err := repo.AttendanceMatrix(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, cells, 1)
	assert.Equal(t, "yes", cells[0].Eff, "the explicit response must win over the not-relevant absence in the matrix too")
}

// A membership flagged exclude_from_stats must be absent from every
// personal-quota-oriented statistics view (MemberStats, SingleMemberStats,
// the attendance matrix), while their past explicit response still counts
// toward EventStats' per-event turnout (a deliberate asymmetry -- see
// exclude-members-from-stats' design.md).
func TestStatsRepository_ExcludedMember_OmittedFromPersonalQuotas_ButCountedInEventStats(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := stats.NewRepository(pool)
	ctx := context.Background()

	excludedUID := uuid.New().String()
	includedUID := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Excluded Member', 'excludedmember@example.com', '#aa00aa')`, excludedUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Included Member', 'includedmember@example.com', '#00aa00')`, includedUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Excluded Member Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id, exclude_from_stats) VALUES ($1, $2, true)`, tid, excludedUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id, exclude_from_stats) VALUES ($1, $2, false)`, tid, includedUID)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	var eid string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Turnout Training', $2, 'active') RETURNING id`,
		tid, today).Scan(&eid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, eid, excludedUID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, 'yes')`, eid, includedUID)
	require.NoError(t, err)

	// A second event both members miss via a covering planned absence (no
	// explicit response), to exercise EventStats' own exclude_from_stats
	// filter across more than one event.
	var missedID string
	err = pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Missed Training', $2, 'active') RETURNING id`,
		tid, today).Scan(&missedID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO absences (team_id, user_id, from_date, to_date) VALUES ($1, $2, $3, $3)`, tid, excludedUID, today)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO absences (team_id, user_id, from_date, to_date) VALUES ($1, $2, $3, $3)`, tid, includedUID, today)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	teamID := uuid.MustParse(tid)

	memberRows, err := repo.MemberStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, memberRows, 1, "excluded member must be omitted from the roster")
	assert.Equal(t, includedUID, memberRows[0].UserID.String())

	// An excluded member is not an invalid target -- their single-member view
	// returns the same "no data" shape as a member with no events in range
	// (not an error), and must not leak their real 'yes' response recorded
	// above for the Turnout Training event.
	excludedStat, err := repo.SingleMemberStats(ctx, teamID, uuid.MustParse(excludedUID), from, to)
	require.NoError(t, err)
	assert.Equal(t, 0, excludedStat.Yes)
	assert.Equal(t, 0, excludedStat.Counted)

	_, cells, err := repo.AttendanceMatrix(ctx, teamID, from, to)
	require.NoError(t, err)
	for _, c := range cells {
		assert.NotEqual(t, excludedUID, c.UserID.String(), "excluded member must have no matrix row")
	}

	eventRows, err := repo.EventStats(ctx, teamID, from, to)
	require.NoError(t, err)
	require.Len(t, eventRows, 2)
	var turnout *stats.EventStatRow
	for i := range eventRows {
		if eventRows[i].EventID.String() == eid {
			turnout = &eventRows[i]
		}
	}
	require.NotNil(t, turnout, "sanity check: the turnout event is still reported")
	assert.Equal(t, 2, turnout.Yes, "excluded member's past response still counts toward this event's turnout")
}

// ─── per-group (role) breakdown + thresholds ─────────────────────────────────

// groupFixture seeds one team with two roles ("Trainer" system role,
// "Spieler" custom role) and returns helpers to add members/events.
type groupFixture struct {
	t       *testing.T
	pool    *pgxpool.Pool
	teamID  uuid.UUID
	trainer uuid.UUID
	spieler uuid.UUID
	from    string
	to      string
	today   string
}

func newGroupFixture(t *testing.T) *groupFixture {
	t.Helper()
	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	f := &groupFixture{
		t:      t,
		pool:   pool,
		teamID: uuid.New(),
		from:   time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02"),
		to:     time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02"),
		today:  time.Now().UTC().Format("2006-01-02"),
	}
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Group Team')`, f.teamID)
	require.NoError(t, err)
	f.trainer = f.addRole(f.teamID, "Trainer", true)
	f.spieler = f.addRole(f.teamID, "Spieler", false)
	return f
}

func (f *groupFixture) addRole(teamID uuid.UUID, name string, system bool) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(context.Background(),
		`INSERT INTO roles (team_id, name, system, color) VALUES ($1, $2, $3, '#123456') RETURNING id`,
		teamID, name, system).Scan(&id)
	require.NoError(f.t, err)
	return id
}

// addMember creates a user + membership in the fixture team holding roles.
func (f *groupFixture) addMember(name string, roles ...uuid.UUID) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	uid := uuid.New()
	_, err := f.pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, $2, $3, '#112233')`,
		uid, name, uid.String()+"@example.com")
	require.NoError(f.t, err)
	var mid uuid.UUID
	err = f.pool.QueryRow(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, f.teamID, uid).Scan(&mid)
	require.NoError(f.t, err)
	for _, r := range roles {
		_, err = f.pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid, r)
		require.NoError(f.t, err)
	}
	return uid
}

func (f *groupFixture) addEvent(title, responseMode string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(context.Background(),
		`INSERT INTO events (team_id, type, title, date, status, response_mode)
		 VALUES ($1, 'training', $2, $3, 'active', $4) RETURNING id`,
		f.teamID, title, f.today, responseMode).Scan(&id)
	require.NoError(f.t, err)
	return id
}

func (f *groupFixture) respond(eventID, userID uuid.UUID, status string) {
	f.t.Helper()
	_, err := f.pool.Exec(context.Background(),
		`INSERT INTO attendance (event_id, user_id, status) VALUES ($1, $2, $3)`, eventID, userID, status)
	require.NoError(f.t, err)
}

func groupByRole(rows []stats.EventGroupStatRow, eventID uuid.UUID) map[uuid.UUID]stats.EventGroupStatRow {
	out := map[uuid.UUID]stats.EventGroupStatRow{}
	for _, r := range rows {
		if r.EventID == eventID {
			out[r.RoleID] = r
		}
	}
	return out
}

func TestStatsRepository_EventGroupStats_CountsPerRole(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	// coach holds both roles -> counts in both groups.
	coach := f.addMember("Coach", f.trainer, f.spieler)
	assistant := f.addMember("Assistant", f.trainer)
	p1 := f.addMember("Player 1", f.spieler)
	p2 := f.addMember("Player 2", f.spieler)
	p3 := f.addMember("Player 3", f.spieler)
	// A member with no role at all appears in no group (but still in the
	// event's own turnout).
	roleless := f.addMember("Roleless")

	ev := f.addEvent("Training", "opt_in")
	f.respond(ev, coach, "yes")
	f.respond(ev, assistant, "yes")
	f.respond(ev, p1, "yes")
	f.respond(ev, p2, "no")
	f.respond(ev, roleless, "yes")
	// p3 never responds on an opt-in event -> pending, in neither number.
	_ = p3

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, rows, 2, "one row per (event, role with at least one member)")

	// Ordered like the role list: system role first.
	assert.Equal(t, f.trainer, rows[0].RoleID)
	assert.Equal(t, f.spieler, rows[1].RoleID)

	byRole := groupByRole(rows, ev)
	tr := byRole[f.trainer]
	assert.Equal(t, "Trainer", tr.Name)
	require.NotNil(t, tr.Color)
	assert.Equal(t, "#123456", *tr.Color)
	assert.Equal(t, 2, tr.Yes, "both trainers attended")
	assert.Equal(t, 2, tr.Counted)

	assert.Equal(t, 0, tr.Pending, "every trainer responded")

	sp := byRole[f.spieler]
	assert.Equal(t, 2, sp.Yes, "coach (two roles) + player 1")
	assert.Equal(t, 3, sp.Counted, "coach, player 1, player 2 -- the pending player 3 is not counted")
	assert.Equal(t, 1, sp.Pending, "player 3 gave no response: reported apart from counted, so the line-up can treat it as not present")

	// Unconfigured roles resolve to the default threshold.
	assert.Equal(t, stats.DefaultThresholdMode, tr.ThresholdMode)
	assert.Equal(t, stats.DefaultThresholdValue, tr.ThresholdValue)

	// The event's own turnout still de-duplicates the two-role coach and
	// includes the roleless member.
	evRows, err := repo.EventStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, evRows, 1)
	assert.Equal(t, 4, evRows[0].Yes)
	assert.Equal(t, 5, evRows[0].Counted)
	assert.Equal(t, 1, evRows[0].Pending, "player 3 is the event's only non-responder")
}

// A member who was not nominated for the event was not expected at it: they
// are neither counted nor pending, in the group and on the event row alike.
func TestStatsRepository_EventGroupStats_NotNominatedIsNeitherCountedNorPending(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	nominated := f.addMember("Nominated", f.spieler)
	benched := f.addMember("Benched", f.spieler)
	silent := f.addMember("Silent", f.spieler)
	_ = silent

	ev := f.addEvent("Match", "opt_in")
	f.respond(ev, nominated, "yes")
	f.respond(ev, benched, "not_nominated")

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].Yes)
	assert.Equal(t, 1, rows[0].Counted)
	assert.Equal(t, 1, rows[0].Pending, "only the silent member is pending; the benched one was not expected")

	evRows, err := repo.EventStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, evRows, 1)
	assert.Equal(t, 1, evRows[0].Yes)
	assert.Equal(t, 1, evRows[0].Counted)
	assert.Equal(t, 1, evRows[0].Pending)
}

// A group in which every member holds the same role reconciles exactly with
// the event row, across the effective-status edge cases the two queries
// share: opt-out default, a covering absence, a not-relevant absence, an
// excluded-from-stats member (still counted in per-event turnout).
func TestStatsRepository_EventGroupStats_ReconcilesWithEventStats(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	silent := f.addMember("Silent", f.spieler)     // opt-out default -> yes
	absent := f.addMember("Absent", f.spieler)     // covering absence -> no
	onLeave := f.addMember("On leave", f.spieler)  // not-relevant absence -> excluded
	excluded := f.addMember("Excluded", f.spieler) // exclude_from_stats, explicit yes
	declined := f.addMember("Declined", f.spieler) // explicit no
	_ = silent

	_, err := f.pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date) VALUES ($1, $2, $3, $3)`,
		f.teamID, absent, f.today)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date, not_relevant_for_stats) VALUES ($1, $2, $3, $3, true)`,
		f.teamID, onLeave, f.today)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx,
		`UPDATE memberships SET exclude_from_stats = true WHERE team_id = $1 AND user_id = $2`, f.teamID, excluded)
	require.NoError(t, err)

	ev := f.addEvent("Opt-out training", "opt_out")
	f.respond(ev, excluded, "yes")
	f.respond(ev, declined, "no")

	// Events outside the stats scope must not produce group rows either.
	var cancelled, flagged uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status, response_mode)
		 VALUES ($1, 'training', 'Cancelled', $2, 'cancelled', 'opt_out') RETURNING id`, f.teamID, f.today).Scan(&cancelled))
	require.NoError(t, f.pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status, response_mode, exclude_from_stats)
		 VALUES ($1, 'training', 'Flagged', $2, 'active', 'opt_out', true) RETURNING id`, f.teamID, f.today).Scan(&flagged))

	evRows, err := repo.EventStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, evRows, 1)

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, rows, 1, "only the Spieler role has members; cancelled/excluded events yield no rows")
	assert.Equal(t, ev, rows[0].EventID)
	assert.Equal(t, f.spieler, rows[0].RoleID)

	assert.Equal(t, 2, rows[0].Yes, "silent (opt-out default) + excluded member's explicit yes")
	assert.Equal(t, 4, rows[0].Counted, "silent, absent, excluded, declined -- the not-relevant absence is dropped")
	assert.Equal(t, evRows[0].Yes, rows[0].Yes, "group must reconcile with the event row")
	assert.Equal(t, evRows[0].Counted, rows[0].Counted, "group must reconcile with the event row")
	// An opt-out event has no pending members by definition: a missing
	// response defaults to attending (or to "no" under an absence).
	assert.Equal(t, 0, rows[0].Pending)
	assert.Equal(t, evRows[0].Pending, rows[0].Pending, "group must reconcile with the event row")
}

func TestStatsRepository_EventGroupStats_ScopedToTeamAndRange(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	p := f.addMember("Player", f.spieler)
	ev := f.addEvent("Training", "opt_in")
	f.respond(ev, p, "yes")

	// The same user is also a member of another team, with that team's own
	// role and event -- none of it may leak into this team's groups.
	otherTeam := uuid.New()
	_, err := f.pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Team')`, otherTeam)
	require.NoError(t, err)
	otherRole := f.addRole(otherTeam, "Other role", false)
	var otherMembership uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, otherTeam, p).Scan(&otherMembership))
	_, err = f.pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, otherMembership, otherRole)
	require.NoError(t, err)
	var otherEvent uuid.UUID
	require.NoError(t, f.pool.QueryRow(ctx,
		`INSERT INTO events (team_id, type, title, date, status) VALUES ($1, 'training', 'Other', $2, 'active') RETURNING id`,
		otherTeam, f.today).Scan(&otherEvent))
	f.respond(otherEvent, p, "yes")

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, ev, rows[0].EventID)
	assert.Equal(t, f.spieler, rows[0].RoleID)
	assert.Equal(t, 1, rows[0].Yes)
	assert.Equal(t, 1, rows[0].Counted)

	// Out of range -> nothing.
	past := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	rows, err = repo.EventGroupStats(ctx, f.teamID, past, past)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestStatsRepository_GroupThresholds_DefaultUpsertAndOrder(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	// Nothing configured: every role, default values, system role first.
	rows, err := repo.ListGroupThresholds(ctx, f.teamID)
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{
		{RoleID: f.trainer, Mode: "percent", Value: 50},
		{RoleID: f.spieler, Mode: "percent", Value: 50},
	}, rows)

	// Saving one role leaves the other untouched and returns the full list.
	rows, err = repo.SetGroupThresholds(ctx, f.teamID, []stats.GroupThresholdRow{{RoleID: f.spieler, Mode: "count", Value: 8}})
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{
		{RoleID: f.trainer, Mode: "percent", Value: 50},
		{RoleID: f.spieler, Mode: "count", Value: 8},
	}, rows)

	// A second save overwrites (upsert), and can change mode.
	_, err = repo.SetGroupThresholds(ctx, f.teamID, []stats.GroupThresholdRow{
		{RoleID: f.spieler, Mode: "percent", Value: 75},
		{RoleID: f.trainer, Mode: "count", Value: 1},
	})
	require.NoError(t, err)
	rows, err = repo.ListGroupThresholds(ctx, f.teamID)
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{
		{RoleID: f.trainer, Mode: "count", Value: 1},
		{RoleID: f.spieler, Mode: "percent", Value: 75},
	}, rows)

	// The stored threshold is what the overview's group rows carry.
	p := f.addMember("Player", f.spieler)
	ev := f.addEvent("Training", "opt_in")
	f.respond(ev, p, "yes")
	groups, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "percent", groups[0].ThresholdMode)
	assert.Equal(t, 75, groups[0].ThresholdValue)

	// Items are applied whatever order they arrive in (the repository
	// upserts in role-id order internally to keep concurrent saves from
	// deadlocking), and the caller's slice is left exactly as it was.
	in := []stats.GroupThresholdRow{
		{RoleID: f.spieler, Mode: "count", Value: 9},
		{RoleID: f.trainer, Mode: "percent", Value: 10},
	}
	reversed := []stats.GroupThresholdRow{in[1], in[0]}
	for _, order := range [][]stats.GroupThresholdRow{in, reversed} {
		snapshot := append([]stats.GroupThresholdRow(nil), order...)
		rows, err = repo.SetGroupThresholds(ctx, f.teamID, order)
		require.NoError(t, err)
		assert.Equal(t, snapshot, order, "the caller's slice must not be reordered")
		assert.Equal(t, []stats.GroupThresholdRow{
			{RoleID: f.trainer, Mode: "percent", Value: 10},
			{RoleID: f.spieler, Mode: "count", Value: 9},
		}, rows)
	}

	// An empty save is a no-op that still returns the list.
	rows, err = repo.SetGroupThresholds(ctx, f.teamID, nil)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

// A role of another team (or one that doesn't exist) rejects the whole
// request: nothing is written, not even for the valid role listed first.
func TestStatsRepository_SetGroupThresholds_ForeignRoleRejectedAtomically(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	otherTeam := uuid.New()
	_, err := f.pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Team')`, otherTeam)
	require.NoError(t, err)
	foreign := f.addRole(otherTeam, "Foreign", false)

	for name, bad := range map[string]uuid.UUID{"role of another team": foreign, "unknown role": uuid.New()} {
		_, err = repo.SetGroupThresholds(ctx, f.teamID, []stats.GroupThresholdRow{
			{RoleID: f.spieler, Mode: "count", Value: 8},
			{RoleID: bad, Mode: "count", Value: 3},
		})
		require.ErrorIs(t, err, stats.ErrRoleNotInTeam, name)
	}

	rows, err := repo.ListGroupThresholds(ctx, f.teamID)
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{
		{RoleID: f.trainer, Mode: "percent", Value: 50},
		{RoleID: f.spieler, Mode: "percent", Value: 50},
	}, rows, "the valid item ahead of the rejected one must have been rolled back")

	other, err := repo.ListGroupThresholds(ctx, otherTeam)
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{{RoleID: foreign, Mode: "percent", Value: 50}}, other,
		"the other team's role must be untouched")

	var n int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM role_stats_thresholds`).Scan(&n))
	assert.Zero(t, n)
}

func TestStatsRepository_GroupThresholds_DeletedWithRole(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	_, err := repo.SetGroupThresholds(ctx, f.teamID, []stats.GroupThresholdRow{{RoleID: f.spieler, Mode: "count", Value: 8}})
	require.NoError(t, err)

	_, err = f.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, f.spieler)
	require.NoError(t, err)

	var n int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM role_stats_thresholds WHERE role_id = $1`, f.spieler).Scan(&n))
	assert.Zero(t, n, "ON DELETE CASCADE must remove the threshold with its role")

	rows, err := repo.ListGroupThresholds(ctx, f.teamID)
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{{RoleID: f.trainer, Mode: "percent", Value: 50}}, rows)
}

// The DB CHECKs are the backstop behind stats.Service's validation.
func TestStatsRepository_SetGroupThresholds_DBChecksRejectOutOfRange(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	for name, bad := range map[string]stats.GroupThresholdRow{
		"unknown mode":      {RoleID: f.spieler, Mode: "ratio", Value: 1},
		"percent above 100": {RoleID: f.spieler, Mode: "percent", Value: 101},
		"negative":          {RoleID: f.spieler, Mode: "count", Value: -1},
		"count above cap":   {RoleID: f.spieler, Mode: "count", Value: 10001},
	} {
		_, err := repo.SetGroupThresholds(ctx, f.teamID, []stats.GroupThresholdRow{bad})
		require.Error(t, err, name)
		assert.NotErrorIs(t, err, stats.ErrRoleNotInTeam, name)
	}
}

// The events module stores role nominations only as events.nominated_role_ids
// (no not_nominated attendance rows), so the stats queries must treat a
// member holding none of the nominated roles as not expected themselves --
// otherwise every non-nominated group would show up as "0 % present".
func TestStatsRepository_EventGroupStats_RoleNominationMakesOthersNotExpected(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	p1 := f.addMember("Player 1", f.spieler)
	p2 := f.addMember("Player 2", f.spieler)
	f.addMember("Player 3", f.spieler) // nominated, silent -> pending
	coach := f.addMember("Coach", f.trainer)
	f.addMember("Assistant", f.trainer) // not nominated, silent -> not expected
	both := f.addMember("Playing coach", f.trainer, f.spieler)

	ev := f.addEvent("Match", "opt_in")
	_, err := f.pool.Exec(ctx, `UPDATE events SET nominated_role_ids = ARRAY[$1::uuid] WHERE id = $2`, f.spieler, ev)
	require.NoError(t, err)
	f.respond(ev, p1, "yes")
	f.respond(ev, p2, "no")
	f.respond(ev, both, "yes")
	// A non-nominated member who answers anyway still counts: an explicit
	// response wins, as everywhere else.
	f.respond(ev, coach, "yes")

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	byRole := groupByRole(rows, ev)

	sp := byRole[f.spieler]
	assert.Equal(t, 2, sp.Yes, "player 1 + playing coach")
	assert.Equal(t, 3, sp.Counted)
	assert.Equal(t, 1, sp.Pending, "only the nominated, silent player 3")

	tr := byRole[f.trainer]
	assert.Equal(t, 2, tr.Yes, "coach answered although not nominated; playing coach is nominated via Spieler")
	assert.Equal(t, 2, tr.Counted)
	assert.Equal(t, 0, tr.Pending, "the silent, non-nominated assistant was not expected")

	evRows, err := repo.EventStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, evRows, 1)
	assert.Equal(t, 3, evRows[0].Yes)
	assert.Equal(t, 4, evRows[0].Counted)
	assert.Equal(t, 1, evRows[0].Pending)

	// An empty nomination list means everyone is nominated.
	_, err = f.pool.Exec(ctx, `UPDATE events SET nominated_role_ids = '{}' WHERE id = $1`, ev)
	require.NoError(t, err)
	evRows, err = repo.EventStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	assert.Equal(t, 2, evRows[0].Pending, "player 3 and the assistant")

	// The non-nominated rule also beats an opt-out default: a member who was
	// not nominated is not assumed to attend.
	_, err = f.pool.Exec(ctx, `UPDATE events SET nominated_role_ids = ARRAY[$1::uuid], response_mode = 'opt_out' WHERE id = $2`, f.spieler, ev)
	require.NoError(t, err)
	rows, err = repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	tr = groupByRole(rows, ev)[f.trainer]
	assert.Equal(t, 2, tr.Counted, "the silent assistant is neither an assumed yes nor pending")
	assert.Equal(t, 0, tr.Pending)
}

// A member flagged exclude_from_stats stays in per-event turnout so that
// their explicit responses still count -- but their silence must not count
// against the event or their groups.
func TestStatsRepository_EventGroupStats_ExcludedMemberSilenceIsNotPending(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	active := f.addMember("Active", f.spieler)
	passive := f.addMember("Passive", f.spieler)
	answering := f.addMember("Answering excluded", f.spieler)
	_, err := f.pool.Exec(ctx,
		`UPDATE memberships SET exclude_from_stats = true WHERE team_id = $1 AND user_id = ANY($2)`,
		f.teamID, []uuid.UUID{passive, answering})
	require.NoError(t, err)

	ev := f.addEvent("Training", "opt_in")
	f.respond(ev, active, "yes")
	f.respond(ev, answering, "yes")

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 2, rows[0].Yes, "the excluded member's explicit yes still counts")
	assert.Equal(t, 2, rows[0].Counted)
	assert.Equal(t, 0, rows[0].Pending, "the silent excluded member is not pending")

	evRows, err := repo.EventStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, evRows, 1)
	assert.Equal(t, 0, evRows[0].Pending)
}

// On an opt-in event: a covering absence makes a silent member "no"
// (counted, not pending), a not-relevant absence makes them not expected.
func TestStatsRepository_EventGroupStats_AbsencesOnOptInEvent(t *testing.T) {
	t.Parallel()

	f := newGroupFixture(t)
	repo := stats.NewRepository(f.pool)
	ctx := context.Background()

	present := f.addMember("Present", f.spieler)
	away := f.addMember("Away", f.spieler)
	onLeave := f.addMember("On leave", f.spieler)
	f.addMember("Silent", f.spieler)

	_, err := f.pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date) VALUES ($1, $2, $3, $3)`, f.teamID, away, f.today)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx,
		`INSERT INTO absences (team_id, user_id, from_date, to_date, not_relevant_for_stats) VALUES ($1, $2, $3, $3, true)`,
		f.teamID, onLeave, f.today)
	require.NoError(t, err)

	ev := f.addEvent("Training", "opt_in")
	f.respond(ev, present, "yes")

	rows, err := repo.EventGroupStats(ctx, f.teamID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].Yes)
	assert.Equal(t, 2, rows[0].Counted, "present (yes) + away (absence -> no)")
	assert.Equal(t, 1, rows[0].Pending, "only the silent member; the not-relevant absence is not expected")
}
