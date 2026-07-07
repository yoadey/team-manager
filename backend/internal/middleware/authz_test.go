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
	assert.Equal(t, http.StatusNotFound, rec.Code)
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

// GET on a core RBAC module requires at least "read"; "none" must be denied,
// since a module permission of "none" is meant to also hide read access.
func TestRequirePermission_GET_RequiresReadPermission(t *testing.T) {
	paths := []string{
		"/api/v1/teams/" + testTeamID.String() + "/events",
		"/api/v1/teams/" + testTeamID.String() + "/finances/transactions",
		"/api/v1/teams/" + testTeamID.String() + "/members",
	}

	readPerms := &mockPermissionChecker{perms: allReadPerms()}
	for _, p := range paths {
		req := makeChiRequest(http.MethodGet, p, testTeamID.String())
		rec := applyPermMW(readPerms, req)
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s with read perms should pass", p)
	}

	nonePerms := &mockPermissionChecker{perms: teams.PermissionsJSON{}}
	for _, p := range paths {
		req := makeChiRequest(http.MethodGet, p, testTeamID.String())
		rec := applyPermMW(nonePerms, req)
		assert.Equal(t, http.StatusForbidden, rec.Code, "GET %s with none perms should be forbidden", p)
	}
}

// Routes with no natural module mapping stay membership-gated only, even with
// "none" permissions on every module.
func TestRequirePermission_GET_UnrestrictedPaths_AlwaysPass(t *testing.T) {
	nonePerms := &mockPermissionChecker{perms: teams.PermissionsJSON{}}
	paths := []string{
		"/api/v1/teams/" + testTeamID.String(), // team info itself
		"/api/v1/teams/" + testTeamID.String() + "/photo",
		"/api/v1/teams/" + testTeamID.String() + "/logo",
		"/api/v1/teams/" + testTeamID.String() + "/absences/mine",
		"/api/v1/teams/" + testTeamID.String() + "/notifications",
	}
	for _, p := range paths {
		req := makeChiRequest(http.MethodGet, p, testTeamID.String())
		rec := applyPermMW(nonePerms, req)
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s should remain unrestricted", p)
	}
}

// Regression test: /stats has no write routes of its own, but its GET
// responses (event titles/types/dates, per-member attendance breakdowns) are
// exactly the data the "events" module's "none" is meant to hide -- gate
// reads behind events:read, the same as the events module itself.
func TestRequirePermission_GET_Stats_RequiresEventsReadPermission(t *testing.T) {
	paths := []string{
		"/api/v1/teams/" + testTeamID.String() + "/stats",
		"/api/v1/teams/" + testTeamID.String() + "/stats/members/" + uuid.New().String(),
	}

	readPerms := &mockPermissionChecker{perms: teams.PermissionsJSON{Events: "read"}}
	for _, p := range paths {
		req := makeChiRequest(http.MethodGet, p, testTeamID.String())
		rec := applyPermMW(readPerms, req)
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s with events:read should pass", p)
	}

	nonePerms := &mockPermissionChecker{perms: teams.PermissionsJSON{}}
	for _, p := range paths {
		req := makeChiRequest(http.MethodGet, p, testTeamID.String())
		rec := applyPermMW(nonePerms, req)
		assert.Equal(t, http.StatusForbidden, rec.Code, "GET %s with events:none should be forbidden", p)
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
		// members/{id}/roles: role assignment must require settings:write, not
		// just members:write — otherwise a members-write-only member could
		// self-grant admin permissions via their own membership.
		{
			"assign member roles settings write ok", http.MethodPut,
			"/api/v1/teams/" + tid + "/members/" + uuid.New().String() + "/roles",
			allWritePerms(), http.StatusOK,
		},
		{
			"assign member roles members-write-only denied", http.MethodPut,
			"/api/v1/teams/" + tid + "/members/" + uuid.New().String() + "/roles",
			teams.PermissionsJSON{Members: "write", Settings: "none"},
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
		// self-service: event comments DELETE — the trailing commentId segment
		// must not stop this from matching the same "events/comments"
		// self-service entry as the POST case above (regression test: this
		// previously fell through to requiring events:write).
		{
			"event comment delete self-service ok (read perms)", http.MethodDelete,
			"/api/v1/teams/" + tid + "/events/" + evID + "/comments/" + uuid.New().String(),
			allReadPerms(), http.StatusOK,
		},
		// events/attendance/nominations is a 4-segment path whose leaf
		// ("events" + "attendance") collapses to the same string as the
		// self-service "events/attendance" 3-segment leaf — regression test
		// for a bug where this made nominating another member self-service
		// (bypassing events:write) instead of requiring it, since nominating
		// is an events:write-only action, never self-service.
		{
			"nominations requires events:write, not self-service (read perms denied)", http.MethodPut,
			"/api/v1/teams/" + tid + "/events/" + evID + "/attendance/nominations",
			allReadPerms(), http.StatusForbidden,
		},
		{
			"nominations write ok", http.MethodPut,
			"/api/v1/teams/" + tid + "/events/" + evID + "/attendance/nominations",
			allWritePerms(), http.StatusOK,
		},
		// self-service: notifications/seen
		{
			"notifications seen self-service ok (read perms)", http.MethodPost,
			"/api/v1/teams/" + tid + "/notifications/seen",
			allReadPerms(), http.StatusOK,
		},
		// Regression: self-service exempts a member from needing "write" on
		// the module, but not from "none" -- a module permission of "none" is
		// documented to hide the module entirely, and these self-service
		// routes read back module data (the attendance matrix, comment
		// thread, or a fully assembled poll including other members' votes),
		// so they must still require at least "read".
		{
			"attendance self-service denied with events:none", http.MethodPost,
			"/api/v1/teams/" + tid + "/events/" + evID + "/attendance",
			teams.PermissionsJSON{Events: "none"},
			http.StatusForbidden,
		},
		{
			"event comments self-service denied with events:none", http.MethodPost,
			"/api/v1/teams/" + tid + "/events/" + evID + "/comments",
			teams.PermissionsJSON{Events: "none"},
			http.StatusForbidden,
		},
		{
			"poll vote self-service denied with polls:none", http.MethodPost,
			"/api/v1/teams/" + tid + "/polls/" + uuid.New().String() + "/vote",
			teams.PermissionsJSON{Polls: "none"},
			http.StatusForbidden,
		},
		// Self-standing self-service routes (no RBAC module) stay ungated even
		// with every module at "none".
		{
			"absences self-service ok with all modules none", http.MethodPost,
			"/api/v1/teams/" + tid + "/absences",
			teams.PermissionsJSON{},
			http.StatusOK,
		},
		{
			"notifications seen self-service ok with all modules none", http.MethodPost,
			"/api/v1/teams/" + tid + "/notifications/seen",
			teams.PermissionsJSON{},
			http.StatusOK,
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

// Unknown path segments must be rejected with 404, not silently mapped to "settings".
func TestRequirePermission_UnknownPathSegment_Returns404(t *testing.T) {
	tid := testTeamID.String()
	unknownPaths := []string{
		"/api/v1/teams/" + tid + "/widgets",
		"/api/v1/teams/" + tid + "/admin",
		"/api/v1/teams/" + tid + "/superpower",
		"/api/v1/teams/" + tid + "/debug/dump",
	}
	// Even with full write permissions, unknown segments must be rejected.
	checker := &mockPermissionChecker{perms: allWritePerms()}
	for _, p := range unknownPaths {
		t.Run(p, func(t *testing.T) {
			req := makeChiRequest(http.MethodPost, p, tid)
			rec := applyPermMW(checker, req)
			assert.Equal(t, http.StatusNotFound, rec.Code, "POST %s with unknown segment should return 404", p)
		})
	}
}

// Known settings-level segments (photo, logo, invite) must still require settings write.
func TestRequirePermission_SettingsSegments_RequireSettingsWrite(t *testing.T) {
	tid := testTeamID.String()
	settingsPaths := []string{
		"/api/v1/teams/" + tid + "/photo",
		"/api/v1/teams/" + tid + "/logo",
		"/api/v1/teams/" + tid + "/invite",
	}
	for _, p := range settingsPaths {
		t.Run("write ok "+p, func(t *testing.T) {
			checker := &mockPermissionChecker{perms: allWritePerms()}
			req := makeChiRequest(http.MethodPost, p, tid)
			rec := applyPermMW(checker, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
		t.Run("read denied "+p, func(t *testing.T) {
			checker := &mockPermissionChecker{perms: allReadPerms()}
			req := makeChiRequest(http.MethodPost, p, tid)
			rec := applyPermMW(checker, req)
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}
