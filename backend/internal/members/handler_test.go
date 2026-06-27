package members_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/members"
)

// ─── mock service ─────────────────────────────────────────────────────────────

type mockMemberService struct {
	listMembers  func(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error)
	addMember    func(ctx context.Context, teamID string, params members.AddMemberParams) (*gen.Member, error)
	updateMember func(ctx context.Context, membershipID string, patch members.MemberPatch) (*gen.Member, error)
	setRoles     func(ctx context.Context, membershipID string, roleIDs []string) (*gen.Member, error)
	removeMember func(ctx context.Context, membershipID string) error
}

func (m *mockMemberService) ListMembers(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error) {
	return m.listMembers(ctx, teamID, limit, cursor)
}

func (m *mockMemberService) AddMember(ctx context.Context, teamID string, params members.AddMemberParams) (*gen.Member, error) {
	return m.addMember(ctx, teamID, params)
}

func (m *mockMemberService) UpdateMember(ctx context.Context, membershipID string, patch members.MemberPatch) (*gen.Member, error) {
	return m.updateMember(ctx, membershipID, patch)
}

func (m *mockMemberService) SetRoles(ctx context.Context, membershipID string, roleIDs []string) (*gen.Member, error) {
	return m.setRoles(ctx, membershipID, roleIDs)
}

func (m *mockMemberService) RemoveMember(ctx context.Context, membershipID string) error {
	return m.removeMember(ctx, membershipID)
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
	h := members.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)))

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

func TestMemberHandler_ListMembers(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	member := fixedGenMember()

	svc := &mockMemberService{
		listMembers: func(_ context.Context, _ string, _ int, _ string) ([]gen.Member, *string, error) {
			return []gen.Member{member}, nil, nil
		},
	}

	h := members.NewHandler(svc, slog.Default())

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
