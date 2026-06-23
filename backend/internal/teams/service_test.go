package teams_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ─────────────────────────────────────────────────────────

type mockTeamRepo struct {
	listTeamsForUser    func(ctx context.Context, userID string) ([]teams.TeamRow, error)
	getTeam             func(ctx context.Context, teamID string) (*teams.TeamRow, error)
	createTeam          func(ctx context.Context, name, creatorUserID string) (*teams.TeamRow, error)
	updateTeam          func(ctx context.Context, teamID string, patch teams.TeamPatch) (*teams.TeamRow, error)
	getMemberCount      func(ctx context.Context, teamID string) (int, error)
	getMembership       func(ctx context.Context, teamID, userID string) (*teams.MembershipRow, error)
	getRolesForMembership func(ctx context.Context, membershipID string) ([]teams.RoleRow, error)
	createInvite        func(ctx context.Context, teamID string, ttl time.Duration) (*teams.InviteRow, error)
	updateTeamPhoto     func(ctx context.Context, teamID string, data []byte, mime string) error
}

func (m *mockTeamRepo) ListTeamsForUser(ctx context.Context, userID string) ([]teams.TeamRow, error) {
	return m.listTeamsForUser(ctx, userID)
}
func (m *mockTeamRepo) GetTeam(ctx context.Context, teamID string) (*teams.TeamRow, error) {
	return m.getTeam(ctx, teamID)
}
func (m *mockTeamRepo) CreateTeam(ctx context.Context, name, creatorUserID string) (*teams.TeamRow, error) {
	return m.createTeam(ctx, name, creatorUserID)
}
func (m *mockTeamRepo) UpdateTeam(ctx context.Context, teamID string, patch teams.TeamPatch) (*teams.TeamRow, error) {
	return m.updateTeam(ctx, teamID, patch)
}
func (m *mockTeamRepo) GetMemberCount(ctx context.Context, teamID string) (int, error) {
	return m.getMemberCount(ctx, teamID)
}
func (m *mockTeamRepo) GetMembership(ctx context.Context, teamID, userID string) (*teams.MembershipRow, error) {
	return m.getMembership(ctx, teamID, userID)
}
func (m *mockTeamRepo) GetRolesForMembership(ctx context.Context, membershipID string) ([]teams.RoleRow, error) {
	return m.getRolesForMembership(ctx, membershipID)
}
func (m *mockTeamRepo) CreateInvite(ctx context.Context, teamID string, ttl time.Duration) (*teams.InviteRow, error) {
	return m.createInvite(ctx, teamID, ttl)
}
func (m *mockTeamRepo) UpdateTeamPhoto(ctx context.Context, teamID string, data []byte, mime string) error {
	return m.updateTeamPhoto(ctx, teamID, data, mime)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func fixedTeamRow(id uuid.UUID, name string) teams.TeamRow {
	return teams.TeamRow{
		Id:        id,
		Name:      name,
		CreatedAt: time.Now(),
		ReasonVisibilityRoleIDs: []uuid.UUID{},
	}
}

func fixedMembershipRow(membershipID, teamID, userID uuid.UUID) *teams.MembershipRow {
	return &teams.MembershipRow{
		Id:       membershipID,
		TeamID:   teamID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
}

func fixedAdminRole(teamID uuid.UUID) teams.RoleRow {
	return teams.RoleRow{
		Id:     uuid.New(),
		TeamID: teamID,
		Name:   "Admin",
		System: true,
		Permissions: teams.PermissionsJSON{
			Events: "write", Members: "write", Finances: "write",
			News: "write", Polls: "write", Settings: "write",
		},
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestTeamService_ListForUser(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	userID := uuid.New()

	row := fixedTeamRow(teamID, "Alpha Team")
	membership := fixedMembershipRow(membershipID, teamID, userID)
	role := fixedAdminRole(teamID)

	repo := &mockTeamRepo{
		listTeamsForUser: func(_ context.Context, uid string) ([]teams.TeamRow, error) {
			assert.Equal(t, userID.String(), uid)
			return []teams.TeamRow{row}, nil
		},
		getMemberCount: func(_ context.Context, _ string) (int, error) { return 3, nil },
		getMembership:  func(_ context.Context, _, _ string) (*teams.MembershipRow, error) { return membership, nil },
		getRolesForMembership: func(_ context.Context, _ string) ([]teams.RoleRow, error) {
			return []teams.RoleRow{role}, nil
		},
	}

	svc := teams.NewService(repo)
	result, err := svc.ListForUser(context.Background(), userID.String())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Alpha Team", result[0].Name)
	assert.Equal(t, 3, result[0].MemberCount)
	assert.Len(t, result[0].MyRoles, 1)
	assert.Equal(t, "write", string(result[0].MyPerms.Events))
}
