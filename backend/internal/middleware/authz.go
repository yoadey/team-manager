package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// MembershipChecker is satisfied by members.Repository (or a mock in tests).
type MembershipChecker interface {
	IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
}

// PermissionChecker is satisfied by members.Repository — returns the effective
// per-module permissions for a (team, user) pair.
type PermissionChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// RequireMembership validates that the authenticated user is a member of the
// team identified by the {teamId} Chi URL parameter.
//
// Routes without a {teamId} parameter are passed through unchanged, so this
// middleware is safe to attach to the entire authenticated sub-router.
func RequireMembership(checker MembershipChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			teamIDStr := chi.URLParam(r, "teamId")
			if teamIDStr == "" {
				next.ServeHTTP(w, r)
				return
			}

			teamID, err := uuid.Parse(teamIDStr)
			if err != nil {
				writeProblem(w, http.StatusBadRequest, "invalid team ID")
				return
			}

			user, ok := auth.UserFromContext(r.Context())
			if !ok {
				writeProblem(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			isMember, err := checker.IsMember(r.Context(), teamID, user.Id)
			if err != nil {
				writeProblem(w, http.StatusInternalServerError, "membership check failed")
				return
			}
			if !isMember {
				writeProblem(w, http.StatusNotFound, "not found")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ─── RequirePermission ────────────────────────────────────────────────────────

// rbacModulePublic is the x-rbac-module sentinel for routes that require
// membership only, no module-level permission (team info itself, photo,
// logo, absences, notifications — see openapi.yaml for the full list).
const rbacModulePublic = "public"

// RequirePermission enforces module-level access for team-scoped routes,
// looked up from the generated rbac_table.gen.go (source of truth:
// x-rbac-module / x-rbac-self-service extensions in openapi.yaml).
//
// A request whose method+path matches no table entry is rejected with 404,
// for every HTTP method including GET — an operation with no RBAC
// classification is never reachable on membership alone.
//
// module == "public" routes require nothing beyond membership (checked
// already by RequireMembership), regardless of method.
//
// Self-service routes (attendance, comments, poll vote) are exempt from the
// module "write" requirement — any member may record their own
// attendance/comment/vote — but still require at least "read" on the module,
// since these routes read back module data (the attendance matrix, comment
// thread, or a fully assembled poll including other members' votes) that a
// module permission of "none" is meant to hide.
//
// Non-self-service routes require "read" on GET/HEAD/OPTIONS and "write" on
// mutating methods — a module permission of "none" hides reads too, not just
// writes.
func RequirePermission(checker PermissionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if chi.URLParam(r, "teamId") == "" {
				// Not a team-scoped route.
				next.ServeHTTP(w, r)
				return
			}
			enforceTeamPermission(checker, next, w, r)
		})
	}
}

// enforceTeamPermission implements RequirePermission for a request already
// known to carry a {teamId} URL parameter; split out so RequirePermission's
// closure nesting doesn't obscure this function's own branch complexity.
func enforceTeamPermission(checker PermissionChecker, next http.Handler, w http.ResponseWriter, r *http.Request) {
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	subPath := subPathAfterTeam(r.URL.Path, teamIDStr)
	entry, found := matchRBACRoute(r.Method, subPath)
	if !found {
		// No RBAC classification for this method+path — reject with 404
		// rather than allowing it through on membership alone.
		writeProblem(w, http.StatusNotFound, "unknown resource path")
		return
	}

	if entry.Module == rbacModulePublic {
		next.ServeHTTP(w, r)
		return
	}

	perms, err := checker.GetPermissions(r.Context(), teamID, user.Id)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "permission check failed")
		return
	}

	if !hasRequiredPermission(perms, entry, r.Method) {
		writeProblem(w, http.StatusForbidden, "insufficient permissions for "+entry.Module)
		return
	}

	next.ServeHTTP(w, r)
}

// hasRequiredPermission decides, for a module-gated (non-"public") route,
// whether perms satisfies it: self-service routes and read methods need only
// "read" on the module (a module set to "none" must hide reads too, not just
// writes); everything else needs "write".
func hasRequiredPermission(perms teams.PermissionsJSON, entry rbacRouteEntry, method string) bool {
	isRead := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	if entry.SelfService || isRead {
		return hasAnyPermission(perms, entry.Module)
	}
	return hasWritePermission(perms, entry.Module)
}

// matchRBACRoute finds the rbacRoutes entry for method+subPath. subPath
// segments are compared positionally against each entry's Segments, where a
// "{...}" template segment matches any non-empty literal segment (a path
// parameter value). Since every (method, path) pair is unique in the OpenAPI
// spec, at most one entry can match.
func matchRBACRoute(method, subPath string) (rbacRouteEntry, bool) {
	var reqSegments []string
	if subPath != "" {
		reqSegments = strings.Split(subPath, "/")
	}

	for _, entry := range rbacRoutes {
		if entry.Method != method || len(entry.Segments) != len(reqSegments) {
			continue
		}
		if segmentsMatch(entry.Segments, reqSegments) {
			return entry, true
		}
	}
	return rbacRouteEntry{}, false
}

func segmentsMatch(template, actual []string) bool {
	for i, seg := range template {
		if strings.HasPrefix(seg, "{") {
			continue // path parameter — matches any non-empty segment
		}
		if seg != actual[i] {
			return false
		}
	}
	return true
}

// subPathAfterTeam extracts the path segments that follow the team UUID.
// e.g. "/api/v1/teams/abc-123/events/456/attendance" → "events/456/attendance".
func subPathAfterTeam(urlPath, teamIDStr string) string {
	idx := strings.Index(urlPath, teamIDStr)
	if idx < 0 {
		return ""
	}
	rest := urlPath[idx+len(teamIDStr):]
	return strings.Trim(rest, "/")
}

// hasWritePermission returns true if the effective permissions include write for module.
func hasWritePermission(p teams.PermissionsJSON, module string) bool {
	var level string
	switch module {
	case "events":
		level = p.Events
	case "members":
		level = p.Members
	case "finances":
		level = p.Finances
	case "news":
		level = p.News
	case "polls":
		level = p.Polls
	case "settings":
		level = p.Settings
	default:
		return false
	}
	return level == "write"
}

// hasAnyPermission returns true if the effective permissions include read or
// write (i.e. anything but "none") for module.
func hasAnyPermission(p teams.PermissionsJSON, module string) bool {
	var level string
	switch module {
	case "events":
		level = p.Events
	case "members":
		level = p.Members
	case "finances":
		level = p.Finances
	case "news":
		level = p.News
	case "polls":
		level = p.Polls
	case "settings":
		level = p.Settings
	default:
		return false
	}
	return level == "read" || level == "write"
}

// writeProblem writes an RFC 9457 Problem Details response via apierror, so
// the Type URI honors ERROR_TYPE_BASE_URI like every other error response.
func writeProblem(w http.ResponseWriter, status int, detail string) {
	titles := map[int]string{
		http.StatusBadRequest:          "Bad Request",
		http.StatusUnauthorized:        "Unauthorized",
		http.StatusForbidden:           "Forbidden",
		http.StatusNotFound:            "Not Found",
		http.StatusInternalServerError: "Internal Server Error",
	}
	title, ok := titles[status]
	if !ok {
		title = http.StatusText(status)
	}
	apierror.New(status, title, detail).Render(w)
}
