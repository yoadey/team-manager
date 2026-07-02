package members_test

import (
	"context"
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

// seedMemberFixtures inserts a user and team into the DB, returning their IDs.
func seedMemberFixtures(t *testing.T, pool *pgxpool.Pool) (userID, teamID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	userID = uuid.New()
	teamID = uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Test Owner', 'owner@example.com', '#334455')`,
		userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Test Team')`, teamID)
	require.NoError(t, err)
	return userID, teamID
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

func TestMembersRepository_AddMember_NewUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)

	params := members.AddMemberParams{
		Name:  "Alice Smith",
		Email: "alice@example.com",
	}
	m, err := repo.AddMember(ctx, teamID.String(), params)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "Alice Smith", m.Name)
	assert.Equal(t, "alice@example.com", m.Email)
	assert.Empty(t, m.Roles)
}

func TestMembersRepository_AddMember_ExistingUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	existingUID, teamID := seedMemberFixtures(t, pool)

	// Add the existing user (owner) to a second team — AddMember should look up
	// the user by email and reuse the existing user row.
	teamID2 := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Team 2')`, teamID2)
	require.NoError(t, err)

	// Confirm owner email
	var ownerEmail string
	err = pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, existingUID).Scan(&ownerEmail)
	require.NoError(t, err)

	params := members.AddMemberParams{
		Name:  "Should Be Ignored",
		Email: ownerEmail,
	}
	m, err := repo.AddMember(ctx, teamID2.String(), params)
	require.NoError(t, err)
	require.NotNil(t, m)
	// The user row already exists — the returned membership should reference the original user.
	assert.Equal(t, existingUID, m.UserID)
	// Unused teamID to avoid compiler error
	_ = teamID
}

func TestMembersRepository_AddMember_WithRoles(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)
	roleID := seedRole(t, pool, teamID, "Player", `{"events":"read","members":"none","finances":"none","news":"none","polls":"read","settings":"none"}`)

	params := members.AddMemberParams{
		Name:    "Bob Jones",
		Email:   "bob@example.com",
		RoleIDs: []string{roleID.String()},
	}
	m, err := repo.AddMember(ctx, teamID.String(), params)
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Len(t, m.Roles, 1)
	assert.Equal(t, roleID, m.Roles[0].Id)
	assert.Equal(t, "Player", m.Roles[0].Name)
}

func TestMembersRepository_ListMembers(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)

	// Add three members with names that sort deterministically.
	names := []string{"Charlie", "Alice", "Bob"}
	for _, name := range names {
		_, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
			Name:  name,
			Email: name + "@example.com",
		})
		require.NoError(t, err)
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

	_, teamID := seedMemberFixtures(t, pool)

	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta"} {
		_, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
			Name:  name,
			Email: name + "@example.com",
		})
		require.NoError(t, err)
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

	_, teamID := seedMemberFixtures(t, pool)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Dana White",
		Email: "dana@example.com",
	})
	require.NoError(t, err)

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

func TestMembersRepository_UpdateMember_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Team')`, otherTeamID)
	require.NoError(t, err)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Cross Team Target",
		Email: "crossteam-member@example.com",
	})
	require.NoError(t, err)

	newName := "Attacker Renamed"
	_, err = repo.UpdateMember(ctx, m.MembershipID.String(), otherTeamID.String(), members.MemberPatch{
		Name: &newName,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestMembersRepository_UpdateMember_PartialPatch(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Eve Original",
		Email: "eve@example.com",
	})
	require.NoError(t, err)

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

	_, teamID := seedMemberFixtures(t, pool)
	roleA := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)
	roleB := seedRole(t, pool, teamID, "Member", `{"events":"read","members":"none","finances":"none","news":"read","polls":"read","settings":"none"}`)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Frank Castle",
		Email: "frank@example.com",
	})
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

	_, teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'SetRoles Other Team')`, otherTeamID)
	require.NoError(t, err)
	roleA := seedRole(t, pool, teamID, "Admin", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Membership Owner",
		Email: "membership-owner@example.com",
	})
	require.NoError(t, err)

	_, err = repo.SetRoles(ctx, m.MembershipID.String(), otherTeamID.String(), []string{roleA.String()})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestMembersRepository_SetRoles_RoleFromOtherTeam_ReturnsErrRoleNotInTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Foreign Role Team')`, otherTeamID)
	require.NoError(t, err)
	foreignRole := seedRole(t, pool, otherTeamID, "Foreign Role", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Legit Member",
		Email: "legit-member@example.com",
	})
	require.NoError(t, err)

	_, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{foreignRole.String()})
	require.ErrorIs(t, err, members.ErrRoleNotInTeam)
}

func TestMembersRepository_AddMember_RoleFromOtherTeam_ReturnsErrRoleNotInTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Foreign Role Team 2')`, otherTeamID)
	require.NoError(t, err)
	foreignRole := seedRole(t, pool, otherTeamID, "Foreign Role 2", `{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	_, err = repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:    "New Member",
		Email:   "new-member-foreign-role@example.com",
		RoleIDs: []string{foreignRole.String()},
	})
	require.ErrorIs(t, err, members.ErrRoleNotInTeam)

	// No membership or user row should have leaked in from the rejected call.
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email = 'new-member-foreign-role@example.com'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "no user row should be created when role validation fails")
}

func TestMembersRepository_RemoveMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Grace Hopper",
		Email: "grace@example.com",
	})
	require.NoError(t, err)

	err = repo.RemoveMember(ctx, m.MembershipID.String(), teamID.String())
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

	_, teamID := seedMemberFixtures(t, pool)
	otherTeamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Remove Other Team')`, otherTeamID)
	require.NoError(t, err)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Should Survive",
		Email: "should-survive@example.com",
	})
	require.NoError(t, err)

	err = repo.RemoveMember(ctx, m.MembershipID.String(), otherTeamID.String())
	require.ErrorIs(t, err, pgx.ErrNoRows)

	list, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestMembersRepository_IsMember(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	_, teamID := seedMemberFixtures(t, pool)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Henry Ford",
		Email: "henry@example.com",
	})
	require.NoError(t, err)

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

	_, teamID := seedMemberFixtures(t, pool)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Iris West",
		Email: "iris@example.com",
	})
	require.NoError(t, err)

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

	_, teamID := seedMemberFixtures(t, pool)
	// roleA grants events=read, members=none.
	roleA := seedRole(t, pool, teamID, "Viewer",
		`{"events":"read","members":"none","finances":"none","news":"none","polls":"none","settings":"none"}`)
	// roleB grants events=write (overrides read), members=read.
	roleB := seedRole(t, pool, teamID, "Editor",
		`{"events":"write","members":"read","finances":"none","news":"none","polls":"none","settings":"none"}`)

	m, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:    "Jack London",
		Email:   "jack@example.com",
		RoleIDs: []string{roleA.String(), roleB.String()},
	})
	require.NoError(t, err)

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

	_, teamID := seedMemberFixtures(t, pool)
	roleID := seedRole(t, pool, teamID, "Coach",
		`{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)

	_, err := repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:    "Karen Page",
		Email:   "karen@example.com",
		RoleIDs: []string{roleID.String()},
	})
	require.NoError(t, err)
	_, err = repo.AddMember(ctx, teamID.String(), members.AddMemberParams{
		Name:  "Luke Cage",
		Email: "luke@example.com",
	})
	require.NoError(t, err)

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
