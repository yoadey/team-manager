package teams_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// insertUser is a small helper shared by the AcceptInvite tests below to cut
// down on repeated boilerplate for tests that need several distinct users.
func insertUser(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var userID string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Invite Test User', $1, '#445566')
		RETURNING id
	`, email).Scan(&userID)
	require.NoError(t, err)
	return userID
}

func TestTeamRepository_CreateTeam(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Test User', 'create@example.com', '#aabbcc')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "My Team", userID, nil, nil, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tr.Id.String())
	assert.Equal(t, "My Team", tr.Name)
	assert.False(t, tr.CreatedAt.IsZero())
}

// Regression test: CreateTeam used to insert only the name column, silently
// discarding icon/iconBg/iconFg even though the frontend's create-team form
// (and the OpenAPI CreateTeamRequest schema) always sends them -- a team
// created against the real backend ended up with NULL icon colors
// regardless of what the user picked, with no error to surface it.
func TestTeamRepository_CreateTeam_PersistsIconFields(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := insertUser(t, pool, "create-icon@example.com")
	repo := teams.NewRepository(pool)

	icon := "🏆"
	iconBg := "#1A1A1A"
	iconFg := "#F5C518"
	tr, err := repo.CreateTeam(ctx, "Icon Team", userID, &icon, &iconBg, &iconFg)
	require.NoError(t, err)
	require.NotNil(t, tr.Icon)
	require.NotNil(t, tr.IconBg)
	require.NotNil(t, tr.IconFg)
	assert.Equal(t, icon, *tr.Icon)
	assert.Equal(t, iconBg, *tr.IconBg)
	assert.Equal(t, iconFg, *tr.IconFg)
}

// Regression test: the seeded default "Member" role (auto-assigned to every
// user who joins via AcceptInvite) used to set members/settings to "none".
// RequirePermission gates GET requests too -- a module set to "none" hides
// reads entirely, not just writes -- and AppContext.afterLoginLoad
// unconditionally fetches both the member roster (members:read) and the
// role catalog (settings:read) for every team member on every login/team
// switch, not just for admins. That combination 403'd both calls for every
// ordinary member, and since afterLoginLoad awaits all five loads via
// Promise.all, the whole dashboard load failed for anyone without the
// Admin role -- i.e. everyone except the team's creator.
func TestTeamRepository_CreateTeam_DefaultMemberRoleCanReadRosterAndRoleCatalog(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Default Role User', 'defaultrole@example.com', '#334455')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Default Role Team", userID, nil, nil, nil)
	require.NoError(t, err)

	var permsJSON []byte
	err = pool.QueryRow(ctx, `
		SELECT permissions FROM roles WHERE team_id = $1 AND system = true AND name = 'Member'
	`, tr.Id).Scan(&permsJSON)
	require.NoError(t, err)

	var perms teams.PermissionsJSON
	require.NoError(t, json.Unmarshal(permsJSON, &perms))
	assert.Equal(t, "read", perms.Members, "ordinary members must be able to see the member roster")
	assert.Equal(t, "read", perms.Settings, "ordinary members must be able to see the role catalog")
	assert.Equal(t, "read", perms.Events)
	assert.Equal(t, "read", perms.News)
	assert.Equal(t, "read", perms.Polls)
	assert.Equal(t, "none", perms.Finances, "financial data stays admin-only by default")
}

func TestTeamRepository_ListForUser(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('List User', 'list@example.com', '#ccddee')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	_, err = repo.CreateTeam(ctx, "List Team", userID, nil, nil, nil)
	require.NoError(t, err)

	result, err := repo.ListTeamsForUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "List Team", result[0].Name)
}

// Regression test for the ListForUser N+1 fix: GetMemberCounts,
// GetMembershipsForUser, and GetRolesForMemberships must return correct,
// per-team results in one batched call each, across teams from different
// creators (so member counts genuinely differ) and with a second member
// added to one team.
func TestTeamRepository_BatchedListEnrichment_ReturnsPerTeamResults(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := insertUser(t, pool, "batch-owner@example.com")
	otherUserID := insertUser(t, pool, "batch-other@example.com")

	repo := teams.NewRepository(pool)
	teamA, err := repo.CreateTeam(ctx, "Batch Team A", userID, nil, nil, nil)
	require.NoError(t, err)
	teamB, err := repo.CreateTeam(ctx, "Batch Team B", userID, nil, nil, nil)
	require.NoError(t, err)

	// Add a second member to team A only, so its member count (2) differs
	// from team B's (1).
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, teamA.Id, otherUserID)
	require.NoError(t, err)

	teamIDs := []string{teamA.Id.String(), teamB.Id.String()}

	counts, err := repo.GetMemberCounts(ctx, teamIDs)
	require.NoError(t, err)
	assert.Equal(t, 2, counts[teamA.Id.String()])
	assert.Equal(t, 1, counts[teamB.Id.String()])

	memberships, err := repo.GetMembershipsForUser(ctx, teamIDs, userID)
	require.NoError(t, err)
	require.Contains(t, memberships, teamA.Id.String())
	require.Contains(t, memberships, teamB.Id.String())
	assert.NotContains(t, memberships, "unrelated-key")
	membershipA := memberships[teamA.Id.String()]
	membershipB := memberships[teamB.Id.String()]
	assert.Equal(t, teamA.Id, membershipA.TeamID)
	assert.Equal(t, teamB.Id, membershipB.TeamID)

	rolesByMembership, err := repo.GetRolesForMemberships(ctx, []string{membershipA.Id.String(), membershipB.Id.String()})
	require.NoError(t, err)
	require.Len(t, rolesByMembership[membershipA.Id.String()], 1)
	require.Len(t, rolesByMembership[membershipB.Id.String()], 1)
	assert.Equal(t, "Admin", rolesByMembership[membershipA.Id.String()][0].Name)
	assert.Equal(t, "Admin", rolesByMembership[membershipB.Id.String()][0].Name)
}

// TestTeamRepository_GetRolesForMemberships_ExcludesCrossTeamRole is a
// defense-in-depth regression test: GetRolesForMemberships joined roles to
// membership_roles with no r.team_id check, unlike the established pattern
// elsewhere (members.getMembershipEffectivePermissionsQ,
// teams.Repository.GetRolesForMembership). Every current INSERT INTO
// membership_roles call site always inserts a role already validated as
// belonging to the target team, so this can't happen through normal API use
// today -- but this feeds ListForUser's per-team displayed Perms, so a
// future change that broke that insert-side invariant would silently merge
// a different team's role permissions into what the caller sees for this
// team instead of failing safe.
func TestTeamRepository_GetRolesForMemberships_ExcludesCrossTeamRole(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := insertUser(t, pool, "crossrole-owner@example.com")
	repo := teams.NewRepository(pool)
	teamA, err := repo.CreateTeam(ctx, "Cross Role Team A", userID, nil, nil, nil)
	require.NoError(t, err)
	teamB, err := repo.CreateTeam(ctx, "Cross Role Team B", userID, nil, nil, nil)
	require.NoError(t, err)

	var membershipA uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, teamA.Id, userID).Scan(&membershipA)
	require.NoError(t, err)

	var foreignRoleID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Foreign Role', '{"settings":"write"}') RETURNING id`,
		teamB.Id,
	).Scan(&foreignRoleID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, membershipA, foreignRoleID)
	require.NoError(t, err)

	rolesByMembership, err := repo.GetRolesForMemberships(ctx, []string{membershipA.String()})
	require.NoError(t, err)
	names := make([]string, len(rolesByMembership[membershipA.String()]))
	for i, r := range rolesByMembership[membershipA.String()] {
		names[i] = r.Name
	}
	assert.Contains(t, names, "Admin", "the membership's own-team role must still be returned")
	assert.NotContains(t, names, "Foreign Role", "a role belonging to a different team must never be returned, even if membership_roles points at it")
}

func TestTeamRepository_UpdateTeam(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Update User', 'update@example.com', '#112233')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Original Name", userID, nil, nil, nil)
	require.NoError(t, err)

	newName := "Updated Name"
	updated, err := repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, tr.Id, updated.Id)
}

func TestTeamRepository_UpdateTeam_ReasonVisibilityRoleIDs_ValidatesOwnership(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Reason Vis User', 'reason-vis@example.com', '#445566')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Reason Vis Team", userID, nil, nil, nil)
	require.NoError(t, err)

	// A role from a different team must be rejected.
	otherTeam, err := repo.CreateTeam(ctx, "Other Team", userID, nil, nil, nil)
	require.NoError(t, err)
	var foreignRoleID string
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Foreign Role', '{}') RETURNING id`,
		otherTeam.Id.String(),
	).Scan(&foreignRoleID)
	require.NoError(t, err)

	_, err = repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{
		ReasonVisibilityRoleIDs: []string{foreignRoleID},
	})
	require.ErrorIs(t, err, teams.ErrRoleNotInTeam)

	// A role belonging to the team is accepted.
	var ownRoleID string
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Own Role', '{}') RETURNING id`,
		tr.Id.String(),
	).Scan(&ownRoleID)
	require.NoError(t, err)

	updated, err := repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{
		ReasonVisibilityRoleIDs: []string{ownRoleID},
	})
	require.NoError(t, err)
	require.Len(t, updated.ReasonVisibilityRoleIDs, 1)
	assert.Equal(t, uuid.MustParse(ownRoleID), updated.ReasonVisibilityRoleIDs[0])

	// Regression: `COUNT(*) FROM roles WHERE id = ANY($1)` counts matching
	// rows once per distinct role, so comparing it against len(ids) directly
	// used to wrongly reject a request that legitimately repeats the same
	// valid role ID. This only asserts the (valid) request is accepted, not
	// that storage deduplicates -- deduping the stored array is a separate,
	// unrelated concern.
	updated, err = repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{
		ReasonVisibilityRoleIDs: []string{ownRoleID, ownRoleID},
	})
	require.NoError(t, err)
	require.Len(t, updated.ReasonVisibilityRoleIDs, 2)
	assert.Equal(t, uuid.MustParse(ownRoleID), updated.ReasonVisibilityRoleIDs[0])
	assert.Equal(t, uuid.MustParse(ownRoleID), updated.ReasonVisibilityRoleIDs[1])
}

func TestTeamRepository_DeleteTeamPhoto_ClearsStoredPhoto(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Photo Delete User', 'photo-delete@example.com', '#778899')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Photo Delete Team", userID, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateTeamPhoto(ctx, tr.Id.String(), "teams/"+tr.Id.String()+"/photo"))
	withPhoto, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.True(t, withPhoto.HasPhoto)
	photoKey, err := repo.GetTeamPhotoKey(ctx, tr.Id.String())
	require.NoError(t, err)
	require.NotEmpty(t, photoKey)

	require.NoError(t, repo.DeleteTeamPhoto(ctx, tr.Id.String()))
	cleared, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.False(t, cleared.HasPhoto)
	_, err = repo.GetTeamPhotoKey(ctx, tr.Id.String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestTeamRepository_DeleteTeamPhoto_UnknownTeam_ReturnsNoRows(t *testing.T) {
	pool := testutil.NewTestDB(t)
	repo := teams.NewRepository(pool)
	err := repo.DeleteTeamPhoto(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestTeamRepository_DeleteTeamLogo_ClearsStoredLogo(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Logo Delete User', 'logo-delete@example.com', '#998877')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Logo Delete Team", userID, nil, nil, nil)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateTeamLogo(ctx, tr.Id.String(), "teams/"+tr.Id.String()+"/logo"))
	withLogo, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.True(t, withLogo.HasLogo)
	logoKey, err := repo.GetTeamLogoKey(ctx, tr.Id.String())
	require.NoError(t, err)
	require.NotEmpty(t, logoKey)

	require.NoError(t, repo.DeleteTeamLogo(ctx, tr.Id.String()))
	cleared, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.False(t, cleared.HasLogo)
	_, err = repo.GetTeamLogoKey(ctx, tr.Id.String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestTeamRepository_DeleteTeamLogo_UnknownTeam_ReturnsNoRows(t *testing.T) {
	pool := testutil.NewTestDB(t)
	repo := teams.NewRepository(pool)
	err := repo.DeleteTeamLogo(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestTeamRepository_AcceptInvite_NewMember_JoinsAndGetsDefaultMemberRole(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := teams.NewRepository(pool)

	creatorID := insertUser(t, pool, "invite-accept-creator@example.com")
	tr, err := repo.CreateTeam(ctx, "Invite Accept Team", creatorID, nil, nil, nil)
	require.NoError(t, err)

	inv, err := repo.CreateInvite(ctx, tr.Id.String(), 7*24*time.Hour)
	require.NoError(t, err)

	joinerID := insertUser(t, pool, "invite-accept-joiner@example.com")
	joined, alreadyMember, err := repo.AcceptInvite(ctx, inv.Code, joinerID)
	require.NoError(t, err)
	assert.Equal(t, tr.Id, joined.Id)
	assert.False(t, alreadyMember, "a brand-new join must not report alreadyMember")

	var membershipID string
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, tr.Id, joinerID).
		Scan(&membershipID)
	require.NoError(t, err)

	var roleName string
	err = pool.QueryRow(ctx, `
		SELECT r.name FROM membership_roles mr
		JOIN roles r ON r.id = mr.role_id
		WHERE mr.membership_id = $1
	`, membershipID).Scan(&roleName)
	require.NoError(t, err)
	assert.Equal(t, "Member", roleName)
}

func TestTeamRepository_AcceptInvite_AlreadyMember_IsIdempotentAndKeepsRoles(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := teams.NewRepository(pool)

	creatorID := insertUser(t, pool, "invite-idempotent-creator@example.com")
	tr, err := repo.CreateTeam(ctx, "Invite Idempotent Team", creatorID, nil, nil, nil)
	require.NoError(t, err)

	inv, err := repo.CreateInvite(ctx, tr.Id.String(), 7*24*time.Hour)
	require.NoError(t, err)

	// Creator re-clicking the team's own invite link is already a member and
	// already holds the (all-write) Admin role -- redeeming the code again
	// must not touch that.
	joined, alreadyMember, err := repo.AcceptInvite(ctx, inv.Code, creatorID)
	require.NoError(t, err)
	assert.Equal(t, tr.Id, joined.Id)
	assert.True(t, alreadyMember, "re-accepting as an existing member must report alreadyMember")

	var membershipID string
	err = pool.QueryRow(ctx, `SELECT id FROM memberships WHERE team_id = $1 AND user_id = $2`, tr.Id, creatorID).
		Scan(&membershipID)
	require.NoError(t, err)

	var roleCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM membership_roles WHERE membership_id = $1`, membershipID).
		Scan(&roleCount)
	require.NoError(t, err)
	assert.Equal(t, 1, roleCount, "re-accepting must not add a second (Member) role alongside the existing Admin role")
}

func TestTeamRepository_AcceptInvite_ExpiredCode_ReturnsErrInviteNotFound(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := teams.NewRepository(pool)

	creatorID := insertUser(t, pool, "invite-expired-creator@example.com")
	tr, err := repo.CreateTeam(ctx, "Invite Expired Team", creatorID, nil, nil, nil)
	require.NoError(t, err)

	inv, err := repo.CreateInvite(ctx, tr.Id.String(), 7*24*time.Hour)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE invites SET expires_at = now() - interval '1 minute' WHERE id = $1`, inv.Id)
	require.NoError(t, err)

	joinerID := insertUser(t, pool, "invite-expired-joiner@example.com")
	_, _, err = repo.AcceptInvite(ctx, inv.Code, joinerID)
	require.ErrorIs(t, err, teams.ErrInviteNotFound)
}

func TestTeamRepository_AcceptInvite_UnknownCode_ReturnsErrInviteNotFound(t *testing.T) {
	pool := testutil.NewTestDB(t)
	repo := teams.NewRepository(pool)

	joinerID := insertUser(t, pool, "invite-unknown-joiner@example.com")
	_, _, err := repo.AcceptInvite(context.Background(), "does-not-exist", joinerID)
	require.ErrorIs(t, err, teams.ErrInviteNotFound)
}

// Regression test: UpdateTeam must not take the team's
// pg_advisory_xact_lock(hashtextextended(teamID, 0)) when
// ReasonVisibilityRoleIDs is a non-nil but EMPTY slice (the caller clearing
// the list) -- otherwise clearing the list needlessly serializes against
// every other privilege-relevant mutation on the team, even though there's
// nothing here that could race with them. Mirrors the same fix already
// applied to events.Repository.validateNominatedRolesInTx.
func TestTeamRepository_UpdateTeam_ClearingReasonVisibilityRoles_DoesNotBlockOnAdvisoryLock(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := teams.NewRepository(pool)
	ctx := context.Background()

	creatorID := insertUser(t, pool, "clear-roles-creator@example.com")
	tr, err := repo.CreateTeam(ctx, "Clear Roles Team", creatorID, nil, nil, nil)
	require.NoError(t, err)
	teamID := tr.Id.String()

	lockHeld := make(chan struct{})
	lockReleased := make(chan struct{})
	go func() {
		defer close(lockReleased)
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID)
		require.NoError(t, err)
		close(lockHeld)
		time.Sleep(2 * time.Second)
		_ = tx.Rollback(ctx)
	}()

	<-lockHeld

	emptyRoleIDs := []string{}
	start := time.Now()
	_, err = repo.UpdateTeam(ctx, teamID, teams.TeamPatch{ReasonVisibilityRoleIDs: emptyRoleIDs})
	elapsed := time.Since(start)
	require.NoError(t, err)
	assert.Less(t, elapsed, 500*time.Millisecond,
		"UpdateTeam clearing ReasonVisibilityRoleIDs should not block on the team's advisory lock; took %v", elapsed)

	<-lockReleased
}
