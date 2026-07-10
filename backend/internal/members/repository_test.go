package members_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// seedMemberFixtures inserts an owner user and a team into the DB, returning
// the team's ID (the owner row exists only to satisfy tables that reference a
// user elsewhere in the fixture; no test needs its ID back).
func seedMemberFixtures(t *testing.T, pool *pgxpool.Pool) (teamID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	userID := uuid.New()
	teamID = uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Test Owner', 'owner@example.com', '#334455')`,
		userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Test Team')`, teamID)
	require.NoError(t, err)
	return teamID
}

// seedRole inserts a role for the given team and returns its ID.
func seedRole(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, name, perms string) uuid.UUID {
	t.Helper()
	var roleID uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, $2, $3) RETURNING id`,
		teamID, name, perms,
	).Scan(&roleID)
	require.NoError(t, err)
	return roleID
}

// seedMember inserts a user and a membership (optionally with roles) directly
// via SQL. Test-only fixture setup, standing in for the removed AddMember
// repository method (deleted along with the unreachable direct-add-member API
// once invite-link redemption became the supported way to join a team).
func seedMember(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, name, email string, roleIDs ...uuid.UUID) *members.MemberRow {
	t.Helper()
	ctx := context.Background()

	var userID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color) VALUES ($1, $2, '#6366f1') RETURNING id
	`, name, email).Scan(&userID)
	require.NoError(t, err)

	var membershipID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id
	`, teamID, userID).Scan(&membershipID)
	require.NoError(t, err)

	for _, roleID := range roleIDs {
		_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, membershipID, roleID)
		require.NoError(t, err)
	}

	return &members.MemberRow{MembershipID: membershipID, UserID: userID, Name: name, Email: email}
}

func TestMembersRepository_ListMembers(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	// Add three members with names that sort deterministically.
	names := []string{"Charlie", "Alice", "Bob"}
	for _, name := range names {
		seedMember(t, pool, teamID, name, name+"@example.com")
	}

	page, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	require.Len(t, page, 3)
	// Should be returned in alphabetical order.
	assert.Equal(t, "Alice", page[0].Name)
	assert.Equal(t, "Bob", page[1].Name)
	assert.Equal(t, "Charlie", page[2].Name)
}

func TestMembersRepository_ListMembers_Pagination(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta"} {
		seedMember(t, pool, teamID, name, name+"@example.com")
	}

	// First page of 2.
	page1, err := repo.ListMembers(ctx, teamID.String(), 2, nil)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, "Alpha", page1[0].Name)
	assert.Equal(t, "Beta", page1[1].Name)

	// Build cursor from last item on page 1.
	cur := &members.ListCursor{
		Name: page1[1].Name,
		ID:   page1[1].MembershipID,
	}
	page2, err := repo.ListMembers(ctx, teamID.String(), 2, cur)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	assert.Equal(t, "Delta", page2[0].Name)
	assert.Equal(t, "Gamma", page2[1].Name)
}

func TestMembersRepository_UpdateMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	m := seedMember(t, pool, teamID, "Dana White", "dana@example.com")

	newName := "Dana Updated"
	newEmail := "dana2@example.com"
	newPhone := "+1234567890"
	newAddr := "123 Main St"
	bday := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)
	grp := "seniors"

	updated, err := repo.UpdateMember(ctx, m.MembershipID.String(), teamID.String(), members.MemberPatch{
		Name:     &newName,
		Email:    &newEmail,
		Phone:    &newPhone,
		Address:  &newAddr,
		Birthday: &bday,
		Group:    &grp,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Dana Updated", updated.Name)
	assert.Equal(t, "dana2@example.com", updated.Email)
	assert.Equal(t, &newPhone, updated.Phone)
	assert.Equal(t, &newAddr, updated.Address)
	require.NotNil(t, updated.Birthday)
	assert.Equal(t, bday.Format("2006-01-02"), updated.Birthday.Format("2006-01-02"))
	assert.Equal(t, &grp, updated.Group)
}

// Regression test: UpdateMember must map a users.email UNIQUE violation to a
// clean ErrEmailTaken instead of letting a raw wrapped Postgres error fall
// through when a member's email is changed to one already used by a
// different user account.
func TestMembersRepository_UpdateMember_EmailTaken_ReturnsErrEmailTaken(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	seedMember(t, pool, teamID, "Existing User", "existing-email-taken@example.com")
	m := seedMember(t, pool, teamID, "Other Member", "other-email-taken@example.com")

	collidingEmail := "existing-email-taken@example.com"
	_, err := repo.UpdateMember(ctx, m.MembershipID.String(), teamID.String(), members.MemberPatch{
		Email: &collidingEmail,
	})
	require.ErrorIs(t, err, members.ErrEmailTaken)
}

func TestMembersRepository_UpdateMember_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Team')`, otherTeamID)
	require.NoError(t, err)

	m := seedMember(t, pool, teamID, "Cross Team Target", "crossteam-member@example.com")

	newName := "Attacker Renamed"
	_, err = repo.UpdateMember(ctx, m.MembershipID.String(), otherTeamID.String(), members.MemberPatch{
		Name: &newName,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

// Regression test: UpdateMember used to resolve the membership's userID via
// a bare r.pool.QueryRow BEFORE starting its transaction, then unconditionally
// UPDATE users on that already-resolved userID inside the tx -- never
// re-checking that the membership (and hence the caller's authority) still
// existed. A concurrent RemoveMember could delete the membership in the
// window between that outside-tx check and the UPDATE users statement: the
// write would still commit (permanently changing the departed user's global
// account fields) while UpdateMember's own final reload returned
// pgx.ErrNoRows, reporting what looked like a clean no-op failure to the
// caller. The fix moves the check inside the tx, behind the same per-team
// advisory lock RemoveMember already takes, so the two fully serialize.
//
// This fires UpdateMember and RemoveMember concurrently against a freshly
// seeded membership across many rounds and asserts the invariant the bug
// violated: whenever UpdateMember reports pgx.ErrNoRows, the user's name in
// the database must still be the pre-update value -- i.e. a reported
// failure must never carry a silent side effect.
func TestMembersRepository_UpdateMember_ConcurrentRemoveMember_NoSilentSideEffect(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	const rounds = 20
	for i := 0; i < rounds; i++ {
		originalName := fmt.Sprintf("Original %d", i)
		newName := fmt.Sprintf("Updated %d", i)
		m := seedMember(t, pool, teamID, originalName, fmt.Sprintf("racer%d@example.com", i))

		var wg sync.WaitGroup
		var updateErr, removeErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, updateErr = repo.UpdateMember(ctx, m.MembershipID.String(), teamID.String(), members.MemberPatch{Name: &newName})
		}()
		go func() {
			defer wg.Done()
			removeErr = repo.RemoveMember(ctx, m.MembershipID.String(), teamID.String())
		}()
		wg.Wait()

		require.True(t, removeErr == nil || errors.Is(removeErr, pgx.ErrNoRows),
			"round %d: RemoveMember returned an unexpected error: %v", i, removeErr)

		if errors.Is(updateErr, pgx.ErrNoRows) {
			var gotName string
			require.NoError(t, pool.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, m.UserID).Scan(&gotName))
			assert.Equal(t, originalName, gotName,
				"round %d: UpdateMember reported ErrNoRows but silently changed the user's name anyway", i)
		} else {
			require.NoError(t, updateErr, "round %d: UpdateMember failed with an unexpected error", i)
		}
	}
}

func TestMembersRepository_UpdateMember_PartialPatch(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	m := seedMember(t, pool, teamID, "Eve Original", "eve@example.com")

	newName := "Eve Renamed"
	updated, err := repo.UpdateMember(ctx, m.MembershipID.String(), teamID.String(), members.MemberPatch{
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "Eve Renamed", updated.Name)
	// Email must remain unchanged.
	assert.Equal(t, "eve@example.com", updated.Email)
}

func TestMembersRepository_SetRoles(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	roleA := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)
	roleB := seedRole(t, pool, teamID, "Member", `{"events":"read","members":"none","finances":"none","news":"read","polls":"read","settings":"none"}`)

	m := seedMember(t, pool, teamID, "Frank Castle", "frank@example.com")

	// A second Admin so demoting/clearing m's roles below never trips the
	// last-settings-admin guard — this test is about role-replacement
	// mechanics, not the admin-guard (covered separately).
	other := seedMember(t, pool, teamID, "Other Admin", "other-admin@example.com")
	_, err := repo.SetRoles(ctx, other.MembershipID.String(), teamID.String(), []string{roleA.String()})
	require.NoError(t, err)

	// Assign roleA.
	updated, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleA.String()})
	require.NoError(t, err)
	require.Len(t, updated.Roles, 1)
	assert.Equal(t, roleA, updated.Roles[0].Id)

	// Replace with roleB.
	updated, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleB.String()})
	require.NoError(t, err)
	require.Len(t, updated.Roles, 1)
	assert.Equal(t, roleB, updated.Roles[0].Id)

	// Assign both roles.
	updated, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleA.String(), roleB.String()})
	require.NoError(t, err)
	assert.Len(t, updated.Roles, 2)

	// Clear all roles.
	updated, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{})
	require.NoError(t, err)
	assert.Empty(t, updated.Roles)
}

func TestMembersRepository_SetRoles_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'SetRoles Other Team')`, otherTeamID)
	require.NoError(t, err)
	roleA := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	m := seedMember(t, pool, teamID, "Membership Owner", "membership-owner@example.com")

	_, err = repo.SetRoles(ctx, m.MembershipID.String(), otherTeamID.String(), []string{roleA.String()})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestMembersRepository_SetRoles_RoleFromOtherTeam_ReturnsErrRoleNotInTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Foreign Role Team')`, otherTeamID)
	require.NoError(t, err)
	foreignRole := seedRole(t, pool, otherTeamID, "Foreign Role", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	m := seedMember(t, pool, teamID, "Legit Member", "legit-member@example.com")

	_, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{foreignRole.String()})
	require.ErrorIs(t, err, members.ErrRoleNotInTeam)
}

// Regression test: the role-existence check (COUNT(*) FROM roles WHERE id =
// ANY($1), compared against len(roleIDs)) used to wrongly reject a caller
// that legitimately repeats the same valid role ID, and even after fixing
// that comparison, the membership_roles INSERT loop would still fail on the
// composite primary key (membership_id, role_id) for the duplicate pair.
// SetRoles must dedupe up front so both problems are avoided.
func TestMembersRepository_SetRoles_DuplicateValidRoleID_Succeeds(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	roleA := seedRole(t, pool, teamID, "Player", `{"events":"read","members":"none","finances":"none","news":"none","polls":"read","settings":"none"}`)
	adminRole := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	// A settings:write admin so assigning Grace a non-admin role below never
	// trips the last-settings-admin guard — this test is about duplicate role
	// ID handling, not the admin-guard (covered separately).
	other := seedMember(t, pool, teamID, "Other Admin", "other-admin-dup-test@example.com")
	_, err := repo.SetRoles(ctx, other.MembershipID.String(), teamID.String(), []string{adminRole.String()})
	require.NoError(t, err)

	m := seedMember(t, pool, teamID, "Grace Hopper", "grace@example.com")

	updated, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleA.String(), roleA.String()})
	require.NoError(t, err)
	require.Len(t, updated.Roles, 1)
	assert.Equal(t, roleA, updated.Roles[0].Id)
}

func TestMembersRepository_RemoveMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	m := seedMember(t, pool, teamID, "Grace Hopper", "grace@example.com")

	err := repo.RemoveMember(ctx, m.MembershipID.String(), teamID.String())
	require.NoError(t, err)

	list, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestMembersRepository_RemoveMember_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Remove Other Team')`, otherTeamID)
	require.NoError(t, err)

	m := seedMember(t, pool, teamID, "Should Survive", "should-survive@example.com")

	err = repo.RemoveMember(ctx, m.MembershipID.String(), otherTeamID.String())
	require.ErrorIs(t, err, pgx.ErrNoRows)

	list, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestMembersRepository_RemoveMember_LastSettingsAdmin_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	adminRole := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	m := seedMember(t, pool, teamID, "Sole Admin", "sole-admin@example.com")
	_, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{adminRole.String()})
	require.NoError(t, err)

	err = repo.RemoveMember(ctx, m.MembershipID.String(), teamID.String())
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)

	// Still present.
	list, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestMembersRepository_RemoveMember_NotLastSettingsAdmin_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	adminRole := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	m1 := seedMember(t, pool, teamID, "Admin One", "admin1@example.com")
	_, err := repo.SetRoles(ctx, m1.MembershipID.String(), teamID.String(), []string{adminRole.String()})
	require.NoError(t, err)

	m2 := seedMember(t, pool, teamID, "Admin Two", "admin2@example.com")
	_, err = repo.SetRoles(ctx, m2.MembershipID.String(), teamID.String(), []string{adminRole.String()})
	require.NoError(t, err)

	// Removing m1 is fine — m2 still holds settings:write.
	err = repo.RemoveMember(ctx, m1.MembershipID.String(), teamID.String())
	require.NoError(t, err)
}

func TestMembersRepository_SetRoles_LastSettingsAdmin_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	adminRole := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)
	memberRole := seedRole(t, pool, teamID, "Member", `{"events":"read","members":"none","finances":"none","news":"read","polls":"read","settings":"none"}`)

	m := seedMember(t, pool, teamID, "Sole Admin", "sole-admin-2@example.com")
	_, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{adminRole.String()})
	require.NoError(t, err)

	// Demoting the sole admin to a non-settings role must be blocked.
	_, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{memberRole.String()})
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)

	// Clearing all roles from the sole admin must also be blocked.
	_, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{})
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)
}

func TestMembersRepository_IsMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	m := seedMember(t, pool, teamID, "Henry Ford", "henry@example.com")

	isMember, err := repo.IsMember(ctx, teamID, m.UserID)
	require.NoError(t, err)
	assert.True(t, isMember)

	strangerID := uuid.New()
	isStranger, err := repo.IsMember(ctx, teamID, strangerID)
	require.NoError(t, err)
	assert.False(t, isStranger)
}

func TestMembersRepository_GetPermissions_NoRoles(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)

	m := seedMember(t, pool, teamID, "Iris West", "iris@example.com")

	perms, err := repo.GetPermissions(ctx, teamID, m.UserID)
	require.NoError(t, err)
	assert.Equal(t, "none", perms.Events)
	assert.Equal(t, "none", perms.Members)
	assert.Equal(t, "none", perms.Finances)
	assert.Equal(t, "none", perms.News)
	assert.Equal(t, "none", perms.Polls)
	assert.Equal(t, "none", perms.Settings)
}

func TestMembersRepository_GetPermissions_MaxFold(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	// roleA grants events=read, members=none.
	roleA := seedRole(t, pool, teamID, "Viewer",
		`{"events":"read","members":"none","finances":"none","news":"none","polls":"none","settings":"none"}`)
	// roleB grants events=write (overrides read), members=read.
	roleB := seedRole(t, pool, teamID, "Editor",
		`{"events":"write","members":"read","finances":"none","news":"none","polls":"none","settings":"none"}`)

	m := seedMember(t, pool, teamID, "Jack London", "jack@example.com", roleA, roleB)

	perms, err := repo.GetPermissions(ctx, teamID, m.UserID)
	require.NoError(t, err)
	// Effective permissions are the maximum across both roles.
	assert.Equal(t, "write", perms.Events)
	assert.Equal(t, "read", perms.Members)
	assert.Equal(t, "none", perms.Finances)
}

func TestMembersRepository_ListMembers_BatchRolesLoaded(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	roleID := seedRole(t, pool, teamID, "Coach",
		`{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	seedMember(t, pool, teamID, "Karen Page", "karen@example.com", roleID)
	seedMember(t, pool, teamID, "Luke Cage", "luke@example.com")

	list, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Karen (alphabetically first) should have the Coach role.
	assert.Equal(t, "Karen Page", list[0].Name)
	require.Len(t, list[0].Roles, 1)
	assert.Equal(t, "Coach", list[0].Roles[0].Name)

	// Luke has no roles.
	assert.Equal(t, "Luke Cage", list[1].Name)
	assert.Empty(t, list[1].Roles)
}
