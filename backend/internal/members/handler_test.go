package members_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/members"
)

// ─── mock service ─────────────────────────────────────────────────────────────

type mockMemberService struct {
	listMembers  func(ctx context.Context, teamID string) ([]gen.Member, error)
	addMember    func(ctx context.Context, teamID string, params members.AddMemberParams) (*gen.Member, error)
	updateMember func(ctx context.Context, membershipID string, patch members.MemberPatch) (*gen.Member, error)
	setRoles     func(ctx context.Context, membershipID string, roleIDs []string) (*gen.Member, error)
	removeMember func(ctx context.Context, membershipID string) error
}

func (m *mockMemberService) ListMembers(ctx context.Context, teamID string) ([]gen.Member, error) {
	return m.listMembers(ctx, teamID)
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
		MembershipId: openapi_types.UUID(uuid.New()),
		UserId:       openapi_types.UUID(uuid.New()),
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

func TestMemberHandler_ListMembers(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	member := fixedGenMember()

	svc := &mockMemberService{
		listMembers: func(_ context.Context, _ string) ([]gen.Member, error) {
			return []gen.Member{member}, nil
		},
	}

	h := members.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.String()+"/members", nil)
	w := httptest.NewRecorder()

	resp, err := h.ListMembers(req.Context(), gen.ListMembersRequestObject{
		TeamId: openapi_types.UUID(teamID),
	})
	require.NoError(t, err)
	_ = resp.VisitListMembersResponse(w)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []gen.Member
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "Bob", result[0].Name)
}
