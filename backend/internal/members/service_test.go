package members_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ─────────────────────────────────────────────────────────

type mockMemberRepo struct {
	listMembers  func(ctx context.Context, teamID string) ([]members.MemberRow, error)
	addMember    func(ctx context.Context, teamID string, params members.AddMemberParams) (*members.MemberRow, error)
	updateMember func(ctx context.Context, membershipID string, patch members.MemberPatch) (*members.MemberRow, error)
	setRoles     func(ctx context.Context, membershipID string, roleIDs []string) (*members.MemberRow, error)
	removeMember func(ctx context.Context, membershipID string) error
}

func (m *mockMemberRepo) ListMembers(ctx context.Context, teamID string) ([]members.MemberRow, error) {
	return m.listMembers(ctx, teamID)
}
func (m *mockMemberRepo) AddMember(ctx context.Context, teamID string, params members.AddMemberParams) (*members.MemberRow, error) {
	return m.addMember(ctx, teamID, params)
}
func (m *mockMemberRepo) UpdateMember(ctx context.Context, membershipID string, patch members.MemberPatch) (*members.MemberRow, error) {
	return m.updateMember(ctx, membershipID, patch)
}
func (m *mockMemberRepo) SetRoles(ctx context.Context, membershipID string, roleIDs []string) (*members.MemberRow, error) {
	return m.setRoles(ctx, membershipID, roleIDs)
}
func (m *mockMemberRepo) RemoveMember(ctx context.Context, membershipID string) error {
	return m.removeMember(ctx, membershipID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func fixedMemberRow() members.MemberRow {
	return members.MemberRow{
		MembershipID: uuid.New(),
		UserID:       uuid.New(),
		Name:         "Alice",
		Email:        "alice@example.com",
		AvatarColor:  "#aabbcc",
		JoinedAt:     time.Now(),
		Roles: []teams.RoleRow{
			{
				Id:     uuid.New(),
				TeamID: uuid.New(),
				Name:   "Admin",
				System: true,
				Permissions: teams.PermissionsJSON{
					Events: "write", Members: "write", Finances: "write",
					News: "write", Polls: "write", Settings: "write",
				},
			},
		},
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestMemberService_ListMembers(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	row := fixedMemberRow()

	repo := &mockMemberRepo{
		listMembers: func(_ context.Context, tid string) ([]members.MemberRow, error) {
			assert.Equal(t, teamID.String(), tid)
			return []members.MemberRow{row}, nil
		},
	}

	svc := members.NewService(repo)
	result, err := svc.ListMembers(context.Background(), teamID.String())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, "alice@example.com", string(result[0].Email))
	require.Len(t, result[0].Roles, 1)
	assert.Equal(t, "Admin", result[0].Roles[0].Name)
	require.NotNil(t, result[0].Perms)
	assert.Equal(t, "write", string(result[0].Perms.Events))
}
