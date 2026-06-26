package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// authService is the interface the Handler relies on.
type authService interface {
	Login(ctx context.Context, email, password string) (token string, user *UserRow, err error)
	ValidateToken(ctx context.Context, tokenString string) (*UserRow, error)
	Logout(ctx context.Context, tokenHash string) error
	UpdatePhoto(ctx context.Context, userID string, data []byte, mime string) (*UserRow, error)
}

// Handler implements the auth-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    authService
	logger *slog.Logger
	codec  *SessionCookieCodec
}

// NewHandler creates a new Handler. The codec is used by AuthMiddleware to read
// the encrypted session cookie.
func NewHandler(svc authService, logger *slog.Logger, codec *SessionCookieCodec) *Handler {
	return &Handler{svc: svc, logger: logger, codec: codec}
}

// ListProviders returns the list of supported login providers (hardcoded to password).
func (h *Handler) ListProviders(ctx context.Context, _ gen.ListProvidersRequestObject) (gen.ListProvidersResponseObject, error) {
	border := "#e2e8f0"
	return gen.ListProviders200JSONResponse([]gen.Provider{
		{
			Id:     "password",
			Name:   "Email & Password",
			Sub:    "Sign in with your email address",
			Glyph:  "lock",
			Bg:     "#ffffff",
			Fg:     "#1e293b",
			Border: &border,
		},
	}), nil
}

// Login authenticates a user with email + password and returns a JWT token.
func (h *Handler) Login(ctx context.Context, request gen.LoginRequestObject) (gen.LoginResponseObject, error) {
	if request.Body == nil {
		return gen.Login401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("missing request body"),
		}, nil
	}

	if err := validate.Email(string(request.Body.Email)); err != nil {
		return gen.Login401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("invalid credentials"),
		}, nil
	}
	if err := validate.PasswordStrength(request.Body.Password); err != nil {
		return gen.Login401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("invalid credentials"),
		}, nil
	}

	token, user, err := h.svc.Login(ctx, string(request.Body.Email), request.Body.Password)
	if err != nil {
		h.logger.WarnContext(ctx, "login failed", "email", request.Body.Email, "err", err)
		return gen.Login401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("invalid credentials"),
		}, nil
	}

	return gen.Login200JSONResponse{
		Token: token,
		User:  toGenUser(user),
	}, nil
}

// GetCurrentUser returns the authenticated user's profile.
func (h *Handler) GetCurrentUser(ctx context.Context, _ gen.GetCurrentUserRequestObject) (gen.GetCurrentUserResponseObject, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return gen.GetCurrentUser401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("not authenticated"),
		}, nil
	}
	return gen.GetCurrentUser200JSONResponse(toGenUser(user)), nil
}

// GetMyPhoto returns the authenticated user's profile photo.
func (h *Handler) GetMyPhoto(ctx context.Context, _ gen.GetMyPhotoRequestObject) (gen.GetMyPhotoResponseObject, error) {
	user, ok := UserFromContext(ctx)
	if !ok || len(user.PhotoData) == 0 {
		title := "Not Found"
		detail := "no profile photo"
		status := 404
		return gen.GetMyPhoto404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
				Title:  &title,
				Detail: &detail,
				Status: &status,
			},
		}, nil
	}
	return gen.GetMyPhoto200ImagejpegResponse{
		Body:          bytes.NewReader(user.PhotoData),
		ContentLength: int64(len(user.PhotoData)),
	}, nil
}

// UploadMyPhoto handles profile photo upload (multipart), resizes, stores, and returns updated user.
func (h *Handler) UploadMyPhoto(ctx context.Context, request gen.UploadMyPhotoRequestObject) (gen.UploadMyPhotoResponseObject, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		// Return a 401 via a plain error so the middleware above catches it,
		// but here we map it to an UploadMyPhoto200 to stay within the response type.
		// Better: the middleware already blocks unauthenticated requests.
		return nil, errUnauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, errBadRequest("missing multipart body")
	}

	part, err := request.Body.NextPart()
	if err != nil {
		h.logger.WarnContext(ctx, "UploadMyPhoto: read multipart failed", "err", err)
		return nil, errBadRequest("cannot read multipart body")
	}
	defer func() {
		if err := part.Close(); err != nil {
			h.logger.ErrorContext(ctx, "close multipart part", "err", err)
		}
	}()

	data, err := io.ReadAll(io.LimitReader(part, 2<<20)) // 2 MB max
	if err != nil {
		h.logger.WarnContext(ctx, "UploadMyPhoto: read file data failed", "err", err)
		return nil, errBadRequest("cannot read file data")
	}

	// Detect MIME from actual content; reject anything other than JPEG/PNG.
	ct := http.DetectContentType(data)
	if ct != "image/jpeg" && ct != "image/png" {
		return nil, errBadRequest("only JPEG and PNG images are accepted")
	}

	updated, err := h.svc.UpdatePhoto(ctx, user.Id.String(), data, ct)
	if err != nil {
		h.logger.ErrorContext(ctx, "update photo failed", "err", err)
		return nil, errInternal("photo update failed")
	}

	return gen.UploadMyPhoto200JSONResponse(toGenUser(updated)), nil
}

// Logout invalidates the current session.
func (h *Handler) Logout(ctx context.Context, _ gen.LogoutRequestObject) (gen.LogoutResponseObject, error) {
	// The raw token is stored in context by AuthMiddleware.
	rawToken, _ := ctx.Value(rawBearerContextKey).(string)
	tokenHash := sha256Hex(rawToken)
	if err := h.svc.Logout(ctx, tokenHash); err != nil {
		h.logger.WarnContext(ctx, "logout failed", "err", err)
	}
	return gen.Logout204Response{}, nil
}

// ─── Middleware ──────────────────────────────────────────────────────────────

// rawBearerContextKey is used internally to pass the raw Bearer token through context.
const rawBearerContextKey contextKey = "auth_raw_token" //nolint:gosec // not a credential

// AuthMiddleware reads and validates the encrypted session cookie. The
// decrypted JWT is validated against the auth service; the raw JWT is stored in
// context so Logout can revoke the session. Unauthenticated requests receive a
// 401 Problem Details response.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.codec.Name())
		if err != nil || cookie.Value == "" {
			writeUnauthorized(w, "missing session cookie")
			return
		}
		rawToken, err := h.codec.Decrypt(cookie.Value)
		if err != nil {
			h.logger.WarnContext(r.Context(), "session cookie decrypt failed", "err", err)
			writeUnauthorized(w, "invalid session cookie")
			return
		}

		user, err := h.svc.ValidateToken(r.Context(), rawToken)
		if err != nil {
			h.logger.WarnContext(r.Context(), "token validation failed", "err", err)
			writeUnauthorized(w, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, rawBearerContextKey, rawToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext retrieves the authenticated *UserRow from the request context.
func UserFromContext(ctx context.Context) (*UserRow, bool) {
	u, ok := ctx.Value(userContextKey).(*UserRow)
	return u, ok && u != nil
}

// ContextWithUser stores a *UserRow in ctx. Used in tests to simulate authenticated requests.
func ContextWithUser(ctx context.Context, user *UserRow) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// ─── helpers ────────────────────────────────────────────────────────────────

// toGenUser maps an internal UserRow to the generated gen.User type.
func toGenUser(u *UserRow) gen.User {
	hasPhoto := len(u.PhotoData) > 0
	gu := gen.User{
		Id:          u.Id,
		Name:        u.Name,
		Email:       openapi_types.Email(u.Email),
		AvatarColor: u.AvatarColor,
		HasPhoto:    &hasPhoto,
	}
	if u.Phone != nil {
		gu.Phone = u.Phone
	}
	if u.Address != nil {
		gu.Address = u.Address
	}
	if u.Birthday != nil {
		d := openapi_types.Date{Time: *u.Birthday}
		gu.Birthday = &d
	}
	return gu
}

// unauthorized builds an UnauthorizedApplicationProblemPlusJSONResponse.
func unauthorized(detail string) gen.UnauthorizedApplicationProblemPlusJSONResponse {
	title := "Unauthorized"
	status := 401
	typeStr := "https://teammanager.example/errors/unauthorized"
	return gen.UnauthorizedApplicationProblemPlusJSONResponse{
		Title:  &title,
		Status: &status,
		Detail: &detail,
		Type:   &typeStr,
	}
}

// writeUnauthorized writes a 401 Problem Details JSON response directly.
func writeUnauthorized(w http.ResponseWriter, detail string) {
	type problem struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(problem{
		Type:   "https://teammanager.example/errors/unauthorized",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: detail,
	})
}

// handlerError is a sentinel error type carrying an HTTP status for use in
// UploadMyPhoto where we cannot use typed response objects for non-200 paths.
type handlerError struct {
	status  int
	message string
}

func (e *handlerError) Error() string { return e.message }

func errUnauthorized(msg string) error { return &handlerError{status: 401, message: msg} }
func errBadRequest(msg string) error   { return &handlerError{status: 400, message: msg} }
func errInternal(msg string) error     { return &handlerError{status: 500, message: msg} }

// ensure time is used (time.Time in UserRow.Birthday).
var _ = time.Time{}
