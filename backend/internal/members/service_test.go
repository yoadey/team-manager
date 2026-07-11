package members_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ─────────────────────────────────────────────────────────

type mockMemberRepo struct {
	listMembers  func(ctx context.Context, teamID string, limit int, cur *members.ListCursor) ([]members.MemberRow, error)
	updateMember func(ctx context.Context, membershipID, teamID, callerUserID string, patch members.MemberPatch) (*members.MemberRow, error)
	setRoles     func(ctx context.Context, membershipID, teamID string, roleIDs []string, callerUserID string) (*members.MemberRow, error)
	removeMember func(ctx context.Context, membershipID, teamID string) error
}

func (m *mockMemberRepo) ListMembers(ctx context.Context, teamID string, limit int, cur *members.ListCursor) ([]members.MemberRow, error) {
	return m.listMembers(ctx, teamID, limit, cur)
}

func (m *mockMemberRepo) UpdateMember(ctx context.Context, membershipID, teamID, callerUserID string, patch members.MemberPatch) (*members.MemberRow, error) {
	return m.updateMember(ctx, membershipID, teamID, callerUserID, patch)
}

func (m *mockMemberRepo) SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string, callerUserID string) (*members.MemberRow, error) {
	return m.setRoles(ctx, membershipID, teamID, roleIDs, callerUserID)
}

func (m *mockMemberRepo) RemoveMember(ctx context.Context, membershipID, teamID string) error {
	return m.removeMember(ctx, membershipID, teamID)
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
		listMembers: func(_ context.Context, tid string, _ int, _ *members.ListCursor) ([]members.MemberRow, error) {
			assert.Equal(t, teamID.String(), tid)
			return []members.MemberRow{row}, nil
		},
	}

	svc := members.NewService(repo, nil)
	result, next, err := svc.ListMembers(context.Background(), teamID.String(), 50, "")
	require.NoError(t, err)
	assert.Nil(t, next)
	require.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, "alice@example.com", string(result[0].Email))
	require.Len(t, result[0].Roles, 1)
	assert.Equal(t, "Admin", result[0].Roles[0].Name)
	require.NotNil(t, result[0].Perms)
	assert.Equal(t, "write", string(result[0].Perms.Events))
}

func TestMemberService_UpdateMember_PassesTeamID(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	callerUserID := uuid.New()
	row := fixedMemberRow()

	repo := &mockMemberRepo{
		updateMember: func(_ context.Context, mid, tid, _ string, _ members.MemberPatch) (*members.MemberRow, error) {
			assert.Equal(t, membershipID.String(), mid)
			assert.Equal(t, teamID.String(), tid)
			return &row, nil
		},
	}

	svc := members.NewService(repo, nil)
	_, err := svc.UpdateMember(context.Background(), membershipID.String(), teamID.String(), callerUserID.String(), members.MemberPatch{})
	require.NoError(t, err)
}

func TestMemberService_UpdateMember_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	repo := &mockMemberRepo{
		updateMember: func(context.Context, string, string, string, members.MemberPatch) (*members.MemberRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := members.NewService(repo, nil)
	_, err := svc.UpdateMember(context.Background(), uuid.New().String(), uuid.New().String(), uuid.New().String(), members.MemberPatch{})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestMemberService_SetRoles_PassesTeamID(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	callerUserID := uuid.New()
	roleIDs := []string{uuid.New().String()}
	row := fixedMemberRow()

	repo := &mockMemberRepo{
		setRoles: func(_ context.Context, mid, tid string, gotRoleIDs []string, gotCaller string) (*members.MemberRow, error) {
			assert.Equal(t, membershipID.String(), mid)
			assert.Equal(t, teamID.String(), tid)
			assert.Equal(t, roleIDs, gotRoleIDs)
			assert.Equal(t, callerUserID.String(), gotCaller)
			return &row, nil
		},
	}

	svc := members.NewService(repo, nil)
	_, err := svc.SetRoles(context.Background(), membershipID.String(), teamID.String(), roleIDs, callerUserID.String())
	require.NoError(t, err)
}

func TestMemberService_SetRoles_RoleNotInTeam_Propagates(t *testing.T) {
	t.Parallel()

	repo := &mockMemberRepo{
		setRoles: func(context.Context, string, string, []string, string) (*members.MemberRow, error) {
			return nil, members.ErrRoleNotInTeam
		},
	}

	svc := members.NewService(repo, nil)
	_, err := svc.SetRoles(context.Background(), uuid.New().String(), uuid.New().String(), []string{uuid.New().String()}, uuid.New().String())
	require.ErrorIs(t, err, members.ErrRoleNotInTeam)
}

func TestMemberService_SetRoles_LastSettingsAdmin_Propagates(t *testing.T) {
	t.Parallel()

	repo := &mockMemberRepo{
		setRoles: func(context.Context, string, string, []string, string) (*members.MemberRow, error) {
			return nil, members.ErrLastSettingsAdmin
		},
	}

	svc := members.NewService(repo, nil)
	_, err := svc.SetRoles(context.Background(), uuid.New().String(), uuid.New().String(), []string{}, uuid.New().String())
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)
}

func TestMemberService_SetRoles_InsufficientPermissionToGrant_Propagates(t *testing.T) {
	t.Parallel()

	repo := &mockMemberRepo{
		setRoles: func(context.Context, string, string, []string, string) (*members.MemberRow, error) {
			return nil, members.ErrInsufficientPermissionToGrant
		},
	}

	svc := members.NewService(repo, nil)
	_, err := svc.SetRoles(context.Background(), uuid.New().String(), uuid.New().String(), []string{uuid.New().String()}, uuid.New().String())
	require.ErrorIs(t, err, members.ErrInsufficientPermissionToGrant)
}

func TestMemberService_RemoveMember_LastSettingsAdmin_Propagates(t *testing.T) {
	t.Parallel()

	repo := &mockMemberRepo{
		removeMember: func(context.Context, string, string) error {
			return members.ErrLastSettingsAdmin
		},
	}

	svc := members.NewService(repo, nil)
	err := svc.RemoveMember(context.Background(), uuid.New().String(), uuid.New().String())
	require.ErrorIs(t, err, members.ErrLastSettingsAdmin)
}

func TestMemberService_RemoveMember_PassesTeamID(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	called := false

	repo := &mockMemberRepo{
		removeMember: func(_ context.Context, mid, tid string) error {
			assert.Equal(t, membershipID.String(), mid)
			assert.Equal(t, teamID.String(), tid)
			called = true
			return nil
		},
	}

	svc := members.NewService(repo, nil)
	err := svc.RemoveMember(context.Background(), membershipID.String(), teamID.String())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestMemberService_RemoveMember_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	repo := &mockMemberRepo{
		removeMember: func(context.Context, string, string) error {
			return pgx.ErrNoRows
		},
	}

	svc := members.NewService(repo, nil)
	err := svc.RemoveMember(context.Background(), uuid.New().String(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
