package teams_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock service ─────────────────────────────────────────────────────────────

type mockTeamService struct {
	listForUser     func(ctx context.Context, userID string) ([]gen.TeamForUser, error)
	createTeam      func(ctx context.Context, userID, name string, icon, iconBg, iconFg *string) (*gen.TeamForUser, error)
	getTeam         func(ctx context.Context, teamID string) (*gen.Team, error)
	updateTeam      func(ctx context.Context, teamID string, patch teams.TeamPatch) (*gen.Team, error)
	createInvite    func(ctx context.Context, teamID string) (*gen.Invite, error)
	acceptInvite    func(ctx context.Context, code, userID string) (*gen.AcceptInviteResponse, error)
	getTeamPhotoURL func(ctx context.Context, teamID string) (string, error)
	updatePhoto     func(ctx context.Context, teamID string, data []byte, mimeType string) (*gen.Team, error)
	deletePhoto     func(ctx context.Context, teamID string) error
	getTeamLogoURL  func(ctx context.Context, teamID string) (string, error)
	updateLogo      func(ctx context.Context, teamID string, data []byte, mimeType string) (*gen.Team, error)
	deleteLogo      func(ctx context.Context, teamID string) error
}

func (m *mockTeamService) ListForUser(ctx context.Context, userID string) ([]gen.TeamForUser, error) {
	return m.listForUser(ctx, userID)
}

func (m *mockTeamService) CreateTeam(ctx context.Context, userID, name string, icon, iconBg, iconFg *string) (*gen.TeamForUser, error) {
	return m.createTeam(ctx, userID, name, icon, iconBg, iconFg)
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

func (m *mockTeamService) AcceptInvite(ctx context.Context, code, userID string) (*gen.AcceptInviteResponse, error) {
	return m.acceptInvite(ctx, code, userID)
}

func (m *mockTeamService) GetTeamPhotoURL(ctx context.Context, teamID string) (string, error) {
	return m.getTeamPhotoURL(ctx, teamID)
}

func (m *mockTeamService) UpdatePhoto(ctx context.Context, teamID string, data []byte, mimeType string) (*gen.Team, error) {
	return m.updatePhoto(ctx, teamID, data, mimeType)
}

func (m *mockTeamService) DeletePhoto(ctx context.Context, teamID string) error {
	return m.deletePhoto(ctx, teamID)
}

func (m *mockTeamService) GetTeamLogoURL(ctx context.Context, teamID string) (string, error) {
	return m.getTeamLogoURL(ctx, teamID)
}

func (m *mockTeamService) UpdateLogo(ctx context.Context, teamID string, data []byte, mimeType string) (*gen.Team, error) {
	return m.updateLogo(ctx, teamID, data, mimeType)
}

func (m *mockTeamService) DeleteLogo(ctx context.Context, teamID string) error {
	return m.deleteLogo(ctx, teamID)
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

func (f *fakeAuthSvc) GetMyPhotoURL(_ context.Context, _ string) (string, error) {
	return "", pgx.ErrNoRows
}
func (f *fakeAuthSvc) EraseAccount(_ context.Context, _, _ string) error { return nil }
func (f *fakeAuthSvc) ExportUserData(_ context.Context, _ string) (*auth.ExportData, error) {
	return &auth.ExportData{}, nil
}

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
	c, err := auth.NewSessionCookieCodec([][]byte{make([]byte, 32)}, false, time.Hour, "")
	if err != nil {
		panic(err)
	}
	return c
}()

// sessionCookie builds an encrypted session cookie carrying a fixed test JWT
// (fakeAuthSvc.ValidateToken ignores the token value, so every caller in this
// file passes the same placeholder).
func sessionCookie() *http.Cookie {
	value, err := testCodec.Encrypt("test-token")
	if err != nil {
		panic(err)
	}
	return &http.Cookie{Name: testCodec.Name(), Value: value}
}

// withAuthUser wraps a handler with auth middleware using a fake user.
func withAuthUser(h http.Handler, user *auth.UserRow) http.Handler {
	logger := slog.Default()
	authH := auth.NewHandler(&fakeAuthSvc{user: user}, logger, testCodec, nil)
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

	h := teams.NewHandler(svc, slog.Default(), nil)

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
	req.AddCookie(sessionCookie())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []gen.TeamForUser
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "Test Team", result[0].Name)
	assert.Equal(t, 5, result[0].MemberCount)
}

func TestTeamHandler_GetTeamLogo_NotFound(t *testing.T) {
	t.Parallel()

	svc := &mockTeamService{
		getTeamLogoURL: func(_ context.Context, _ string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}
	h := teams.NewHandler(svc, slog.Default(), nil)

	resp, err := h.GetTeamLogo(context.Background(), gen.GetTeamLogoRequestObject{TeamId: uuid.New()})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetTeamLogoResponse(w))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTeamHandler_GetTeamLogo_RedirectsToPresignedURL(t *testing.T) {
	t.Parallel()

	svc := &mockTeamService{
		getTeamLogoURL: func(_ context.Context, _ string) (string, error) {
			return "https://s3.example.com/bucket/teams/t1/logo?sig=abc", nil
		},
	}
	h := teams.NewHandler(svc, slog.Default(), nil)

	resp, err := h.GetTeamLogo(context.Background(), gen.GetTeamLogoRequestObject{TeamId: uuid.New()})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetTeamLogoResponse(w))
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "teams/t1/logo")
}

func TestTeamHandler_UploadTeamLogo_StoresAndReturnsTeam(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	hasLogo := true
	svc := &mockTeamService{
		updateLogo: func(_ context.Context, tid string, data []byte, mime string) (*gen.Team, error) {
			assert.Equal(t, teamID.String(), tid)
			assert.Equal(t, "image/jpeg", mime)
			assert.NotEmpty(t, data)
			return &gen.Team{Id: teamID, Name: "Test Team", HasLogo: &hasLogo}, nil
		},
	}
	h := teams.NewHandler(svc, slog.Default(), nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		require.NoError(t, err)
		mr := multipart.NewReader(r.Body, params["boundary"])
		resp, err := h.UploadTeamLogo(r.Context(), gen.UploadTeamLogoRequestObject{TeamId: teamID, Body: mr})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = resp.VisitUploadTeamLogoResponse(w)
	})
	handler := withAuthUser(inner, testAuthUser())

	// Minimal valid JPEG (SOI + APP0 marker) so http.DetectContentType sees image/jpeg.
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("logo", "logo.jpg")
	require.NoError(t, err)
	_, err = fw.Write(jpegData)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/teams/"+teamID.String()+"/logo", &buf)
	req.AddCookie(sessionCookie())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Regression test: io.LimitReader silently truncates rather than erroring
// once its cap is reached, so a photo/logo between 2 MB and the global
// body-size cap used to sail past the "cannot read file data" check with
// intact JPEG magic bytes but a truncated body -- decoding it downstream
// failed with a plain error (not ErrImageTooLarge), falling through to a
// generic 500 instead of the 413 openapi.yaml documents for this endpoint.
// Routes the error through apierror.ResponseErrorHandler (like the real
// strict-server dispatch does) rather than a hardcoded status, since a
// hardcoded one would mask exactly this kind of wrong-status-code bug.
func TestTeamHandler_UploadTeamLogo_RejectsOversizedFile(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		updateLogo: func(context.Context, string, []byte, string) (*gen.Team, error) {
			t.Fatal("service must not be called for an oversized upload")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.Default(), nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		require.NoError(t, err)
		mr := multipart.NewReader(r.Body, params["boundary"])
		resp, err := h.UploadTeamLogo(r.Context(), gen.UploadTeamLogoRequestObject{TeamId: teamID, Body: mr})
		if err != nil {
			apierror.ResponseErrorHandler(slog.Default())(w, r, err)
			return
		}
		_ = resp.VisitUploadTeamLogoResponse(w)
	})
	handler := withAuthUser(inner, testAuthUser())

	// JPEG magic bytes followed by > 2 MB of filler so DetectContentType
	// still identifies it as a JPEG, but the file exceeds the 2 MB cap.
	oversized := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 3<<20)...)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("logo", "huge.jpg")
	require.NoError(t, err)
	_, err = fw.Write(oversized)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/teams/"+teamID.String()+"/logo", &buf)
	req.AddCookie(sessionCookie())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestTeamHandler_UpdateTeam_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		updateTeam: func(_ context.Context, _ string, _ teams.TeamPatch) (*gen.Team, error) {
			return &gen.Team{Id: teamID, Name: "Renamed Team"}, nil
		},
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	newName := "Renamed Team"
	body := &gen.UpdateTeamJSONRequestBody{Name: &newName}
	_, err := h.UpdateTeam(ctx, gen.UpdateTeamRequestObject{TeamId: teamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.update", rec["event"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

// Regression test: UploadTeamPhoto/UploadTeamLogo/DeleteTeamPhoto/
// DeleteTeamLogo used to be the only settings-gated team mutations with no
// audit trail at all, unlike UpdateTeam/CreateTeam/CreateInvite/AcceptInvite
// which all emit one. A settings:write holder could silently replace or
// remove a team's public-facing branding with no compliance-log trace.
func TestTeamHandler_UploadTeamPhoto_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	hasPhoto := true
	svc := &mockTeamService{
		updatePhoto: func(_ context.Context, tid string, _ []byte, _ string) (*gen.Team, error) {
			return &gen.Team{Id: teamID, Name: "Test Team", HasPhoto: &hasPhoto}, nil
		},
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		require.NoError(t, err)
		mr := multipart.NewReader(r.Body, params["boundary"])
		resp, err := h.UploadTeamPhoto(r.Context(), gen.UploadTeamPhotoRequestObject{TeamId: teamID, Body: mr})
		require.NoError(t, err)
		_ = resp.VisitUploadTeamPhotoResponse(w)
	})
	handler := withAuthUser(inner, testAuthUser())

	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("photo", "photo.jpg")
	require.NoError(t, err)
	_, err = fw.Write(jpegData)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/teams/"+teamID.String()+"/photo", &body)
	req.AddCookie(sessionCookie())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.branding_update", rec["event"])
	assert.Equal(t, "photo.upload", rec["operation"])
	assert.Equal(t, testAuthUser().Id.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

func TestTeamHandler_UploadTeamLogo_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	hasLogo := true
	svc := &mockTeamService{
		updateLogo: func(_ context.Context, tid string, _ []byte, _ string) (*gen.Team, error) {
			return &gen.Team{Id: teamID, Name: "Test Team", HasLogo: &hasLogo}, nil
		},
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		require.NoError(t, err)
		mr := multipart.NewReader(r.Body, params["boundary"])
		resp, err := h.UploadTeamLogo(r.Context(), gen.UploadTeamLogoRequestObject{TeamId: teamID, Body: mr})
		require.NoError(t, err)
		_ = resp.VisitUploadTeamLogoResponse(w)
	})
	handler := withAuthUser(inner, testAuthUser())

	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("logo", "logo.jpg")
	require.NoError(t, err)
	_, err = fw.Write(jpegData)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/teams/"+teamID.String()+"/logo", &body)
	req.AddCookie(sessionCookie())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.branding_update", rec["event"])
	assert.Equal(t, "logo.upload", rec["operation"])
	assert.Equal(t, testAuthUser().Id.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

func TestTeamHandler_DeleteTeamPhoto_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		deletePhoto: func(_ context.Context, _ string) error { return nil },
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	_, err := h.DeleteTeamPhoto(ctx, gen.DeleteTeamPhotoRequestObject{TeamId: teamID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.branding_update", rec["event"])
	assert.Equal(t, "photo.delete", rec["operation"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

func TestTeamHandler_DeleteTeamLogo_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		deleteLogo: func(_ context.Context, _ string) error { return nil },
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	_, err := h.DeleteTeamLogo(ctx, gen.DeleteTeamLogoRequestObject{TeamId: teamID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.branding_update", rec["event"])
	assert.Equal(t, "logo.delete", rec["operation"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

// Regression test: CreateTeam mints a new Admin role with full write
// permissions and assigns it to the caller -- more privilege than
// UpdateTeam/CreateInvite/AcceptInvite grant, all three of which already
// emit an audit record. CreateTeam used to be the one team-mutating handler
// with no audit trail at all.
func TestTeamHandler_CreateTeam_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		createTeam: func(_ context.Context, _, _ string, _, _, _ *string) (*gen.TeamForUser, error) {
			return &gen.TeamForUser{Id: teamID, Name: "New Team"}, nil
		},
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	body := &gen.CreateTeamJSONRequestBody{Name: "New Team"}
	_, err := h.CreateTeam(ctx, gen.CreateTeamRequestObject{Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.create", rec["event"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

func TestTeamHandler_CreateTeam_RejectsEmptyName_Returns400(t *testing.T) {
	t.Parallel()

	svc := &mockTeamService{
		createTeam: func(context.Context, string, string, *string, *string, *string) (*gen.TeamForUser, error) {
			t.Fatal("service must not be called when validation fails")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	body := &gen.CreateTeamJSONRequestBody{Name: ""}
	_, err := h.CreateTeam(ctx, gen.CreateTeamRequestObject{Body: body})

	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok, "expected *apierror.APIError, got %T (%v) — invalid input must not fall through to the generic 500", err, err)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestTeamHandler_UpdateTeam_RejectsEmptyName(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		updateTeam: func(_ context.Context, _ string, _ teams.TeamPatch) (*gen.Team, error) {
			t.Fatal("service must not be called when validation fails")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	empty := ""
	body := &gen.UpdateTeamJSONRequestBody{Name: &empty}
	_, err := h.UpdateTeam(ctx, gen.UpdateTeamRequestObject{TeamId: teamID, Body: body})
	require.Error(t, err)
}

func TestTeamHandler_UpdateTeam_RejectsTooManyReasonVisibilityRoleIds(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		updateTeam: func(_ context.Context, _ string, _ teams.TeamPatch) (*gen.Team, error) {
			t.Fatal("service must not be called when validation fails")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	roleIDs := make([]uuid.UUID, 201)
	for i := range roleIDs {
		roleIDs[i] = uuid.New()
	}
	body := &gen.UpdateTeamJSONRequestBody{ReasonVisibilityRoleIds: &roleIDs}
	_, err := h.UpdateTeam(ctx, gen.UpdateTeamRequestObject{TeamId: teamID, Body: body})
	require.Error(t, err)
}

func TestTeamHandler_UpdateTeam_RejectsOversizedShort(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		updateTeam: func(_ context.Context, _ string, _ teams.TeamPatch) (*gen.Team, error) {
			t.Fatal("service must not be called when validation fails")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
	oversized := strings.Repeat("x", 51)
	body := &gen.UpdateTeamJSONRequestBody{Short: &oversized}
	_, err := h.UpdateTeam(ctx, gen.UpdateTeamRequestObject{TeamId: teamID, Body: body})
	require.Error(t, err)
}

func TestTeamHandler_CreateInvite_EmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	inviteID := uuid.New()
	svc := &mockTeamService{
		createInvite: func(_ context.Context, _ string) (*gen.Invite, error) {
			return &gen.Invite{Id: inviteID, TeamId: teamID, Code: "ABC123"}, nil
		},
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Admin", Email: "a@x.c"})
	_, err := h.CreateInvite(ctx, gen.CreateInviteRequestObject{TeamId: teamID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.invite_create", rec["event"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
	assert.Equal(t, inviteID.String(), rec["inviteId"])
}

func TestTeamHandler_AcceptInvite_RequiresAuthentication(t *testing.T) {
	t.Parallel()

	svc := &mockTeamService{
		acceptInvite: func(_ context.Context, _, _ string) (*gen.AcceptInviteResponse, error) {
			t.Fatal("service must not be called when unauthenticated")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	_, err := h.AcceptInvite(context.Background(), gen.AcceptInviteRequestObject{Code: "ABC123"})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Status)
}

func TestTeamHandler_AcceptInvite_ReturnsTeamAndEmitsAuditEvent(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		acceptInvite: func(_ context.Context, code, userID string) (*gen.AcceptInviteResponse, error) {
			assert.Equal(t, "ABC123", code)
			return &gen.AcceptInviteResponse{Id: teamID, Name: "Joined Team"}, nil
		},
	}
	var buf bytes.Buffer
	h := teams.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	actorID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: actorID, Name: "Joiner", Email: "j@x.c"})
	resp, err := h.AcceptInvite(ctx, gen.AcceptInviteRequestObject{Code: "ABC123"})
	require.NoError(t, err)
	require.IsType(t, gen.AcceptInvite200JSONResponse{}, resp)
	assert.Equal(t, teamID, gen.AcceptInviteResponse(resp.(gen.AcceptInvite200JSONResponse)).Id)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "team.invite_accept", rec["event"])
	assert.Equal(t, actorID.String(), rec["actor"])
	assert.Equal(t, teamID.String(), rec["teamId"])
}

// Regression test: the response used to have no way to signal "you were
// already a member," so the frontend showed a misleading "joined" toast on
// every repeat visit to an old invite link, not just the first.
func TestTeamHandler_AcceptInvite_PropagatesAlreadyMemberFlag(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := &mockTeamService{
		acceptInvite: func(_ context.Context, _, _ string) (*gen.AcceptInviteResponse, error) {
			return &gen.AcceptInviteResponse{Id: teamID, Name: "Existing Team", AlreadyMember: true}, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Joiner", Email: "j@x.c"})
	resp, err := h.AcceptInvite(ctx, gen.AcceptInviteRequestObject{Code: "ABC123"})
	require.NoError(t, err)
	require.IsType(t, gen.AcceptInvite200JSONResponse{}, resp)
	assert.True(t, gen.AcceptInviteResponse(resp.(gen.AcceptInvite200JSONResponse)).AlreadyMember)
}

func TestTeamHandler_AcceptInvite_RejectsOverlongCode(t *testing.T) {
	t.Parallel()

	svc := &mockTeamService{
		acceptInvite: func(_ context.Context, _, _ string) (*gen.AcceptInviteResponse, error) {
			t.Fatal("service must not be called when the code fails validation")
			return nil, nil
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Joiner", Email: "j@x.c"})
	_, err := h.AcceptInvite(ctx, gen.AcceptInviteRequestObject{Code: strings.Repeat("a", 65)})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestTeamHandler_AcceptInvite_InviteNotFound_Returns404(t *testing.T) {
	t.Parallel()

	svc := &mockTeamService{
		acceptInvite: func(_ context.Context, _, _ string) (*gen.AcceptInviteResponse, error) {
			return nil, teams.ErrInviteNotFound
		},
	}
	h := teams.NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Joiner", Email: "j@x.c"})
	resp, err := h.AcceptInvite(ctx, gen.AcceptInviteRequestObject{Code: "does-not-exist"})
	require.NoError(t, err)
	require.IsType(t, gen.AcceptInvite404ApplicationProblemPlusJSONResponse{}, resp)
}
