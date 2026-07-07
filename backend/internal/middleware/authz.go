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

// selfServiceWritePathsWithTrailingID lists the subset of selfServiceWritePaths
// leaves that are also reached via a 4-segment path with a trailing
// sub-resource ID, e.g. DELETE .../events/{eventId}/comments/{commentId}.
// Deliberately a separate, narrower set from selfServiceWritePaths — matching
// every 4-segment path against the full self-service set previously
// misclassified PUT .../events/{eventId}/attendance/nominations (nominating
// another member, an events:write-only action never exposed to non-writers
// in the UI) as self-service, since "events"+"attendance" also collapses to
// the "events/attendance" leaf. Only list a leaf here when its real 4th path
// segment is a resource ID, never a fixed route keyword like "nominations".
var selfServiceWritePathsWithTrailingID = map[string]bool{
	"events/comments": true,
}

// RequirePermission enforces module-level access for team-scoped routes.
//
// Mutating methods (POST, PUT, PATCH, DELETE) require "write" on the relevant
// module, as before. Self-service routes (attendance, comments, vote,
// absences, notifications) are passed through for any member regardless of
// method and without requiring module "write" — but a self-service leaf
// mapped in selfServiceModule (events/attendance, events/comments,
// polls/vote) still requires at least "read" on its module, since these
// routes can read back module data (the attendance matrix, comment thread,
// or a fully assembled poll including other members' votes) that "none" is
// meant to hide. Self-standing self-service routes with no module mapping
// (absences, notifications/seen) remain ungated, exactly as before.
//
// GET/HEAD/OPTIONS additionally require at least "read" (i.e. not "none") on
// the six core RBAC modules (events, members, finances, news, polls, settings
// via /roles) — a module permission of "none" must also hide read access, not
// just block writes. Routes with no natural module mapping (team info itself,
// photo, logo, invite, stats) remain gated by membership only, exactly as
// before; they carry no module-level sensitivity or don't correspond to one of
// the six modules.
//
// Path parsing: given a URL like /api/v1/teams/{teamId}/events/123/attendance,
// the segment right after the teamId UUID is used to look up the module.
func RequirePermission(checker PermissionChecker) func(http.Handler) http.Handler { //nolint:gocognit,cyclop // complexity inherent in RBAC permission checking
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isRead := r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions

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

			// Self-service routes — any member may read or write, but a leaf
			// mapped in selfServiceModule still requires at least "read" on
			// its module: self-service exempts "write", not "none".
			if leaf, ok := selfServiceLeaf(subPath); ok {
				if module, gated := selfServiceModule[leaf]; gated {
					perms, err := checker.GetPermissions(r.Context(), teamID, user.Id)
					if err != nil {
						writeProblem(w, http.StatusInternalServerError, "permission check failed")
						return
					}
					if !hasAnyPermission(perms, module) {
						writeProblem(w, http.StatusForbidden, "insufficient permissions for "+module)
						return
					}
				}
				next.ServeHTTP(w, r)
				return
			}

			if isRead {
				module, restrict := readModuleForPath(subPath)
				if !restrict {
					// No module mapping for reads (team info, photo, logo,
					// invite, stats) — membership check is sufficient.
					next.ServeHTTP(w, r)
					return
				}
				perms, err := checker.GetPermissions(r.Context(), teamID, user.Id)
				if err != nil {
					writeProblem(w, http.StatusInternalServerError, "permission check failed")
					return
				}
				if !hasAnyPermission(perms, module) {
					writeProblem(w, http.StatusForbidden, "insufficient permissions for "+module)
					return
				}
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

// readModuleForPath returns the RBAC module that governs GET-visibility for a
// sub-path, and whether that module should be enforced at all. Only the six
// core RBAC modules (via routeModule, excluding the "" self-service entries)
// are read-restricted; team info, photo/logo, invite, and stats have no
// module-level sensitivity and remain visible to any team member.
func readModuleForPath(subPath string) (module string, restrict bool) {
	first := strings.SplitN(subPath, "/", 2)[0]
	m, ok := routeModule[first]
	if !ok || m == "" {
		return "", false
	}
	return m, true
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

// selfServiceModule maps a self-service leaf (as computed by selfServiceLeaf)
// to the RBAC module a member must have at least "read" on to use it.
// Self-service exempts a member from needing "write" on the module — they
// may always record their own attendance/vote/comment regardless of write
// permission — but a module permission of "none" is documented (see
// RequirePermission below) to hide that module entirely, not just block
// writes. Without this map, a member demoted to "none" could still read the
// full attendance matrix / comment thread, or vote and receive back the
// fully assembled poll (question, options, vote counts, and other members'
// names for non-anonymous polls) by hitting the self-service route
// directly, bypassing the "none" guarantee. Leaves with no module mapping
// here (absences, notifications/seen) are self-standing and not gated by
// any of the six RBAC modules.
var selfServiceModule = map[string]string{
	"events/attendance": "events",
	"events/comments":   "events",
	"polls/vote":        "polls",
}

// selfServiceLeaf returns the canonical self-service key for subPath (as
// used by selfServiceWritePaths / selfServiceModule) and whether subPath is
// a self-service route at all.
func selfServiceLeaf(subPath string) (leaf string, ok bool) {
	if selfServiceWritePaths[subPath] {
		return subPath, true
	}
	parts := strings.Split(subPath, "/")
	// "events/{id}/attendance" or "events/{id}/comments" (create) — a plain
	// 3-segment resource path.
	if len(parts) == 3 {
		leaf := parts[0] + "/" + parts[2]
		if selfServiceWritePaths[leaf] {
			return leaf, true
		}
	}
	// "events/{id}/comments/{commentId}" (delete one's own comment) — a
	// trailing sub-resource ID doesn't change which self-service route this
	// is. Checked against the narrower selfServiceWritePathsWithTrailingID,
	// NOT the full selfServiceWritePaths set: the latter would also match
	// "events/{id}/attendance/nominations", whose 4th segment is a fixed
	// route keyword (not a resource ID) naming an events:write-only action,
	// not a self-service one. The ownership check for the trailing-ID case
	// (e.g. a comment delete scoped to its own author) happens below in the
	// handler/service, not here.
	if len(parts) == 4 {
		leaf := parts[0] + "/" + parts[2]
		if selfServiceWritePathsWithTrailingID[leaf] {
			return leaf, true
		}
	}
	return "", false
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
	// members/{membershipId}/roles assigns role-derived privileges (including
	// settings:write itself) to a member, so it requires settings write —
	// members:write alone must not be enough to self-grant admin access.
	if isMemberRolesPath(subPath) {
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

// isMemberRolesPath returns true for "members/{membershipId}/roles".
func isMemberRolesPath(subPath string) bool {
	parts := strings.Split(subPath, "/")
	return len(parts) == 3 && parts[0] == "members" && parts[2] == "roles"
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
