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
