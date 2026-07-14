package absences_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

const (
	repoTestTeamID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	repoTestUserID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func TestAbsenceRepository_CreateAndList(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Absent User', 'abs@example.com', '#ff0000')`,
		repoTestUserID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Absence Team')`,
		repoTestTeamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		repoTestTeamID, repoTestUserID)
	require.NoError(t, err)

	teamID := uuid.MustParse(repoTestTeamID)
	userID := uuid.MustParse(repoTestUserID)

	from := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	reason := "holiday"

	ab, err := repo.Create(ctx, teamID, userID, from, to, &reason)
	require.NoError(t, err)
	require.NotNil(t, ab)
	assert.Equal(t, teamID, ab.TeamId)
	assert.Equal(t, userID, ab.UserId)
	assert.Equal(t, &reason, ab.Reason)
	assert.Equal(t, "Absent User", *ab.MemberName)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, ab.Id, all[0].Id)

	mine, err := repo.ListByUser(ctx, teamID, userID, 50, nil)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, ab.Id, mine[0].Id)
}

// TestAbsenceRepository_ListByTeam_ExcludesCrossTeamRole is a defense-in-
// depth regression test: absenceRoleJoins joined roles to membership_roles
// with no r.team_id check, unlike the established pattern elsewhere
// (members.getMembershipEffectivePermissionsQ,
// teams.Repository.GetRolesForMembership). Every current INSERT INTO
// membership_roles call site always inserts a role already validated as
// belonging to the target team, so this can't happen through normal API use
// today -- but this feeds the displayed RoleName/RoleColor on absence rows,
// so a future change that broke that insert-side invariant would silently
// show a different team's role name/color instead of failing safe.
func TestAbsenceRepository_ListByTeam_ExcludesCrossTeamRole(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	otherTeamID := uuid.New()
	foreignRoleID := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Cross Role Absent User', 'abs-crossrole@example.com', '#ff0000')`,
		userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Absence Own Team'), ($2, 'Absence Other Team')`,
		teamID, otherTeamID)
	require.NoError(t, err)
	var membershipID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, teamID, userID,
	).Scan(&membershipID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO roles (id, team_id, name) VALUES ($1, $2, 'Foreign Role')`, foreignRoleID, otherTeamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, membershipID, foreignRoleID)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	_, err = repo.Create(ctx, teamID, userID, from, to, nil)
	require.NoError(t, err)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Nil(t, all[0].RoleName, "a role belonging to a different team must never be surfaced, even if membership_roles points at it")
}

// Regression test: absenceRoleJoins fans an absence row out into one
// candidate row per custom role a member holds, and ListByTeam/ListByUser
// collapse that with SELECT DISTINCT ON (a.id) -- previously ORDER BY a.id
// alone, with no tiebreak among the fanned-out rows, so which role's
// name/color survived was Postgres's unspecified row order, not
// deterministic. A member with two custom roles could see a different role
// badge on their absence entry from one request to the next. This mirrors
// the primary-role-ordering bug round 80 fixed between members.Repository
// and events.batchGetPrimaryRoles: the fix here is the same convention
// (ORDER BY ..., r.id), applied to absences' own role join.
func TestAbsenceRepository_ListByTeam_RoleBadgeIsDeterministicForMultiRoleMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	roleA := uuid.New()
	roleB := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Multi Role Absent User', 'abs-multirole@example.com', '#ff0000')`,
		userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Absence Multi Role Team')`, teamID)
	require.NoError(t, err)
	var membershipID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, teamID, userID,
	).Scan(&membershipID)
	require.NoError(t, err)
	// Insert in an order deliberately unrelated to (and, if UUIDs happen to
	// sort that way, opposite to) role ID order, so a test relying on
	// insertion/physical row order rather than a real ORDER BY tiebreak
	// would be exposed by running this multiple times or on different
	// Postgres versions -- assert against the actual computed lowest ID
	// instead of assuming which of roleA/roleB sorts first.
	_, err = pool.Exec(ctx, `INSERT INTO roles (id, team_id, name) VALUES ($1, $2, 'Trainer'), ($3, $2, 'Co-Trainer')`,
		roleB, teamID, roleA)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2), ($1, $3)`,
		membershipID, roleB, roleA)
	require.NoError(t, err)

	from := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	_, err = repo.Create(ctx, teamID, userID, from, to, nil)
	require.NoError(t, err)

	var wantRoleID uuid.UUID
	if roleA.String() < roleB.String() {
		wantRoleID = roleA
	} else {
		wantRoleID = roleB
	}
	var wantRoleName string
	err = pool.QueryRow(ctx, `SELECT name FROM roles WHERE id = $1`, wantRoleID).Scan(&wantRoleName)
	require.NoError(t, err)

	// Repeat the query several times: a flaky/unspecified-order bug may not
	// reproduce on every single call, but should show up across a few.
	for i := 0; i < 5; i++ {
		byTeam, err := repo.ListByTeam(ctx, teamID, 50, nil)
		require.NoError(t, err)
		require.Len(t, byTeam, 1)
		require.NotNil(t, byTeam[0].RoleName)
		assert.Equal(t, wantRoleName, *byTeam[0].RoleName, "ListByTeam role badge must deterministically pick the lowest role id")

		byUser, err := repo.ListByUser(ctx, teamID, userID, 50, nil)
		require.NoError(t, err)
		require.Len(t, byUser, 1)
		require.NotNil(t, byUser[0].RoleName)
		assert.Equal(t, wantRoleName, *byUser[0].RoleName, "ListByUser role badge must deterministically pick the lowest role id")
	}
}

func TestAbsenceRepository_Update(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Update User', 'upd@example.com', '#aaaaaa')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Upd Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	from := "2025-03-01"
	to := "2025-03-07"
	ab, err := repo.Create(ctx, teamID, userID, from, to, nil)
	require.NoError(t, err)

	newTo := "2025-03-14"
	newReason := "extended vacation"
	updated, err := repo.Update(ctx, ab.Id, teamID, userID, nil, &newTo, &newReason)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, newTo, updated.ToDate.Format("2006-01-02"))
	assert.Equal(t, &newReason, updated.Reason)
}

// TestAbsenceRepository_Update_PartialPatch_RejectsExcessiveSpan regression-tests
// a gap where UpdateAbsence's maxAbsenceSpanDays check only ran when a PATCH
// supplied both from/to in the same request -- a PATCH supplying only `to`
// (or only `from`) skipped it entirely, since the resulting span isn't
// computable without an extra read. The absences_span_within_limit DB CHECK
// constraint (migration 00016) now catches this at the repository layer
// regardless of which fields a given PATCH happens to touch.
func TestAbsenceRepository_Update_PartialPatch_RejectsExcessiveSpan(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Span User', 'span@example.com', '#aaaaaa')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Span Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	from := "2025-03-01"
	to := "2025-03-07"
	ab, err := repo.Create(ctx, teamID, userID, from, to, nil)
	require.NoError(t, err)

	// Patching only `to` to a date thousands of days past `from` bypasses the
	// handler's in-memory span check (which requires both fields present).
	farFuture := "9999-12-31"
	_, err = repo.Update(ctx, ab.Id, teamID, userID, nil, &farFuture, nil)
	require.ErrorIs(t, err, absences.ErrSpanTooLong)
}

func TestAbsenceRepository_Update_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Cross Team User', 'crossteam@example.com', '#123456')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Owning Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Attacker Team')`,
		otherTid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	otherTeamID := uuid.MustParse(otherTid)
	userID := uuid.MustParse(uid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-03-01", "2025-03-07", nil)
	require.NoError(t, err)

	newReason := "attacker-supplied reason"
	_, err = repo.Update(ctx, ab.Id, otherTeamID, userID, nil, nil, &newReason)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestAbsenceRepository_Update_WrongUser_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	otherUid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Owner User', 'owner@example.com', '#111111')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Other Member', 'othermember@example.com', '#222222')`,
		otherUid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Shared Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	otherUserID := uuid.MustParse(otherUid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-03-01", "2025-03-07", nil)
	require.NoError(t, err)

	// Another member of the same team must not be able to update this absence.
	newReason := "teammate-supplied reason"
	_, err = repo.Update(ctx, ab.Id, teamID, otherUserID, nil, nil, &newReason)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestAbsenceRepository_Delete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del User', 'del@example.com', '#bbbbbb')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.NoError(t, err)

	err = repo.Delete(ctx, ab.Id, teamID, userID)
	require.NoError(t, err)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestAbsenceRepository_Delete_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Cross User', 'delcross@example.com', '#654321')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Owning Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Attacker Team')`,
		otherTid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	otherTeamID := uuid.MustParse(otherTid)
	userID := uuid.MustParse(uid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.NoError(t, err)

	err = repo.Delete(ctx, ab.Id, otherTeamID, userID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// Absence must still exist under the real team.
	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestAbsenceRepository_Delete_WrongUser_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	otherUid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Owner User', 'delowner@example.com', '#333333')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Other Member', 'delothermember@example.com', '#444444')`,
		otherUid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Shared Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`,
		tid, uid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	otherUserID := uuid.MustParse(otherUid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.NoError(t, err)

	// Another member of the same team must not be able to delete this absence.
	err = repo.Delete(ctx, ab.Id, teamID, otherUserID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

// TestAbsenceRepository_Create_NotMember_ReturnsErrNotMember regression-tests
// a gap where Create had no membership check at all, unlike
// events.SetAttendance/SetNomination's identical self-service writes --
// RequirePermission/RequireMembership only check membership once at the
// start of the request, so a membership removal racing a concurrent
// CreateAbsence call (e.g. an admin's RemoveMember) could otherwise still
// leave an orphaned absence row attached to a team the user no longer
// belongs to.
func TestAbsenceRepository_Create_NotMember_ReturnsErrNotMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Not A Member', 'notamember@example.com', '#999999')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Membership Team')`,
		tid)
	require.NoError(t, err)
	// Deliberately no memberships row for uid/tid.

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	_, err = repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.ErrorIs(t, err, absences.ErrNotMember)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, all, "no orphaned absence row must be created for a non-member")
}

// TestAbsenceRepository_Update_NotMember_ReturnsNoRows and
// TestAbsenceRepository_Delete_NotMember_ReturnsNoRows regression-test the
// same TOCTOU gap Create's ErrNotMember guard closes, applied to Update and
// Delete: without a membership re-check inside the write itself, a
// membership removal racing a concurrent self-service Update/Delete could
// still let a just-departed member mutate or delete an absence row tied to
// a team they no longer belong to.
func TestAbsenceRepository_Update_NotMember_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	teamID, userID := seedAbsenceUser(t, pool)
	ab, err := repo.Create(ctx, teamID, userID, "2025-06-01", "2025-06-05", nil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	require.NoError(t, err)

	newReason := "attacker-controlled edit"
	_, err = repo.Update(ctx, ab.Id, teamID, userID, nil, nil, &newReason)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// The row must be untouched by the rejected update.
	remaining, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Nil(t, remaining[0].Reason)
}

func TestAbsenceRepository_Delete_NotMember_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	teamID, userID := seedAbsenceUser(t, pool)
	ab, err := repo.Create(ctx, teamID, userID, "2025-06-01", "2025-06-05", nil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM memberships WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	require.NoError(t, err)

	err = repo.Delete(ctx, ab.Id, teamID, userID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	remaining, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, remaining, 1, "the row must not have been deleted by a non-member")
}

// seedAbsenceUser inserts a user + team + membership and returns their IDs,
// for the overlap regression tests below.
func seedAbsenceUser(t *testing.T, pool *pgxpool.Pool) (teamID, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	teamID, userID = uuid.New(), uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Overlap User', $2, '#333333')`,
		userID, fmt.Sprintf("overlap-%s@example.com", userID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Overlap Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, userID)
	require.NoError(t, err)
	return teamID, userID
}

// Regression test: nothing prevented a user from recording two overlapping
// absence entries for the same date range, whether via a genuine race
// between concurrent requests or simply two sequential creates -- the
// team's absence calendar and any downstream "days absent" aggregate would
// then double-count the overlapping days.
func TestAbsenceRepository_Create_OverlappingRange_Rejected(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()
	teamID, userID := seedAbsenceUser(t, pool)

	_, err := repo.Create(ctx, teamID, userID, "2025-06-01", "2025-06-10", nil)
	require.NoError(t, err)

	// Fully contained within the existing range.
	_, err = repo.Create(ctx, teamID, userID, "2025-06-03", "2025-06-05", nil)
	require.ErrorIs(t, err, absences.ErrOverlappingAbsence)

	// Partial overlap at the tail end.
	_, err = repo.Create(ctx, teamID, userID, "2025-06-08", "2025-06-15", nil)
	require.ErrorIs(t, err, absences.ErrOverlappingAbsence)

	all, err := repo.ListByUser(ctx, teamID, userID, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1, "no rejected overlapping entry should have been created")
}

// A different user's overlapping range is unaffected (overlap is scoped per
// user), and a range that starts the day after an existing one ends is not
// an overlap.
func TestAbsenceRepository_Create_NonOverlappingRanges_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()
	teamID, userID := seedAbsenceUser(t, pool)

	otherUserID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Other Overlap User', $2, '#444444')`,
		otherUserID, fmt.Sprintf("other-overlap-%s@example.com", otherUserID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamID, otherUserID)
	require.NoError(t, err)

	_, err = repo.Create(ctx, teamID, userID, "2025-06-01", "2025-06-10", nil)
	require.NoError(t, err)

	// Adjacent, non-overlapping (starts the day after the first ends).
	_, err = repo.Create(ctx, teamID, userID, "2025-06-11", "2025-06-15", nil)
	require.NoError(t, err)

	// Same dates, but a different user -- must not be blocked by userID's range.
	_, err = repo.Create(ctx, teamID, otherUserID, "2025-06-01", "2025-06-10", nil)
	require.NoError(t, err)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

// Regression test: Update must check the resulting range after merging a
// partial patch (only `to` supplied here) against the current row, not just
// whatever field the request happened to include -- same class of gap
// ErrSpanTooLong/ErrInvalidDateRange already guard for partial PATCHes.
func TestAbsenceRepository_Update_OverlappingRange_Rejected(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()
	teamID, userID := seedAbsenceUser(t, pool)

	_, err := repo.Create(ctx, teamID, userID, "2025-07-01", "2025-07-05", nil)
	require.NoError(t, err)
	second, err := repo.Create(ctx, teamID, userID, "2025-07-10", "2025-07-15", nil)
	require.NoError(t, err)

	// Extending the second entry's `to` alone now reaches back far enough
	// (only `from` stays as-is) to overlap the first -- the merged range,
	// not just the patched field, must be checked.
	newFrom := "2025-07-04"
	_, err = repo.Update(ctx, second.Id, teamID, userID, &newFrom, nil, nil)
	require.ErrorIs(t, err, absences.ErrOverlappingAbsence)

	// The second entry's dates must remain unchanged after the rejected update.
	mine, err := repo.ListByUser(ctx, teamID, userID, 50, nil)
	require.NoError(t, err)
	require.Len(t, mine, 2)
	for _, ab := range mine {
		if ab.Id == second.Id {
			assert.Equal(t, "2025-07-10", ab.FromDate.Format("2006-01-02"))
		}
	}

	// Updating it to legitimately not overlap (e.g. just its reason) still works.
	reason := "still fine"
	updated, err := repo.Update(ctx, second.Id, teamID, userID, nil, nil, &reason)
	require.NoError(t, err)
	assert.Equal(t, &reason, updated.Reason)
}
