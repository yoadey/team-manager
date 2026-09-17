package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
)

// Login error codes handed back to the frontend as ?login_error=<code>. They
// are deliberately coarse: enough for the login screen to show something
// actionable, never enough to reveal whether an account exists for a given
// address.
const (
	loginErrorGeneric         = "oidc_failed"
	loginErrorUnavailable     = "oidc_unavailable"
	loginErrorDenied          = "oidc_denied"
	loginErrorEmailUnverified = "oidc_email_unverified"
	loginErrorAccountDeleted  = "oidc_account_deleted"
)

// oidcStateContextKey carries the decrypted tv_oidc payload from
// OIDCStateMiddleware to OidcCallback. The generated strict handler receives
// no *http.Request, so a cookie can only reach it through the context.
type oidcStateContextKey struct{}

// SetOIDC enables the OIDC login flow on this handler. Wired in by
// cmd/server/main.go, mirroring SetImageDeliveryProxyEnabled; leaving it unset
// keeps the deployment password-only and makes the OIDC endpoints 404.
func (h *Handler) SetOIDC(client *OIDCClient, stateCodec *SessionCookieCodec) {
	h.oidc = client
	h.oidcStateCodec = stateCodec
}

// oidcEnabled reports whether both halves of the OIDC wiring are present.
func (h *Handler) oidcEnabled() bool {
	return h.oidc != nil && h.oidcStateCodec != nil
}

// OIDCStateMiddleware decrypts the short-lived state cookie into the request
// context for the OIDC routes. A missing or undecryptable cookie is not an
// error here -- the callback treats an absent state as a failed login, which
// is the same outcome and keeps this middleware free of response handling.
func (h *Handler) OIDCStateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.oidcEnabled() {
			if cookie, err := r.Cookie(h.oidcStateCodec.Name()); err == nil {
				if payload, decErr := h.oidcStateCodec.Decrypt(cookie.Value); decErr == nil {
					r = r.WithContext(context.WithValue(r.Context(), oidcStateContextKey{}, payload))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// StartOidcLogin begins the authorization-code flow: mint per-attempt secrets,
// stash them in the state cookie, and send the browser to the provider.
func (h *Handler) StartOidcLogin(ctx context.Context, _ gen.StartOidcLoginRequestObject) (gen.StartOidcLoginResponseObject, error) {
	if !h.oidcEnabled() {
		return gen.StartOidcLogin404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("no OIDC provider is configured"),
		}, nil
	}

	state, err := newOIDCState()
	if err != nil {
		h.logger.ErrorContext(ctx, "OIDC start: could not mint state", "err", err)
		return h.startRedirect(h.loginErrorURL(loginErrorGeneric)), nil
	}

	authURL, err := h.oidc.AuthCodeURL(ctx, state.State, state.Nonce, state.Verifier)
	if err != nil {
		// Discovery is lazy, so an unreachable or misconfigured provider
		// surfaces here rather than at startup. Send the user back to the
		// login screen with an explanation instead of an error page -- the
		// password form is right there and still works.
		h.logger.ErrorContext(ctx, "OIDC start: provider unavailable", "err", err)
		return h.startRedirect(h.loginErrorURL(loginErrorUnavailable)), nil
	}

	payload, err := state.encode()
	if err != nil {
		h.logger.ErrorContext(ctx, "OIDC start: could not encode state", "err", err)
		return h.startRedirect(h.loginErrorURL(loginErrorGeneric)), nil
	}
	cookie, err := h.oidcStateCodec.StateCookie(payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "OIDC start: could not seal state cookie", "err", err)
		return h.startRedirect(h.loginErrorURL(loginErrorGeneric)), nil
	}
	AddResponseCookie(ctx, cookie)

	return h.startRedirect(authURL), nil
}

// OidcCallback completes the flow. It always redirects: the caller is a
// browser mid-navigation, so an error document would strand the user on a
// blank page instead of back at the login screen.
func (h *Handler) OidcCallback(ctx context.Context, request gen.OidcCallbackRequestObject) (gen.OidcCallbackResponseObject, error) {
	if !h.oidcEnabled() {
		return h.callbackRedirect(h.loginErrorURL(loginErrorGeneric)), nil
	}

	// However this attempt ends, it is spent -- expire the cookie so a
	// replayed callback has no state left to validate against.
	AddResponseCookie(ctx, h.oidcStateCodec.ClearedStateCookie())

	code, failure := h.validateCallback(ctx, request)
	if failure != "" {
		return h.callbackRedirect(h.loginErrorURL(failure)), nil
	}

	state, _ := ctx.Value(oidcStateContextKey{}).(string)
	decoded, err := decodeOIDCState(state)
	if err != nil {
		h.recordOIDCFailure(ctx, "state")
		return h.callbackRedirect(h.loginErrorURL(loginErrorGeneric)), nil
	}

	claims, err := h.oidc.Exchange(ctx, code, decoded.Verifier, decoded.Nonce)
	if err != nil {
		h.logger.WarnContext(ctx, "OIDC callback: token exchange failed", "err", err)
		h.recordOIDCFailure(ctx, "exchange")
		return h.callbackRedirect(h.loginErrorURL(loginErrorUnavailable)), nil
	}

	token, user, outcome, err := h.svc.LoginWithOIDC(ctx, h.oidc.Config().ProviderID, *claims)
	if err != nil {
		h.logger.WarnContext(ctx, "OIDC callback: login rejected", "err", err)
		h.recordOIDCFailure(ctx, "login")
		switch {
		case errors.Is(err, ErrOIDCEmailUnverified):
			return h.callbackRedirect(h.loginErrorURL(loginErrorEmailUnverified)), nil
		case errors.Is(err, ErrOIDCAccountDeleted):
			return h.callbackRedirect(h.loginErrorURL(loginErrorAccountDeleted)), nil
		default:
			return h.callbackRedirect(h.loginErrorURL(loginErrorGeneric)), nil
		}
	}

	metrics.LoginAttempts.WithLabelValues("success").Inc()
	providerID := h.oidc.Config().ProviderID
	h.audit.Record(ctx, audit.EventOIDCLogin, audit.Success, user.Id.String(),
		slog.String("provider", providerID), slog.String("outcome", string(outcome)))
	// A first-time link or provision attaches an external identity to a local
	// account, which is worth being able to find in the audit log without
	// reading every routine login next to it.
	switch outcome {
	case OIDCLoginLinked:
		h.audit.Record(ctx, audit.EventOIDCLink, audit.Success, user.Id.String(),
			slog.String("provider", providerID))
	case OIDCLoginProvisioned:
		h.audit.Record(ctx, audit.EventOIDCProvision, audit.Success, user.Id.String(),
			slog.String("provider", providerID))
	case OIDCLoginExisting:
		// Ordinary repeat login; EventOIDCLogin above already covers it.
	}
	// Same contract as password login: the session JWT reaches the browser
	// only as the httpOnly cookie applyCookie sets, never in a body or URL.
	SetSessionToken(ctx, token)
	return h.callbackRedirect(h.postLoginURL()), nil
}

// validateCallback checks everything that can be decided before any outbound
// request is made, returning the authorization code or a login error code.
//
// Ordering matters: the state check has to come before the token exchange, so
// an unsolicited callback can never make this server call out to the provider.
func (h *Handler) validateCallback(ctx context.Context, request gen.OidcCallbackRequestObject) (code, failure string) {
	if request.Params.Error != nil && *request.Params.Error != "" {
		h.logger.InfoContext(ctx, "OIDC callback: provider returned an error",
			"error", *request.Params.Error)
		h.recordOIDCFailure(ctx, "provider")
		if *request.Params.Error == "access_denied" {
			return "", loginErrorDenied
		}
		return "", loginErrorGeneric
	}

	stateParam := ""
	if request.Params.State != nil {
		stateParam = *request.Params.State
	}
	cookieState, _ := ctx.Value(oidcStateContextKey{}).(string)
	decoded, err := decodeOIDCState(cookieState)
	if err != nil || stateParam == "" || decoded.State != stateParam {
		h.logger.WarnContext(ctx, "OIDC callback: state did not match")
		h.recordOIDCFailure(ctx, "state")
		return "", loginErrorGeneric
	}

	if request.Params.Code == nil || *request.Params.Code == "" {
		h.recordOIDCFailure(ctx, "code")
		return "", loginErrorGeneric
	}
	return *request.Params.Code, ""
}

// recordOIDCFailure emits the audit record and metric for a rejected login.
// actor is empty: a failed OIDC login has no authenticated subject, exactly
// like a failed password login.
func (h *Handler) recordOIDCFailure(ctx context.Context, reason string) {
	metrics.LoginAttempts.WithLabelValues("failure").Inc()
	h.audit.Record(ctx, audit.EventOIDCLogin, audit.Failure, "", slog.String("reason", reason))
}

// postLoginURL is where a successful callback returns the browser.
func (h *Handler) postLoginURL() string {
	base := h.oidc.Config().PostLoginURL
	if base == "" {
		return "/"
	}
	return base + "/"
}

// loginErrorURL builds the frontend URL carrying a login error code.
func (h *Handler) loginErrorURL(code string) string {
	base := "/"
	if h.oidcEnabled() {
		if configured := h.oidc.Config().PostLoginURL; configured != "" {
			base = configured + "/"
		}
	}
	return base + "?login_error=" + url.QueryEscape(code)
}

func (h *Handler) startRedirect(location string) gen.StartOidcLogin302Response {
	return gen.StartOidcLogin302Response{
		Headers: gen.OidcRedirectResponseHeaders{Location: &location},
	}
}

func (h *Handler) callbackRedirect(location string) gen.OidcCallback302Response {
	return gen.OidcCallback302Response{
		Headers: gen.OidcRedirectResponseHeaders{Location: &location},
	}
}
