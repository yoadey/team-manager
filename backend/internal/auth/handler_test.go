package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockAuthService struct {
	login          func(ctx context.Context, email, password string) (string, *auth.UserRow, error)
	validateToken  func(ctx context.Context, token string) (*auth.UserRow, error)
	logout         func(ctx context.Context, tokenHash string) error
	updatePhoto    func(ctx context.Context, userID string, data []byte, mime string) (*auth.UserRow, error)
	eraseAccount   func(ctx context.Context, userID, password string) error
	exportUserData func(ctx context.Context, userID string) (*auth.ExportData, error)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, *auth.UserRow, error) {
	return m.login(ctx, email, password)
}

func (m *mockAuthService) ValidateToken(ctx context.Context, token string) (*auth.UserRow, error) {
	return m.validateToken(ctx, token)
}

func (m *mockAuthService) Logout(ctx context.Context, tokenHash string) error {
	return m.logout(ctx, tokenHash)
}

func (m *mockAuthService) UpdatePhoto(ctx context.Context, userID string, data []byte, mime string) (*auth.UserRow, error) {
	return m.updatePhoto(ctx, userID, data, mime)
}

func (m *mockAuthService) EraseAccount(ctx context.Context, userID, password string) error {
	return m.eraseAccount(ctx, userID, password)
}

func (m *mockAuthService) ExportUserData(ctx context.Context, userID string) (*auth.ExportData, error) {
	if m.exportUserData != nil {
		return m.exportUserData(ctx, userID)
	}
	return &auth.ExportData{}, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func testUser() *auth.UserRow {
	return &auth.UserRow{
		Id:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Name:        "Test User",
		Email:       "test@example.com",
		AvatarColor: "#6366f1",
		CreatedAt:   time.Now(),
	}
}

// testCodec builds a session cookie codec with a fixed all-zero key for tests.
func testCodec(t *testing.T) *auth.SessionCookieCodec {
	t.Helper()
	codec, err := auth.NewSessionCookieCodec([][]byte{make([]byte, 32)}, false, time.Hour, "")
	require.NoError(t, err)
	return codec
}

// addSessionCookie encrypts jwt with the codec and attaches it as the session cookie.
func addSessionCookie(t *testing.T, codec *auth.SessionCookieCodec, req *http.Request, jwt string) {
	t.Helper()
	value, err := codec.Encrypt(jwt)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: codec.Name(), Value: value})
}

// callListProviders invokes the handler method and writes the response.
func callListProviders(h *auth.Handler, w http.ResponseWriter, r *http.Request) {
	resp, err := h.ListProviders(r.Context(), gen.ListProvidersRequestObject{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = resp.VisitListProvidersResponse(w)
}

// callLogin invokes the handler Login method.
func callLogin(h *auth.Handler, w http.ResponseWriter, r *http.Request) {
	var body gen.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.Login(r.Context(), gen.LoginRequestObject{Body: &body})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = resp.VisitLoginResponse(w)
}

// callGetCurrentUser invokes the handler GetCurrentUser method.
func callGetCurrentUser(h *auth.Handler, w http.ResponseWriter, r *http.Request) {
	resp, err := h.GetCurrentUser(r.Context(), gen.GetCurrentUserRequestObject{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = resp.VisitGetCurrentUserResponse(w)
}

// callLogout invokes the handler Logout method.
func callLogout(h *auth.Handler, w http.ResponseWriter, r *http.Request) {
	resp, err := h.Logout(r.Context(), gen.LogoutRequestObject{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = resp.VisitLogoutResponse(w)
}

// callUploadMyPhoto invokes the handler UploadMyPhoto method, routing any
// returned error through apierror.ResponseErrorHandler exactly like the real
// strict-server dispatch in cmd/server/main.go does. A hardcoded
// http.Error(..., http.StatusBadRequest) here previously masked a real bug:
// every UploadMyPhoto error path returned a bespoke error type that
// ResponseErrorHandler's errors.As(*apierror.APIError) didn't recognize, so
// production actually served a 500 for all of them, but the old test helper
// couldn't observe that because it forced 400 regardless of the real status.
func callUploadMyPhoto(h *auth.Handler, w http.ResponseWriter, r *http.Request) {
	mr := multipart.NewReader(r.Body, extractBoundary(r.Header.Get("Content-Type")))
	resp, err := h.UploadMyPhoto(r.Context(), gen.UploadMyPhotoRequestObject{Body: mr})
	if err != nil {
		apierror.ResponseErrorHandler(slog.Default())(w, r, err)
		return
	}
	_ = resp.VisitUploadMyPhotoResponse(w)
}

// extractBoundary pulls the boundary parameter out of a Content-Type value.
func extractBoundary(ct string) string {
	const prefix = "boundary="
	for _, part := range splitSemicolon(ct) {
		if len(part) > len(prefix) && part[:len(prefix)] == prefix {
			return part[len(prefix):]
		}
	}
	return ""
}

func splitSemicolon(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			out = append(out, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, trimSpace(s[start:]))
	return out
}

func trimSpace(s string) string {
	for s != "" && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for s != "" && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestHandler_ListProviders(t *testing.T) {
	t.Parallel()

	h := auth.NewHandler(&mockAuthService{}, slog.Default(), nil, nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/providers", http.NoBody)
	w := httptest.NewRecorder()
	callListProviders(h, w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var providers []gen.Provider
	require.NoError(t, json.NewDecoder(w.Body).Decode(&providers))
	require.Len(t, providers, 1)
	assert.Equal(t, "password", providers[0].Id)
}

func TestHandler_Login_Success(t *testing.T) {
	t.Parallel()

	user := testUser()
	svc := &mockAuthService{
		login: func(_ context.Context, email, password string) (string, *auth.UserRow, error) {
			assert.Equal(t, "test@example.com", email)
			assert.Equal(t, "Secret123!", password)
			return "jwt.token.here", user, nil
		},
	}
	h := auth.NewHandler(svc, slog.Default(), nil, nil)

	body := `{"email":"test@example.com","password":"Secret123!"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	callLogin(h, w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp gen.LoginResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "jwt.token.here", resp.Token)
	assert.Equal(t, "Test User", resp.User.Name)
}

func TestHandler_Login_BadCredentials(t *testing.T) {
	t.Parallel()

	svc := &mockAuthService{
		login: func(_ context.Context, _, _ string) (string, *auth.UserRow, error) {
			return "", nil, errors.New("invalid credentials")
		},
	}
	h := auth.NewHandler(svc, slog.Default(), nil, nil)

	body := `{"email":"bad@example.com","password":"wrong"}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	callLogin(h, w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "problem+json")
}

func TestHandler_GetCurrentUser_NoAuth(t *testing.T) {
	t.Parallel()

	h := auth.NewHandler(&mockAuthService{}, slog.Default(), nil, nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", http.NoBody)
	// No user in context — simulates unauthenticated request.
	w := httptest.NewRecorder()
	callGetCurrentUser(h, w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetCurrentUser_WithAuth(t *testing.T) {
	t.Parallel()

	user := testUser()
	svc := &mockAuthService{
		validateToken: func(_ context.Context, token string) (*auth.UserRow, error) {
			if token == "valid-token" {
				return user, nil
			}
			return nil, errors.New("bad token")
		},
	}

	codec := testCodec(t)
	h := auth.NewHandler(svc, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/auth/me", func(w http.ResponseWriter, req *http.Request) {
			callGetCurrentUser(h, w, req)
		})
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", http.NoBody)
	addSessionCookie(t, codec, req, "valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp gen.User
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "Test User", resp.Name)
}

func TestHandler_Logout(t *testing.T) {
	t.Parallel()

	user := testUser()
	logoutCalled := false

	svc := &mockAuthService{
		validateToken: func(_ context.Context, token string) (*auth.UserRow, error) {
			if token == "valid-logout-token" {
				return user, nil
			}
			return nil, errors.New("bad token")
		},
		logout: func(_ context.Context, tokenHash string) error {
			logoutCalled = true
			assert.NotEmpty(t, tokenHash)
			return nil
		},
	}

	codec := testCodec(t)
	h := auth.NewHandler(svc, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Post("/auth/logout", func(w http.ResponseWriter, req *http.Request) {
			callLogout(h, w, req)
		})
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout", http.NoBody)
	addSessionCookie(t, codec, req, "valid-logout-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, logoutCalled, "Logout should have been called on the service")
}

func TestHandler_GetCurrentUser_InvalidToken(t *testing.T) {
	t.Parallel()

	svc := &mockAuthService{
		validateToken: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return nil, errors.New("invalid token")
		},
	}

	codec := testCodec(t)
	h := auth.NewHandler(svc, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/auth/me", func(w http.ResponseWriter, req *http.Request) {
			callGetCurrentUser(h, w, req)
		})
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", http.NoBody)
	addSessionCookie(t, codec, req, "bad-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AuthMiddleware_MissingCookie(t *testing.T) {
	t.Parallel()

	codec := testCodec(t)
	h := auth.NewHandler(&mockAuthService{}, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/auth/me", func(_ http.ResponseWriter, _ *http.Request) {
			t.Fatal("inner handler must not run without a session cookie")
		})
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AuthMiddleware_TamperedCookie(t *testing.T) {
	t.Parallel()

	codec := testCodec(t)
	h := auth.NewHandler(&mockAuthService{}, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/auth/me", func(_ http.ResponseWriter, _ *http.Request) {
			t.Fatal("inner handler must not run with a tampered session cookie")
		})
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", http.NoBody)
	req.AddCookie(&http.Cookie{Name: codec.Name(), Value: "not-a-valid-encrypted-value"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UploadMyPhoto(t *testing.T) {
	t.Parallel()

	user := testUser()
	updatedUser := *user
	updatedUser.PhotoData = []byte("fake-jpeg")
	updatedUser.PhotoMime = "image/jpeg"

	svc := &mockAuthService{
		validateToken: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return user, nil
		},
		updatePhoto: func(_ context.Context, userID string, _ []byte, _ string) (*auth.UserRow, error) {
			assert.Equal(t, user.Id.String(), userID)
			return &updatedUser, nil
		},
	}

	codec := testCodec(t)
	h := auth.NewHandler(svc, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Put("/auth/me/photo", func(w http.ResponseWriter, req *http.Request) {
			callUploadMyPhoto(h, w, req)
		})
	})

	// Build a multipart body with a minimal valid JPEG (SOI + EOF markers).
	// http.DetectContentType requires the JPEG magic bytes (\xFF\xD8\xFF) to
	// identify the content as image/jpeg.
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "avatar.jpg")
	require.NoError(t, err)
	_, err = fw.Write(jpegData)
	require.NoError(t, err)
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/auth/me/photo", &buf)
	addSessionCookie(t, codec, req, "token")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Regression test: UploadMyPhoto's error paths (errUnauthorized/errBadRequest/
// errInternal) used to return a bespoke *handlerError type that
// apierror.ResponseErrorHandler's errors.As(*apierror.APIError) never
// matched, so every one of these fell through to a generic 500 in
// production regardless of the intended status. They must now return an
// *apierror.APIError so the real status/detail reaches the client.
func TestHandler_UploadMyPhoto_RejectsNonImageContent(t *testing.T) {
	t.Parallel()

	svc := &mockAuthService{
		validateToken: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return testUser(), nil
		},
	}
	codec := testCodec(t)
	h := auth.NewHandler(svc, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Put("/auth/me/photo", func(w http.ResponseWriter, req *http.Request) {
			callUploadMyPhoto(h, w, req)
		})
	})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "not-an-image.txt")
	require.NoError(t, err)
	_, err = fw.Write([]byte("plain text, not an image"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/auth/me/photo", &buf)
	addSessionCookie(t, codec, req, "token")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "JPEG")
}

// Regression test: io.LimitReader silently truncates rather than erroring
// once its cap is reached, so a photo between 2 MB and the global body-size
// cap used to sail past the "cannot read file data" check with intact JPEG
// magic bytes but truncated body -- decoding it downstream failed with a
// plain error (not ErrImageTooLarge), falling through to a generic 500
// instead of a clean 413 for what's really an oversized-payload problem.
func TestHandler_UploadMyPhoto_RejectsOversizedFile(t *testing.T) {
	t.Parallel()

	svc := &mockAuthService{
		validateToken: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return testUser(), nil
		},
		updatePhoto: func(context.Context, string, []byte, string) (*auth.UserRow, error) {
			t.Fatal("service must not be called for an oversized upload")
			return nil, nil
		},
	}
	codec := testCodec(t)
	h := auth.NewHandler(svc, slog.Default(), codec, nil)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Put("/auth/me/photo", func(w http.ResponseWriter, req *http.Request) {
			callUploadMyPhoto(h, w, req)
		})
	})

	// JPEG magic bytes followed by > 2 MB of filler so DetectContentType
	// still identifies it as a JPEG, but the file exceeds the 2 MB cap.
	oversized := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 3<<20)...)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "huge.jpg")
	require.NoError(t, err)
	_, err = fw.Write(oversized)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/auth/me/photo", &buf)
	addSessionCookie(t, codec, req, "token")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

// Regression test: GetMyPhoto's 404 used to be hand-built with no `type`
// field at all (the only error response in this package that shipped
// without one), instead of going through apierror.NotFound like every other
// error path.
func TestHandler_GetMyPhoto_NoPhoto_Returns404WithType(t *testing.T) {
	t.Parallel()

	h := auth.NewHandler(&mockAuthService{}, slog.Default(), nil, nil)
	ctx := auth.ContextWithUser(context.Background(), testUser())
	resp, err := h.GetMyPhoto(ctx, gen.GetMyPhotoRequestObject{})
	require.Error(t, err)
	require.Nil(t, resp)

	w := httptest.NewRecorder()
	apierror.ResponseErrorHandler(slog.Default())(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me/photo", http.NoBody), err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotEmpty(t, body["type"])
}

func TestHandler_GetMyDataExport(t *testing.T) {
	t.Parallel()

	svc := &mockAuthService{
		exportUserData: func(_ context.Context, userID string) (*auth.ExportData, error) {
			return &auth.ExportData{
				Profile: auth.ExportProfile{ID: userID, Name: "Test User", Email: "test@example.com"},
			}, nil
		},
	}
	h := auth.NewHandler(svc, slog.Default(), nil, nil)

	ctx := auth.ContextWithUser(context.Background(), testUser())
	resp, err := h.GetMyDataExport(ctx, gen.GetMyDataExportRequestObject{})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetMyDataExportResponse(w))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".json")

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	profile, ok := body["profile"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test@example.com", profile["email"])
}

func TestHandler_GetMyDataExport_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := auth.NewHandler(&mockAuthService{}, slog.Default(), nil, nil)
	resp, err := h.GetMyDataExport(context.Background(), gen.GetMyDataExportRequestObject{})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetMyDataExportResponse(w))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
