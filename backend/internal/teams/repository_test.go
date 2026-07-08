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
	tr, err := repo.CreateTeam(ctx, "My Team", userID)
	require.NoError(t, err)
	assert.NotEmpty(t, tr.Id.String())
	assert.Equal(t, "My Team", tr.Name)
	assert.False(t, tr.CreatedAt.IsZero())
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
	tr, err := repo.CreateTeam(ctx, "Default Role Team", userID)
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
	_, err = repo.CreateTeam(ctx, "List Team", userID)
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
	teamA, err := repo.CreateTeam(ctx, "Batch Team A", userID)
	require.NoError(t, err)
	teamB, err := repo.CreateTeam(ctx, "Batch Team B", userID)
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
	tr, err := repo.CreateTeam(ctx, "Original Name", userID)
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
	tr, err := repo.CreateTeam(ctx, "Reason Vis Team", userID)
	require.NoError(t, err)

	// A role from a different team must be rejected.
	otherTeam, err := repo.CreateTeam(ctx, "Other Team", userID)
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
	tr, err := repo.CreateTeam(ctx, "Photo Delete Team", userID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateTeamPhoto(ctx, tr.Id.String(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"))
	withPhoto, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.True(t, withPhoto.HasPhoto)
	photoData, photoMime, err := repo.GetTeamPhotoBytes(ctx, tr.Id.String())
	require.NoError(t, err)
	require.NotEmpty(t, photoData)
	require.NotNil(t, photoMime)

	require.NoError(t, repo.DeleteTeamPhoto(ctx, tr.Id.String()))
	cleared, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.False(t, cleared.HasPhoto)
	_, clearedMime, err := repo.GetTeamPhotoBytes(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.Nil(t, clearedMime)
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
	tr, err := repo.CreateTeam(ctx, "Logo Delete Team", userID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateTeamLogo(ctx, tr.Id.String(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"))
	withLogo, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.True(t, withLogo.HasLogo)
	logoData, logoMime, err := repo.GetTeamLogoBytes(ctx, tr.Id.String())
	require.NoError(t, err)
	require.NotEmpty(t, logoData)
	require.NotNil(t, logoMime)

	require.NoError(t, repo.DeleteTeamLogo(ctx, tr.Id.String()))
	cleared, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.False(t, cleared.HasLogo)
	_, clearedMime, err := repo.GetTeamLogoBytes(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.Nil(t, clearedMime)
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
	tr, err := repo.CreateTeam(ctx, "Invite Accept Team", creatorID)
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
	tr, err := repo.CreateTeam(ctx, "Invite Idempotent Team", creatorID)
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
	tr, err := repo.CreateTeam(ctx, "Invite Expired Team", creatorID)
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
