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

// seedAdminCaller seeds a member holding full write on every module and
// returns their userID, for use as SetRoles' callerUserID in tests that
// aren't specifically exercising enforceNoPermissionEscalation -- mirrors
// how teams.Repository.CreateTeam grants the team creator this same "Admin"
// role directly (bypassing SetRoles) as the real bootstrap path.
func seedAdminCaller(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID) uuid.UUID {
	t.Helper()
	adminRole := seedRole(t, pool, teamID, "Full Admin Caller",
		`{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}`)
	m := seedMember(t, pool, teamID, "Full Admin Caller", fmt.Sprintf("admin-caller-%s@example.com", uuid.New()), adminRole)
	return m.UserID
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
	caller := seedAdminCaller(t, pool, teamID)

	m := seedMember(t, pool, teamID, "Frank Castle", "frank@example.com")

	// A second Admin so demoting/clearing m's roles below never trips the
	// last-settings-admin guard — this test is about role-replacement
	// mechanics, not the admin-guard (covered separately).
	other := seedMember(t, pool, teamID, "Other Admin", "other-admin@example.com")
	_, err := repo.SetRoles(ctx, other.MembershipID.String(), teamID.String(), []string{roleA.String()}, caller.String())
	require.NoError(t, err)

	// Assign roleA.
	updated, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleA.String()}, caller.String())
	require.NoError(t, err)
	require.Len(t, updated.Roles, 1)
	assert.Equal(t, roleA, updated.Roles[0].Id)

	// Replace with roleB.
	updated, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleB.String()}, caller.String())
	require.NoError(t, err)
	require.Len(t, updated.Roles, 1)
	assert.Equal(t, roleB, updated.Roles[0].Id)

	// Assign both roles.
	updated, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleA.String(), roleB.String()}, caller.String())
	require.NoError(t, err)
	assert.Len(t, updated.Roles, 2)

	// Clear all roles.
	updated, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{}, caller.String())
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
	caller := seedAdminCaller(t, pool, teamID)

	m := seedMember(t, pool, teamID, "Membership Owner", "membership-owner@example.com")

	_, err = repo.SetRoles(ctx, m.MembershipID.String(), otherTeamID.String(), []string{roleA.String()}, caller.String())
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
	caller := seedAdminCaller(t, pool, teamID)

	m := seedMember(t, pool, teamID, "Legit Member", "legit-member@example.com")

	_, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{foreignRole.String()}, caller.String())
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
	caller := seedAdminCaller(t, pool, teamID)

	// A settings:write admin so assigning Grace a non-admin role below never
	// trips the last-settings-admin guard — this test is about duplicate role
	// ID handling, not the admin-guard (covered separately).
	other := seedMember(t, pool, teamID, "Other Admin", "other-admin-dup-test@example.com")
	_, err := repo.SetRoles(ctx, other.MembershipID.String(), teamID.String(), []string{adminRole.String()}, caller.String())
	require.NoError(t, err)

	m := seedMember(t, pool, teamID, "Grace Hopper", "grace@example.com")

	updated, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{roleA.String(), roleA.String()}, caller.String())
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

	// adminRole assigned directly via seedMember (not repo.SetRoles) so this
	// stays the team's ONLY settings:write holder -- a caller seeded via
	// seedAdminCaller would itself be a second one, defeating the "sole
	// admin" premise this test exists to check.
	m := seedMember(t, pool, teamID, "Sole Admin", "sole-admin@example.com", adminRole)

	err := repo.RemoveMember(ctx, m.MembershipID.String(), teamID.String())
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

	m1 := seedMember(t, pool, teamID, "Admin One", "admin1@example.com", adminRole)
	seedMember(t, pool, teamID, "Admin Two", "admin2@example.com", adminRole)

	// Removing m1 is fine — Admin Two still holds settings:write.
	err := repo.RemoveMember(ctx, m1.MembershipID.String(), teamID.String())
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

	// adminRole assigned directly via seedMember so m stays the team's ONLY
	// settings:write holder. m then acts as its own caller below (the sole
	// admin attempting to demote themselves) -- their own effective
	// permissions already cover both target role sets, so the calls below
	// reach (and are correctly rejected by) the last-settings-admin guard
	// rather than the unrelated escalation check.
	m := seedMember(t, pool, teamID, "Sole Admin", "sole-admin-2@example.com", adminRole)

	// Demoting the sole admin to a non-settings role must be blocked.
	_, err := repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{memberRole.String()}, m.UserID.String())
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)

	// Clearing all roles from the sole admin must also be blocked.
	_, err = repo.SetRoles(ctx, m.MembershipID.String(), teamID.String(), []string{}, m.UserID.String())
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)
}

// Regression test for a privilege-escalation path: middleware gates both role
// definition (POST/PATCH .../roles) and role assignment (PUT
// .../members/{id}/roles) on nothing more than settings:write. Without a
// caller-side check, a member holding only settings:write could create a
// role granting arbitrary module permissions and assign it to themselves,
// ending up with de facto full admin despite never having been granted
// anything beyond settings:write. SetRoles must refuse to let a caller grant
// a module permission level higher than their own effective permission.
func TestMembersRepository_SetRoles_InsufficientPermissionToGrant_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	// A settings-only role -- deliberately no other module write, so the
	// caller below has authority to manage role assignments but nothing
	// else.
	settingsOnlyRole := seedRole(t, pool, teamID, "Settings Only",
		`{"events":"none","members":"none","finances":"none","news":"none","polls":"none","settings":"write"}`)
	// A second, pre-existing settings:write holder so none of the calls
	// below can be confused with the (unrelated) last-settings-admin guard.
	seedAdminCaller(t, pool, teamID)

	financeAdminRole := seedRole(t, pool, teamID, "Finance Admin",
		`{"events":"none","members":"none","finances":"write","news":"none","polls":"none","settings":"none"}`)

	attacker := seedMember(t, pool, teamID, "Attacker", "attacker@example.com", settingsOnlyRole)

	// The attacker tries to grant themselves finances:write, which they do
	// not themselves hold.
	_, err := repo.SetRoles(ctx, attacker.MembershipID.String(), teamID.String(), []string{financeAdminRole.String()}, attacker.UserID.String())
	require.ErrorIs(t, err, members.ErrInsufficientPermissionToGrant)

	// Same result granting it to a DIFFERENT membership (e.g. a colluding
	// second account) rather than themselves -- the check isn't merely a
	// self-assignment guard.
	victim := seedMember(t, pool, teamID, "Second Account", "second-account@example.com")
	_, err = repo.SetRoles(ctx, victim.MembershipID.String(), teamID.String(), []string{financeAdminRole.String()}, attacker.UserID.String())
	require.ErrorIs(t, err, members.ErrInsufficientPermissionToGrant)

	// The attacker's own role assignment must be untouched by the rejected
	// attempt.
	list, err := repo.ListMembers(ctx, teamID.String(), 10, nil)
	require.NoError(t, err)
	for _, mr := range list {
		if mr.MembershipID == attacker.MembershipID {
			require.Len(t, mr.Roles, 1)
			assert.Equal(t, settingsOnlyRole, mr.Roles[0].Id)
		}
	}
}

// Companion test: SetRoles fully replaces a membership's role set, so a
// caller reorganizing/demoting an EXISTING permission holder's roles must
// stay allowed even if the result still exceeds the caller's own permission
// ceiling -- only an actual INCREASE beyond what the target already had
// counts as a grant. Otherwise a settings:write-only caller could never
// touch the role assignment of a member who legitimately holds e.g.
// finances:write via someone else's earlier grant, even just to demote or
// reorganize them.
func TestMembersRepository_SetRoles_ReorganizingExistingHigherPermission_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := members.NewRepository(pool)
	ctx := context.Background()

	teamID := seedMemberFixtures(t, pool)
	settingsOnlyRole := seedRole(t, pool, teamID, "Settings Only",
		`{"events":"none","members":"none","finances":"none","news":"none","polls":"none","settings":"write"}`)
	financeAdminRole := seedRole(t, pool, teamID, "Finance Admin",
		`{"events":"none","members":"none","finances":"write","news":"none","polls":"none","settings":"none"}`)
	financeReadRole := seedRole(t, pool, teamID, "Finance Reader",
		`{"events":"none","members":"none","finances":"read","news":"none","polls":"none","settings":"none"}`)
	seedAdminCaller(t, pool, teamID)

	caller := seedMember(t, pool, teamID, "Settings Admin", "settings-admin@example.com", settingsOnlyRole)
	// treasurer already holds finances:write, granted by someone else
	// (direct SQL, standing in for a prior legitimate grant) -- caller
	// themselves never held finances:write.
	treasurer := seedMember(t, pool, teamID, "Treasurer", "treasurer@example.com", financeAdminRole)

	// Demoting the treasurer from finances:write to finances:read is a
	// REDUCTION relative to what they already had, so it must be allowed
	// even though the caller's own finances permission is "none".
	updated, err := repo.SetRoles(ctx, treasurer.MembershipID.String(), teamID.String(), []string{financeReadRole.String()}, caller.UserID.String())
	require.NoError(t, err)
	require.Len(t, updated.Roles, 1)
	assert.Equal(t, financeReadRole, updated.Roles[0].Id)
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
