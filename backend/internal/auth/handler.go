package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// authService is the interface the Handler relies on.
type authService interface {
	Login(ctx context.Context, email, password string) (token string, user *UserRow, err error)
	ValidateToken(ctx context.Context, tokenString string) (*UserRow, error)
	Logout(ctx context.Context, tokenHash string) error
	UpdatePhoto(ctx context.Context, userID string, data []byte, mime string) (*UserRow, error)
	EraseAccount(ctx context.Context, userID, password string) error
	ExportUserData(ctx context.Context, userID string) (*ExportData, error)
}

// Handler implements the auth-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    authService
	logger *slog.Logger
	codec  *SessionCookieCodec
	audit  *audit.Logger
}

// NewHandler creates a new Handler. The codec is used by AuthMiddleware to read
// the encrypted session cookie. al is the shared audit logger; when nil a
// log-only logger is created from logger.
func NewHandler(svc authService, logger *slog.Logger, codec *SessionCookieCodec, al *audit.Logger) *Handler {
	if al == nil {
		al = audit.New(logger)
	}
	return &Handler{svc: svc, logger: logger, codec: codec, audit: al}
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

	token, user, err := h.svc.Login(ctx, string(request.Body.Email), request.Body.Password)
	if err != nil {
		h.logger.WarnContext(ctx, "login failed", "err", err)
		h.audit.Record(ctx, audit.EventLogin, audit.Failure, "", slog.String("email", string(request.Body.Email)))
		metrics.LoginAttempts.WithLabelValues("failure").Inc()
		return gen.Login401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("invalid credentials"),
		}, nil
	}

	metrics.LoginAttempts.WithLabelValues("success").Inc()
	h.audit.Record(ctx, audit.EventLogin, audit.Success, user.Id.String(), slog.String("email", string(request.Body.Email)))
	return gen.Login200JSONResponse{
		Token: token,
		User:  toGenUser(user),
	}, nil
}

// DeleteCurrentUser erases the authenticated account by anonymization
// (GDPR Art. 17). The account's own email must be supplied in confirmEmail to
// re-authenticate the request (not the password, so OIDC-only accounts with
// no password can still confirm); on success the session cookie is cleared
// by the cookie middleware.
func (h *Handler) DeleteCurrentUser(ctx context.Context, request gen.DeleteCurrentUserRequestObject) (gen.DeleteCurrentUserResponseObject, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return gen.DeleteCurrentUser401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("not authenticated"),
		}, nil
	}
	if request.Body == nil || request.Body.ConfirmEmail == "" {
		return gen.DeleteCurrentUser401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("email confirmation required"),
		}, nil
	}

	if err := h.svc.EraseAccount(ctx, user.Id.String(), string(request.Body.ConfirmEmail)); err != nil {
		h.logger.WarnContext(ctx, "account erasure failed", "err", err)
		var soleErr *SoleSettingsAdminError
		if errors.As(err, &soleErr) {
			h.audit.Record(ctx, audit.EventAccountErase, audit.Failure, user.Id.String(),
				slog.Any("blockedByTeamIds", soleErr.TeamIDs))
			return nil, apierror.Conflict(ErrSoleSettingsAdmin.Error())
		}
		h.audit.Record(ctx, audit.EventAccountErase, audit.Failure, user.Id.String())
		return gen.DeleteCurrentUser401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("invalid credentials"),
		}, nil
	}

	h.audit.Record(ctx, audit.EventAccountErase, audit.Success, user.Id.String())
	metrics.TeamEvents.WithLabelValues("user", "delete").Inc()
	return gen.DeleteCurrentUser204Response{}, nil
}

// GetMyDataExport returns the authenticated user's personal data (GDPR Art. 15)
// as a downloadable JSON document.
func (h *Handler) GetMyDataExport(ctx context.Context, _ gen.GetMyDataExportRequestObject) (gen.GetMyDataExportResponseObject, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return gen.GetMyDataExport401ApplicationProblemPlusJSONResponse{
			UnauthorizedApplicationProblemPlusJSONResponse: unauthorized("not authenticated"),
		}, nil
	}

	data, err := h.svc.ExportUserData(ctx, user.Id.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "data export failed", "err", err)
		return nil, errInternal("data export failed")
	}

	// The strict 200 body is a free-form object; round-trip the typed export
	// through JSON to populate it.
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, errInternal("data export encoding failed")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, errInternal("data export encoding failed")
	}

	disposition := `attachment; filename="teamverwaltung-datenexport-` + time.Now().Format("2006-01-02") + `.json"`
	return gen.GetMyDataExport200JSONResponse{
		Body:    body,
		Headers: gen.GetMyDataExport200ResponseHeaders{ContentDisposition: &disposition},
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
		return nil, apierror.NotFound("no profile photo")
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

	const maxPhotoBytes = 2 << 20 // 2 MB max
	data, err := io.ReadAll(io.LimitReader(part, maxPhotoBytes+1))
	if err != nil {
		h.logger.WarnContext(ctx, "UploadMyPhoto: read file data failed", "err", err)
		return nil, errBadRequest("cannot read file data")
	}
	// io.LimitReader silently truncates rather than erroring once the cap is
	// reached, so io.ReadAll alone can't distinguish "exactly at the limit"
	// from "oversized" -- reading one extra byte lets us detect the latter
	// and reject it explicitly (413) instead of letting the truncated data
	// fail image decoding downstream and fall through to a generic 500.
	if len(data) > maxPhotoBytes {
		return nil, apierror.New(http.StatusRequestEntityTooLarge, "Payload Too Large", "image exceeds the 2 MB upload limit")
	}

	// Detect MIME from actual content; reject anything other than JPEG/PNG.
	ct := http.DetectContentType(data)
	if ct != "image/jpeg" && ct != "image/png" {
		return nil, errBadRequest("only JPEG and PNG images are accepted")
	}

	updated, err := h.svc.UpdatePhoto(ctx, user.Id.String(), data, ct)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			return nil, errBadRequest("image dimensions exceed the allowed maximum")
		}
		h.logger.ErrorContext(ctx, "update photo failed", "err", err)
		return nil, errInternal("photo update failed")
	}

	metrics.TeamEvents.WithLabelValues("user", "update").Inc()
	return gen.UploadMyPhoto200JSONResponse(toGenUser(updated)), nil
}

// Logout invalidates the current session.
func (h *Handler) Logout(ctx context.Context, _ gen.LogoutRequestObject) (gen.LogoutResponseObject, error) {
	// The raw token is stored in context by AuthMiddleware.
	rawToken, _ := ctx.Value(rawBearerContextKey).(string)
	tokenHash := sha256Hex(rawToken)
	var actor string
	if u, ok := UserFromContext(ctx); ok {
		actor = u.Id.String()
	}
	if err := h.svc.Logout(ctx, tokenHash); err != nil {
		h.logger.WarnContext(ctx, "logout failed", "err", err)
		h.audit.Record(ctx, audit.EventLogout, audit.Failure, actor)
		return gen.Logout204Response{}, nil
	}
	h.audit.Record(ctx, audit.EventLogout, audit.Success, actor)
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

// unauthorized builds an UnauthorizedApplicationProblemPlusJSONResponse, with
// a Type URI computed via apierror so it honors ERROR_TYPE_BASE_URI like
// every other error response.
func unauthorized(detail string) gen.UnauthorizedApplicationProblemPlusJSONResponse {
	e := apierror.New(http.StatusUnauthorized, "Unauthorized", detail)
	return gen.UnauthorizedApplicationProblemPlusJSONResponse{
		Title:  &e.Title,
		Status: &e.Status,
		Detail: &detail,
		Type:   &e.Type,
	}
}

// writeUnauthorized writes a 401 Problem Details JSON response directly.
func writeUnauthorized(w http.ResponseWriter, detail string) {
	apierror.Unauthorized(detail).Render(w)
}

// errUnauthorized/errBadRequest/errInternal build *apierror.APIError values
// for handler methods (e.g. UploadMyPhoto) that return non-200 responses as
// a plain error rather than a typed response object. ResponseErrorHandler's
// errors.As(err, *apierror.APIError) only matches *apierror.APIError, so
// these must return that concrete type -- a bespoke error type here would
// silently fall through to a generic 500 on every call site.
func errUnauthorized(msg string) error { return apierror.Unauthorized(msg) }
func errBadRequest(msg string) error   { return apierror.BadRequest(msg) }
func errInternal(msg string) error     { return apierror.Internal(msg) }

// ensure time is used (time.Time in UserRow.Birthday).
var _ = time.Time{}
