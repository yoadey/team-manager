package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/auth"
)

// MembershipChecker is satisfied by members.Repository (or a mock in tests).
type MembershipChecker interface {
	IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
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

// writeProblem writes an RFC 9457 Problem Details response.
func writeProblem(w http.ResponseWriter, status int, detail string) {
	titles := map[int]string{
		http.StatusBadRequest:          "Bad Request",
		http.StatusUnauthorized:        "Unauthorized",
		http.StatusForbidden:           "Forbidden",
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
