package teams_test

import (
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
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock service ─────────────────────────────────────────────────────────────

type mockTeamService struct {
	listForUser      func(ctx context.Context, userID string) ([]gen.TeamForUser, error)
	createTeam       func(ctx context.Context, userID, name string) (*gen.TeamForUser, error)
	getTeam          func(ctx context.Context, teamID string) (*gen.Team, error)
	updateTeam       func(ctx context.Context, teamID string, patch teams.TeamPatch) (*gen.Team, error)
	createInvite     func(ctx context.Context, teamID string) (*gen.Invite, error)
	getTeamPhotoData func(ctx context.Context, teamID string) ([]byte, string, error)
	updatePhoto      func(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error)
}

func (m *mockTeamService) ListForUser(ctx context.Context, userID string) ([]gen.TeamForUser, error) {
	return m.listForUser(ctx, userID)
}

func (m *mockTeamService) CreateTeam(ctx context.Context, userID, name string) (*gen.TeamForUser, error) {
	return m.createTeam(ctx, userID, name)
}

func (m *mockTeamService) GetTeam(ctx context.Context, teamID string) (*gen.Team, error) {
	return m.getTeam(ctx, teamID)
}

func (m *mockTeamService) UpdateTeam(ctx context.Context, teamID string, patch teams.TeamPatch) (*gen.Team, error) {
	return m.updateTeam(ctx, teamID, patch)
}

func (m *mockTeamService) CreateInvite(ctx context.Context, teamID string) (*gen.Invite, error) {
	return m.createInvite(ctx, teamID)
}

func (m *mockTeamService) GetTeamPhotoData(ctx context.Context, teamID string) (data []byte, mime string, err error) {
	return m.getTeamPhotoData(ctx, teamID)
}

func (m *mockTeamService) UpdatePhoto(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error) {
	return m.updatePhoto(ctx, teamID, data, mime)
}

// fakeAuthSvc satisfies the internal authService interface for the auth.Handler.
type fakeAuthSvc struct {
	user *auth.UserRow
}

func (f *fakeAuthSvc) Login(_ context.Context, _, _ string) (string, *auth.UserRow, error) {
	return "token", f.user, nil
}

func (f *fakeAuthSvc) ValidateToken(_ context.Context, _ string) (*auth.UserRow, error) {
	return f.user, nil
}
func (f *fakeAuthSvc) Logout(_ context.Context, _ string) error { return nil }
func (f *fakeAuthSvc) UpdatePhoto(_ context.Context, _ string, _ []byte, _ string) (*auth.UserRow, error) {
	return f.user, nil
}
func (f *fakeAuthSvc) EraseAccount(_ context.Context, _, _ string) error { return nil }

// ─── helpers ─────────────────────────────────────────────────────────────────

func testAuthUser() *auth.UserRow {
	return &auth.UserRow{
		Id:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:        "Handler Test User",
		Email:       "handler@example.com",
		AvatarColor: "#6366f1",
		CreatedAt:   time.Now(),
	}
}

// testCodec is a shared session cookie codec (fixed all-zero key) for tests.
var testCodec = func() *auth.SessionCookieCodec {
	c, err := auth.NewSessionCookieCodec(make([]byte, 32), false, time.Hour, "")
	if err != nil {
		panic(err)
	}
	return c
}()

// sessionCookie builds an encrypted session cookie carrying jwt.
func sessionCookie(jwt string) *http.Cookie {
	value, err := testCodec.Encrypt(jwt)
	if err != nil {
		panic(err)
	}
	return &http.Cookie{Name: testCodec.Name(), Value: value}
}

// withAuthUser wraps a handler with auth middleware using a fake user.
func withAuthUser(h http.Handler, user *auth.UserRow) http.Handler {
	logger := slog.Default()
	authH := auth.NewHandler(&fakeAuthSvc{user: user}, logger, testCodec)
	return authH.AuthMiddleware(h)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestTeamHandler_ListTeams(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	membershipID := uuid.New()
	hasPhoto := false
	hasLogo := false

	svc := &mockTeamService{
		listForUser: func(_ context.Context, _ string) ([]gen.TeamForUser, error) {
			return []gen.TeamForUser{
				{
					Id:           teamID,
					Name:         "Test Team",
					MemberCount:  5,
					MembershipId: membershipID,
					MyRoles:      []gen.Role{},
					MyPerms: gen.Permissions{
						Events: "write", Members: "write", Finances: "write",
						News: "write", Polls: "write", Settings: "write",
					},
					HasPhoto: &hasPhoto,
					HasLogo:  &hasLogo,
				},
			}, nil
		},
	}

	h := teams.NewHandler(svc, slog.Default())

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.ListTeams(r.Context(), gen.ListTeamsRequestObject{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = resp.VisitListTeamsResponse(w)
	})

	handler := withAuthUser(inner, testAuthUser())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/teams", http.NoBody)
	req.AddCookie(sessionCookie("test-token"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []gen.TeamForUser
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "Test Team", result[0].Name)
	assert.Equal(t, 5, result[0].MemberCount)
}
