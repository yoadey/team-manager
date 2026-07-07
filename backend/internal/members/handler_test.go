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
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/members"
)

// ─── mock service ─────────────────────────────────────────────────────────────

type mockMemberService struct {
	listMembers  func(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error)
	updateMember func(ctx context.Context, membershipID, teamID string, patch members.MemberPatch) (*gen.Member, error)
	setRoles     func(ctx context.Context, membershipID, teamID string, roleIDs []string) (*gen.Member, error)
	removeMember func(ctx context.Context, membershipID, teamID string) error
}

func (m *mockMemberService) ListMembers(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error) {
	return m.listMembers(ctx, teamID, limit, cursor)
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

func TestMemberHandler_UpdateMember_PhoneTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, slog.Default(), nil)
	ctx := context.Background()
	longPhone := strings.Repeat("1", 33)
	body := &gen.UpdateMemberJSONRequestBody{Phone: &longPhone}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_UpdateMember_AddressTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, slog.Default(), nil)
	ctx := context.Background()
	longAddress := strings.Repeat("a", 501)
	body := &gen.UpdateMemberJSONRequestBody{Address: &longAddress}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestMemberHandler_UpdateMember_GroupTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, slog.Default(), nil)
	ctx := context.Background()
	longGroup := strings.Repeat("g", 101)
	body := &gen.UpdateMemberJSONRequestBody{Group: &longGroup}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
}

// Regression test: unlike every other free-text/date field, birthday had no
// server-side range validation at all -- a members:write holder could set an
// arbitrary, nonsensical date (in the future, or centuries in the past) with
// no rejection.
func TestMemberHandler_UpdateMember_BirthdayOutOfRange_Returns400(t *testing.T) {
	t.Parallel()

	h := members.NewHandler(&mockMemberService{}, slog.Default(), nil)
	ctx := context.Background()

	future := openapi_types.Date{Time: time.Now().AddDate(0, 0, 1)}
	_, err := h.UpdateMember(ctx, gen.UpdateMemberRequestObject{
		TeamId: uuid.New(), MembershipId: uuid.New(),
		Body: &gen.UpdateMemberJSONRequestBody{Birthday: &future},
	})
	require.Error(t, err, "future birthday must be rejected")

	tooOld := openapi_types.Date{Time: time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)}
	_, err = h.UpdateMember(ctx, gen.UpdateMemberRequestObject{
		TeamId: uuid.New(), MembershipId: uuid.New(),
		Body: &gen.UpdateMemberJSONRequestBody{Birthday: &tooOld},
	})
	require.Error(t, err, "birthday before 1900 must be rejected")
}

// Regression test: unlike a plain wrapped error, UpdateMember's users.email
// UNIQUE violation used to have no special handling at all, so changing a
// member's email to one already used by a different account surfaced as a
// raw wrapped error -> generic 500, instead of a clean 409.
func TestMemberHandler_UpdateMember_EmailTaken_Returns409(t *testing.T) {
	t.Parallel()

	svc := &mockMemberService{
		updateMember: func(context.Context, string, string, members.MemberPatch) (*gen.Member, error) {
			return nil, members.ErrEmailTaken
		},
	}
	h := members.NewHandler(svc, slog.Default(), nil)

	email := openapi_types.Email("taken@example.com")
	body := &gen.UpdateMemberJSONRequestBody{Email: &email}
	_, err := h.UpdateMember(context.Background(), gen.UpdateMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New(), Body: body})

	require.Error(t, err)
	require.NotContains(t, err.Error(), "members.Handler.UpdateMember", "must map to the specific 409, not fall through to the generic wrapped error")
}

func TestMemberHandler_SetMemberRoles_LastSettingsAdmin_Returns409(t *testing.T) {
	t.Parallel()

	svc := &mockMemberService{
		setRoles: func(context.Context, string, string, []string) (*gen.Member, error) {
			return nil, members.ErrLastSettingsAdmin
		},
	}
	h := members.NewHandler(svc, slog.Default(), nil)

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
	h := members.NewHandler(svc, slog.Default(), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	_, err := h.RemoveMember(ctx, gen.RemoveMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New()})
	require.Error(t, err)
}

// Regression test: a rejected attempt to strip roles from or remove the last
// settings-admin -- a security-relevant rejected privilege change -- used to
// leave no audit trail at all, unlike every successful role/member change.
func TestMemberHandler_SetMemberRoles_LastSettingsAdmin_RecordsAuditFailure(t *testing.T) {
	t.Parallel()

	svc := &mockMemberService{
		setRoles: func(context.Context, string, string, []string) (*gen.Member, error) {
			return nil, members.ErrLastSettingsAdmin
		},
	}
	var buf bytes.Buffer
	h := members.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	body := &gen.SetMemberRolesJSONRequestBody{RoleIds: []uuid.UUID{}}
	_, err := h.SetMemberRoles(ctx, gen.SetMemberRolesRequestObject{
		TeamId: uuid.New(), MembershipId: uuid.New(), Body: body,
	})
	require.Error(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "member.roles_change", rec["event"])
	assert.Equal(t, "failure", rec["outcome"])
	assert.Equal(t, "last_settings_admin", rec["reason"])
}

func TestMemberHandler_RemoveMember_LastSettingsAdmin_RecordsAuditFailure(t *testing.T) {
	t.Parallel()

	svc := &mockMemberService{
		removeMember: func(context.Context, string, string) error {
			return members.ErrLastSettingsAdmin
		},
	}
	var buf bytes.Buffer
	h := members.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	_, err := h.RemoveMember(ctx, gen.RemoveMemberRequestObject{TeamId: uuid.New(), MembershipId: uuid.New()})
	require.Error(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "member.remove", rec["event"])
	assert.Equal(t, "failure", rec["outcome"])
	assert.Equal(t, "last_settings_admin", rec["reason"])
}

func TestMemberHandler_SetMemberRoles_TooManyRoleIds_Returns400(t *testing.T) {
	t.Parallel()

	roleIDs := make([]uuid.UUID, 201)
	for i := range roleIDs {
		roleIDs[i] = uuid.New()
	}
	h := members.NewHandler(&mockMemberService{}, slog.Default(), nil)
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

	h := members.NewHandler(svc, slog.Default(), nil)

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
