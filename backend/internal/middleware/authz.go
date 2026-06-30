package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
				writeProblem(w, http.StatusForbidden, "you are not a member of this team")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ─── RequirePermission ────────────────────────────────────────────────────────

// routeModule maps the first URL segment after the {teamId} prefix to the RBAC
// module that governs write access. Segments not in this table are unknown and
// will be rejected with 404 rather than silently falling back to "settings".
var routeModule = map[string]string{
	"events":        "events",
	"members":       "members",
	"roles":         "settings",
	"news":          "news",
	"polls":         "polls",
	"finances":      "finances",
	"absences":      "", // self-service: no write check
	"notifications": "", // self-service: no write check
	// settings-level segments handled explicitly below (photo, logo, invite)
}

// knownSettingsSegments are path segments that map to the "settings" module but
// are not listed in routeModule because they are handled by explicit checks in
// moduleForPath.
var knownSettingsSegments = map[string]bool{
	"photo":  true,
	"logo":   true,
	"invite": true,
}

// selfServiceWritePaths lists sub-paths (after {teamId}) that any member may
// mutate regardless of module write permission.
var selfServiceWritePaths = map[string]bool{
	"events/attendance":  true,
	"events/comments":    true,
	"polls/vote":         true,
	"absences":           true,
	"absences/mine":      true,
	"notifications/seen": true,
}

// RequirePermission enforces module-level write access for mutating HTTP methods
// (POST, PUT, PATCH, DELETE). GET requests are always passed through (membership
// check by RequireMembership is sufficient). Self-service routes (attendance,
// comments, vote, absences, notifications) are also passed through for any member.
//
// Path parsing: given a URL like /api/v1/teams/{teamId}/events/123/attendance,
// the segment right after the teamId UUID is used to look up the module.
func RequirePermission(checker PermissionChecker) func(http.Handler) http.Handler { //nolint:gocognit // complexity inherent in RBAC permission checking
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only enforce on mutations.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			teamIDStr := chi.URLParam(r, "teamId")
			if teamIDStr == "" {
				// Not a team-scoped route; PATCH /teams/{teamId} itself falls here.
				// Check: is this a team update that needs settings write?
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

			// Determine the sub-path after /teams/{teamId}/.
			subPath := subPathAfterTeam(r.URL.Path, teamIDStr)

			// Self-service routes — any member may write.
			if isSelfService(subPath) {
				next.ServeHTTP(w, r)
				return
			}

			// Determine the required module.
			module, known := moduleForPath(subPath, r)
			if !known {
				// Unknown path segment — reject with 404 rather than silently
				// falling back to the "settings" module.
				writeProblem(w, http.StatusNotFound, "unknown resource path")
				return
			}
			if module == "" {
				// No restriction beyond membership (self-service segments).
				next.ServeHTTP(w, r)
				return
			}

			perms, err := checker.GetPermissions(r.Context(), teamID, user.Id)
			if err != nil {
				writeProblem(w, http.StatusInternalServerError, "permission check failed")
				return
			}

			if !hasWritePermission(perms, module) {
				writeProblem(w, http.StatusForbidden, "insufficient permissions for "+module)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
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

// isSelfService returns true when the sub-path is a self-service write route.
func isSelfService(subPath string) bool {
	if selfServiceWritePaths[subPath] {
		return true
	}
	// Match generic patterns like "events/{id}/attendance" or "events/{id}/comments"
	parts := strings.SplitN(subPath, "/", 3)
	if len(parts) >= 3 {
		leaf := parts[0] + "/" + parts[2]
		if selfServiceWritePaths[leaf] {
			return true
		}
	}
	return false
}

// moduleForPath returns the RBAC module name for a sub-path and whether the
// path segment is a known route. Returns ("", false) only when the first path
// segment is not recognised — callers should respond with 404.
//
// Return values:
//
//	("settings", true)  — settings module write check required
//	("events",   true)  — events module write check required
//	("",         true)  — no additional restriction (self-service or open)
//	("",         false) — unknown segment; caller must return 404
func moduleForPath(subPath string, r *http.Request) (string, bool) {
	// PATCH on the team itself (subPath == "") needs settings permission.
	if subPath == "" && r.Method == http.MethodPatch {
		return "settings", true
	}
	// photo / logo / invite are explicit settings mutations.
	first := strings.SplitN(subPath, "/", 2)[0]
	if knownSettingsSegments[first] {
		return "settings", true
	}
	m, ok := routeModule[first]
	if !ok {
		// Unknown segment — do not fall back to "settings".
		return "", false
	}
	return m, true
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

// writeProblem writes an RFC 9457 Problem Details response.
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
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://teammanager.example/errors/authz",
		"title":  title,
		"status": status,
		"detail": detail,
	})
}
