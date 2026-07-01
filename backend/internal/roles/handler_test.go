package roles_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/yoadey/team-manager/backend/internal/roles"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockRoleService struct {
	listRoles  func(ctx context.Context, teamID uuid.UUID) ([]gen.Role, error)
	createRole func(ctx context.Context, teamID uuid.UUID, body *gen.CreateRoleJSONRequestBody) (*gen.Role, error)
	updateRole func(ctx context.Context, roleID, teamID uuid.UUID, body *gen.UpdateRoleJSONRequestBody) (*gen.Role, error)
	deleteRole func(ctx context.Context, roleID, teamID uuid.UUID) error
}

func (m *mockRoleService) ListRoles(ctx context.Context, teamID uuid.UUID) ([]gen.Role, error) {
	return m.listRoles(ctx, teamID)
}

func (m *mockRoleService) CreateRole(ctx context.Context, teamID uuid.UUID, body *gen.CreateRoleJSONRequestBody) (*gen.Role, error) {
	return m.createRole(ctx, teamID, body)
}

func (m *mockRoleService) UpdateRole(ctx context.Context, roleID, teamID uuid.UUID, body *gen.UpdateRoleJSONRequestBody) (*gen.Role, error) {
	return m.updateRole(ctx, roleID, teamID, body)
}

func (m *mockRoleService) DeleteRole(ctx context.Context, roleID, teamID uuid.UUID) error {
	return m.deleteRole(ctx, roleID, teamID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

var (
	rolesTeamID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	testRoleID  = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
)

func rolesAuthedCtx() context.Context {
	user := &auth.UserRow{
		Id:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Name:        "Test User",
		Email:       "test@example.com",
		AvatarColor: "#6366f1",
		CreatedAt:   time.Now(),
	}
	return auth.ContextWithUser(context.Background(), user)
}

func testRole() gen.Role {
	color := "#ff0000"
	return gen.Role{
		Id:     testRoleID,
		TeamId: rolesTeamID,
		Name:   "Coach",
		System: false,
		Color:  &color,
		Permissions: gen.Permissions{
			Events:   "write",
			Members:  "read",
			Finances: "none",
			News:     "write",
			Polls:    "read",
			Settings: "none",
		},
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestHandler_ListRoles_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := roles.NewHandler(&mockRoleService{}, slog.Default(), nil)
	_, err := h.ListRoles(context.Background(), gen.ListRolesRequestObject{TeamId: rolesTeamID})
	require.Error(t, err)
}

func TestHandler_ListRoles_Success(t *testing.T) {
	t.Parallel()
	role := testRole()
	svc := &mockRoleService{
		listRoles: func(_ context.Context, _ uuid.UUID) ([]gen.Role, error) {
			return []gen.Role{role}, nil
		},
	}
	h := roles.NewHandler(svc, slog.Default(), nil)

	resp, err := h.ListRoles(rolesAuthedCtx(), gen.ListRolesRequestObject{TeamId: rolesTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListRolesResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var result []gen.Role
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "Coach", result[0].Name)
}

func TestHandler_ListRoles_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockRoleService{
		listRoles: func(_ context.Context, _ uuid.UUID) ([]gen.Role, error) {
			return nil, errors.New("db error")
		},
	}
	h := roles.NewHandler(svc, slog.Default(), nil)
	_, err := h.ListRoles(rolesAuthedCtx(), gen.ListRolesRequestObject{TeamId: rolesTeamID})
	require.Error(t, err)
}

func TestHandler_CreateRole_MissingBody(t *testing.T) {
	t.Parallel()
	h := roles.NewHandler(&mockRoleService{}, slog.Default(), nil)
	_, err := h.CreateRole(rolesAuthedCtx(), gen.CreateRoleRequestObject{TeamId: rolesTeamID, Body: nil})
	require.Error(t, err)
}

func TestHandler_CreateRole_Success(t *testing.T) {
	t.Parallel()
	role := testRole()
	svc := &mockRoleService{
		createRole: func(_ context.Context, _ uuid.UUID, body *gen.CreateRoleJSONRequestBody) (*gen.Role, error) {
			assert.Equal(t, "Coach", body.Name)
			return &role, nil
		},
	}
	h := roles.NewHandler(svc, slog.Default(), nil)

	body := &gen.CreateRoleJSONRequestBody{
		Name: "Coach",
		Permissions: gen.Permissions{
			Events:   "write",
			Members:  "read",
			Finances: "none",
			News:     "write",
			Polls:    "read",
			Settings: "none",
		},
	}
	resp, err := h.CreateRole(rolesAuthedCtx(), gen.CreateRoleRequestObject{TeamId: rolesTeamID, Body: body})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreateRoleResponse(w))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateRole_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	role := testRole()
	svc := &mockRoleService{
		createRole: func(_ context.Context, _ uuid.UUID, _ *gen.CreateRoleJSONRequestBody) (*gen.Role, error) {
			return &role, nil
		},
	}
	var buf bytes.Buffer
	h := roles.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreateRoleJSONRequestBody{Name: "Coach", Permissions: gen.Permissions{
		Events: "write", Members: "read", Finances: "none", News: "write", Polls: "read", Settings: "none",
	}}
	_, err := h.CreateRole(rolesAuthedCtx(), gen.CreateRoleRequestObject{TeamId: rolesTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "role.create", rec["event"])
	assert.Equal(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", rec["actor"])
	assert.Equal(t, testRoleID.String(), rec["roleId"])
}

func TestHandler_DeleteRole_Success(t *testing.T) {
	t.Parallel()
	svc := &mockRoleService{
		deleteRole: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	h := roles.NewHandler(svc, slog.Default(), nil)

	resp, err := h.DeleteRole(rolesAuthedCtx(), gen.DeleteRoleRequestObject{
		TeamId: rolesTeamID,
		RoleId: testRoleID,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitDeleteRoleResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}
