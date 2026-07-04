package members_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock service ─────────────────────────────────────────────────────────────

type mockMemberService struct {
	listMembers  func(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error)
	addMember    func(ctx context.Context, teamID string, params members.AddMemberParams) (*gen.Member, error)
	updateMember func(ctx context.Context, membershipID, teamID string, patch members.MemberPatch) (*gen.Member, error)
	setRoles     func(ctx context.Context, membershipID, teamID string, roleIDs []string) (*gen.Member, error)
	removeMember func(ctx context.Context, membershipID, teamID string) error
}

func (m *mockMemberService) ListMembers(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error) {
	return m.listMembers(ctx, teamID, limit, cursor)
}

func (m *mockMemberService) AddMember(ctx context.Context, teamID string, params members.AddMemberParams) (*gen.Member, error) {
	return m.addMember(ctx, teamID, params)
}

func (m *mockMemberService) UpdateMember(ctx context.Context, membershipID, teamID string, patch members.MemberPatch) (*gen.Member, error) {
	return m.updateMember(ctx, membershipID, teamID, patch)
}

func (m *mockMemberService) SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string) (*gen.Member, error) {
	return m.setRoles(ctx, membershipID, teamID, roleIDs)
}

func (m *mockMemberService) RemoveMember(ctx context.Context, membershipID, teamID string) error {
	return m.removeMember(ctx, membershipID, teamID)
}

// mockPermissionChecker returns a fixed permission set regardless of team/user.
type mockPermissionChecker struct {
	perms teams.PermissionsJSON
}

func (m *mockPermissionChecker) GetPermissions(_ context.Context, _, _ uuid.UUID) (teams.PermissionsJSON, error) {
	return m.perms, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func fixedGenMember() gen.Member {
	hasPhoto := false
	perms := gen.Permissions{
		Events: "write", Members: "write", Finances: "write",
		News: "write", Polls: "write", Settings: "write",
	}
	return gen.Member{
		MembershipId: uuid.New(),
		UserId:       uuid.New(),
		Name:         "Bob",
		Email:        "bob@example.com",
		AvatarColor:  "#bbccdd",
		HasPhoto:     &hasPhoto,
		JoinedAt:     time.Now(),
		Roles:        []gen.Role{},
		Perms:        &perms,
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestMemberHandler_AddMember_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	member := fixedGenMember()
	svc := &mockMemberService{
		addMember: func(_ context.Context, _ string, _ members.AddMemberParams) (*gen.Member, error) {
			return &member, nil
		},
	}
	var buf bytes.Buffer
	perms := &mockPermissionChecker{}
	h := members.NewHandler(svc, perms, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com"}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: teamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "member.add", rec["event"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
	assert.Equal(t, member.MembershipId.String(), rec["membershipId"])
}

func TestMemberHandler_AddMember_PhoneTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{}, slog.Default(), nil)
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	longPhone := strings.Repeat("1", 33)
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com", Phone: &longPhone}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_AddMember_GroupTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{}, slog.Default(), nil)
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	longGroup := strings.Repeat("g", 101)
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com", Group: &longGroup}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_UpdateMember_PhoneTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{}, slog.Default(), nil)
	ctx := context.Background()
	longPhone := strings.Repeat("1", 33)
	body := &gen.UpdateMemberJSONRequestBody{Phone: &longPhone}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_UpdateMember_AddressTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{}, slog.Default(), nil)
	ctx := context.Background()
	longAddress := strings.Repeat("a", 501)
	body := &gen.UpdateMemberJSONRequestBody{Address: &longAddress}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_UpdateMember_GroupTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{}, slog.Default(), nil)
	ctx := context.Background()
	longGroup := strings.Repeat("g", 101)
	body := &gen.UpdateMemberJSONRequestBody{Group: &longGroup}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_AddMember_DuplicateMembership_Returns409(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockMemberService{
		addMember: func(_ context.Context, _ string, _ members.AddMemberParams) (*gen.Member, error) {
			return nil, members.ErrDuplicateMembership
		},
	}
	h := members.NewHandler(svc, &mockPermissionChecker{}, slog.Default(), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com"}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: teamID, Body: body})

	require.Error(t, err)
	require.NotContains(t, err.Error(), "members.Handler.AddMember", "must map to the specific 409, not fall through to the generic wrapped error")
}

func TestMemberHandler_SetMemberRoles_LastSettingsAdmin_Returns409(t *testing.T) {
	t.Parallel()

	svc := &mockMemberService{
		setRoles: func(context.Context, string, string, []string) (*gen.Member, error) {
			return nil, members.ErrLastSettingsAdmin
		},
	}
	h := members.NewHandler(svc, &mockPermissionChecker{}, slog.Default(), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	body := &gen.SetMemberRolesJSONRequestBody{RoleIds: []uuid.UUID{}}
	_, err := h.SetMemberRoles(ctx, gen.SetMemberRolesRequestObject{
		TeamId: uuid.New(), MembershipId: uuid.New(), Body: body,
	})
	require.Error(t, err)
}

func TestMemberHandler_RemoveMember_LastSettingsAdmin_Returns409(t *testing.T) {
	t.Parallel()

	svc := &mockMemberService{
		removeMember: func(context.Context, string, string) error {
			return members.ErrLastSettingsAdmin
		},
	}
	h := members.NewHandler(svc, &mockPermissionChecker{}, slog.Default(), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	_, err := h.RemoveMember(ctx, gen.RemoveMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New()})
	require.Error(t, err)
}

func TestMemberHandler_AddMember_WithRoleIds_RequiresSettingsWrite(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	roleID := uuid.New()
	member := fixedGenMember()
	called := false
	svc := &mockMemberService{
		addMember: func(_ context.Context, _ string, _ members.AddMemberParams) (*gen.Member, error) {
			called = true
			return &member, nil
		},
	}
	// members:write only, no settings:write — must NOT be able to assign roles.
	perms := &mockPermissionChecker{perms: teams.PermissionsJSON{Members: "write", Settings: "none"}}
	h := members.NewHandler(svc, perms, slog.Default(), nil)

	actorID := uuid.New()
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Roster Manager", Email: "a@x.c"})
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com", RoleIds: &[]uuid.UUID{roleID}}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: teamID, Body: body})

	require.Error(t, err)
	assert.False(t, called, "service must not be invoked when the permission ceiling check fails")
}

func TestMemberHandler_AddMember_WithRoleIds_AllowedForSettingsWriter(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	roleID := uuid.New()
	member := fixedGenMember()
	svc := &mockMemberService{
		addMember: func(_ context.Context, _ string, params members.AddMemberParams) (*gen.Member, error) {
			require.Equal(t, []string{roleID.String()}, params.RoleIDs)
			return &member, nil
		},
	}
	perms := &mockPermissionChecker{perms: teams.PermissionsJSON{Members: "write", Settings: "write"}}
	h := members.NewHandler(svc, perms, slog.Default(), nil)

	actorID := uuid.New()
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com", RoleIds: &[]uuid.UUID{roleID}}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: teamID, Body: body})

	require.NoError(t, err)
}

func TestMemberHandler_AddMember_TooManyRoleIds_Returns400(t *testing.T) {
	t.Parallel()

	roleIDs := make([]uuid.UUID, 201)
	for i := range roleIDs {
		roleIDs[i] = uuid.New()
	}
	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{perms: teams.PermissionsJSON{Settings: "write"}}, slog.Default(), nil)
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	body := &gen.AddMemberJSONRequestBody{Name: "Bob", Email: "bob@example.com", RoleIds: &roleIDs}
	_, err := h.AddMember(ctx, gen.AddMemberRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_SetMemberRoles_TooManyRoleIds_Returns400(t *testing.T) {
	t.Parallel()

	roleIDs := make([]uuid.UUID, 201)
	for i := range roleIDs {
		roleIDs[i] = uuid.New()
	}
	h := members.NewHandler(&mockMemberService{}, &mockPermissionChecker{}, slog.Default(), nil)
	body := &gen.SetMemberRolesJSONRequestBody{RoleIds: roleIDs}
	_, err := h.SetMemberRoles(context.Background(), gen.SetMemberRolesRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_ListMembers(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	member := fixedGenMember()

	svc := &mockMemberService{
		listMembers: func(_ context.Context, _ string, _ int, _ string) ([]gen.Member, *string, error) {
			return []gen.Member{member}, nil, nil
		},
	}

	h := members.NewHandler(svc, &mockPermissionChecker{}, slog.Default(), nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/teams/"+teamID.String()+"/members", http.NoBody)
	w := httptest.NewRecorder()

	resp, err := h.ListMembers(req.Context(), gen.ListMembersRequestObject{
		TeamId: teamID,
	})
	require.NoError(t, err)
	_ = resp.VisitListMembersResponse(w)

	assert.Equal(t, http.StatusOK, w.Code)
	var result struct {
		Items      []gen.Member `json:"items"`
		NextCursor *string      `json:"nextCursor"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Bob", result.Items[0].Name)
}
