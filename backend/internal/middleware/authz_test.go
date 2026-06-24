package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/middleware"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

type mockMembershipChecker struct {
	isMember bool
	err      error
}

func (m *mockMembershipChecker) IsMember(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isMember, m.err
}

type mockPermissionChecker struct {
	perms teams.PermissionsJSON
	err   error
}

func (m *mockPermissionChecker) GetPermissions(_ context.Context, _, _ uuid.UUID) (teams.PermissionsJSON, error) {
	return m.perms, m.err
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

var (
	testTeamID = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	testUserID = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

// makeChiRequest builds an *http.Request with chi URL params set.
func makeChiRequest(method, urlPath, teamIDStr string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, urlPath, http.NoBody)
	rctx := chi.NewRouteContext()
	if teamIDStr != "" {
		rctx.URLParams.Add("teamId", teamIDStr)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	// Inject authenticated user.
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.UserRow{Id: testUserID}))
	return req
}

func ok200(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

// ─── RequireMembership tests ─────────────────────────────────────────────────

func TestRequireMembership_NoTeamID_Passthrough(t *testing.T) {
	mw := middleware.RequireMembership(&mockMembershipChecker{isMember: false})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/auth/login", http.NoBody)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(ok200)).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireMembership_Member_Passes(t *testing.T) {
	mw := middleware.RequireMembership(&mockMembershipChecker{isMember: true})
	req := makeChiRequest(http.MethodGet, "/api/v1/teams/"+testTeamID.String()+"/events", testTeamID.String())
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(ok200)).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireMembership_NonMember_Forbidden(t *testing.T) {
	mw := middleware.RequireMembership(&mockMembershipChecker{isMember: false})
	req := makeChiRequest(http.MethodGet, "/api/v1/teams/"+testTeamID.String()+"/events", testTeamID.String())
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(ok200)).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// ─── RequirePermission tests ──────────────────────────────────────────────────

// writePerms returns a PermissionsJSON with "write" for all modules.
func allWritePerms() teams.PermissionsJSON {
	return teams.PermissionsJSON{
		Events: "write", Members: "write", Finances: "write",
		News: "write", Polls: "write", Settings: "write",
	}
}

// readOnlyPerms returns a PermissionsJSON with "read" for all modules.
func allReadPerms() teams.PermissionsJSON {
	return teams.PermissionsJSON{
		Events: "read", Members: "read", Finances: "read",
		News: "read", Polls: "read", Settings: "read",
	}
}

func applyPermMW(checker middleware.PermissionChecker, req *http.Request) *httptest.ResponseRecorder {
	mw := middleware.RequirePermission(checker)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(ok200)).ServeHTTP(rec, req)
	return rec
}

// GET always passes through.
func TestRequirePermission_GET_AlwaysPasses(t *testing.T) {
	perms := &mockPermissionChecker{perms: allReadPerms()}
	paths := []string{
		"/api/v1/teams/" + testTeamID.String() + "/events",
		"/api/v1/teams/" + testTeamID.String() + "/finances/transactions",
		"/api/v1/teams/" + testTeamID.String() + "/members",
	}
	for _, p := range paths {
		req := makeChiRequest(http.MethodGet, p, testTeamID.String())
		rec := applyPermMW(perms, req)
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s should pass", p)
	}
}

// Table-driven test: mutation × path × permissions → expected status.
func TestRequirePermission_Mutations(t *testing.T) {
	tid := testTeamID.String()
	evID := uuid.New().String()

	type tc struct {
		name     string
		method   string
		path     string
		perms    teams.PermissionsJSON
		wantCode int
	}

	tests := []tc{
		// events write allowed
		{"create event write ok", http.MethodPost, "/api/v1/teams/" + tid + "/events", allWritePerms(), http.StatusOK},
		// events write denied (read only)
		{"create event read denied", http.MethodPost, "/api/v1/teams/" + tid + "/events", allReadPerms(), http.StatusForbidden},
		// members
		{"invite member write ok", http.MethodPost, "/api/v1/teams/" + tid + "/members", allWritePerms(), http.StatusOK},
		{"invite member read denied", http.MethodPost, "/api/v1/teams/" + tid + "/members", allReadPerms(), http.StatusForbidden},
		// finances
		{"create transaction write ok", http.MethodPost, "/api/v1/teams/" + tid + "/finances/transactions", allWritePerms(), http.StatusOK},
		{"create transaction read denied", http.MethodPost, "/api/v1/teams/" + tid + "/finances/transactions", allReadPerms(), http.StatusForbidden},
		// news
		{"create news write ok", http.MethodPost, "/api/v1/teams/" + tid + "/news", allWritePerms(), http.StatusOK},
		{"create news read denied", http.MethodPost, "/api/v1/teams/" + tid + "/news", allReadPerms(), http.StatusForbidden},
		// polls
		{"create poll write ok", http.MethodPost, "/api/v1/teams/" + tid + "/polls", allWritePerms(), http.StatusOK},
		{"create poll read denied", http.MethodPost, "/api/v1/teams/" + tid + "/polls", allReadPerms(), http.StatusForbidden},
		// roles (settings module)
		{"create role write ok", http.MethodPost, "/api/v1/teams/" + tid + "/roles", allWritePerms(), http.StatusOK},
		{
			"create role settings denied", http.MethodPost, "/api/v1/teams/" + tid + "/roles",
			teams.PermissionsJSON{Events: "write", Members: "write", Settings: "read"},
			http.StatusForbidden,
		},
		// self-service: attendance (POST) — any member may write
		{
			"attendance self-service ok (read perms)", http.MethodPost,
			"/api/v1/teams/" + tid + "/events/" + evID + "/attendance",
			allReadPerms(), http.StatusOK,
		},
		// self-service: poll vote
		{
			"poll vote self-service ok (read perms)", http.MethodPost,
			"/api/v1/teams/" + tid + "/polls/" + uuid.New().String() + "/vote",
			allReadPerms(), http.StatusOK,
		},
		// self-service: absences
		{
			"absences self-service ok (no perms)", http.MethodPost,
			"/api/v1/teams/" + tid + "/absences",
			teams.PermissionsJSON{},
			http.StatusOK,
		},
		// self-service: event comments POST
		{
			"event comment self-service ok (read perms)", http.MethodPost,
			"/api/v1/teams/" + tid + "/events/" + evID + "/comments",
			allReadPerms(), http.StatusOK,
		},
		// self-service: notifications/seen
		{
			"notifications seen self-service ok (read perms)", http.MethodPost,
			"/api/v1/teams/" + tid + "/notifications/seen",
			allReadPerms(), http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &mockPermissionChecker{perms: tt.perms}
			req := makeChiRequest(tt.method, tt.path, tid)
			rec := applyPermMW(checker, req)
			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

// No teamId in URL → middleware is a no-op.
func TestRequirePermission_NoTeamID_Passthrough(t *testing.T) {
	checker := &mockPermissionChecker{perms: allReadPerms()}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/auth/logout", http.NoBody)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.UserRow{Id: testUserID}))
	rec := applyPermMW(checker, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
