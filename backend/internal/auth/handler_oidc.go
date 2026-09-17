package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

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
	loginErrorRateLimited     = "oidc_rate_limited"
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
func (h *Handler) StartOidcLogin(ctx context.Context, request gen.StartOidcLoginRequestObject) (gen.StartOidcLoginResponseObject, error) {
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
	// Where the user was heading before being sent to the provider. It rides
	// in the encrypted state cookie rather than in the authorization request,
	// so the provider never sees it and nothing outside this server can alter
	// it between start and callback.
	if request.Params.ReturnTo != nil {
		state.ReturnURL = sanitizeReturnPath(*request.Params.ReturnTo)
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

	code, decoded, failure := h.validateCallback(ctx, request)
	if failure != "" {
		return h.callbackRedirect(h.loginErrorURL(failure)), nil
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
	return h.callbackRedirect(h.postLoginURL(decoded.ReturnURL)), nil
}

// maxReturnPathLen matches the spec's maxLength for return_to. Anything longer
// is not a path this application produces.
const maxReturnPathLen = 512

// sanitizeReturnPath reduces a caller-supplied return_to to something safe to
// append to this deployment's own origin, or to "" when it is not.
//
// Only a root-relative path is accepted, and the checks are the ones an
// open-redirect actually needs: "//host" and "/\host" are both protocol-
// relative once a browser resolves them, and a browser strips tab/CR/LF from a
// URL before resolving it, so a control character can smuggle either shape
// past a naive prefix test. The value reaches here from a query parameter on
// an unauthenticated endpoint, so it is assumed hostile.
func sanitizeReturnPath(raw string) string {
	if raw == "" || len(raw) > maxReturnPathLen {
		return ""
	}
	if strings.ContainsFunc(raw, func(r rune) bool {
		return r <= ' ' || r == '\\' || r == 0x7f
	}) {
		return ""
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return ""
	}
	return raw
}

// validateCallback checks everything that can be decided before any outbound
// request is made, returning the authorization code or a login error code.
//
// Ordering matters: the state check has to come before the token exchange, so
// an unsolicited callback can never make this server call out to the provider.
func (h *Handler) validateCallback(ctx context.Context, request gen.OidcCallbackRequestObject) (code string, state *oidcState, failure string) {
	if request.Params.Error != nil && *request.Params.Error != "" {
		// error_description is free text chosen by the provider, so it is
		// truncated before being logged rather than trusted to be short.
		description := ""
		if request.Params.ErrorDescription != nil {
			description = truncate(*request.Params.ErrorDescription, maxLoggedProviderError)
		}
		h.logger.InfoContext(ctx, "OIDC callback: provider returned an error",
			"error", truncate(*request.Params.Error, maxLoggedProviderError),
			"error_description", description)
		h.recordOIDCFailure(ctx, "provider")
		if *request.Params.Error == "access_denied" {
			return "", nil, loginErrorDenied
		}
		return "", nil, loginErrorGeneric
	}

	stateParam := ""
	if request.Params.State != nil {
		stateParam = *request.Params.State
	}
	cookieState, _ := ctx.Value(oidcStateContextKey{}).(string)
	decoded, err := decodeOIDCState(cookieState)
	// subtle.ConstantTimeCompare, not ==: the cookie is the only thing binding
	// this callback to a login this server started, so the comparison is a
	// secret comparison like any other.
	if err != nil || stateParam == "" ||
		subtle.ConstantTimeCompare([]byte(decoded.State), []byte(stateParam)) != 1 {
		h.logger.WarnContext(ctx, "OIDC callback: state did not match")
		h.recordOIDCFailure(ctx, "state")
		return "", nil, loginErrorGeneric
	}

	if request.Params.Code == nil || *request.Params.Code == "" {
		h.recordOIDCFailure(ctx, "code")
		return "", nil, loginErrorGeneric
	}
	return *request.Params.Code, decoded, ""
}

// maxLoggedProviderError bounds how much provider-supplied error text reaches
// the log. The provider controls it and it arrives on an unauthenticated
// endpoint, so it is neither trusted nor unbounded.
const maxLoggedProviderError = 200

// truncate shortens s to at most n runes, marking that it was cut.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// recordOIDCFailure emits the audit record and metric for a rejected login.
// actor is empty: a failed OIDC login has no authenticated subject, exactly
// like a failed password login.
func (h *Handler) recordOIDCFailure(ctx context.Context, reason string) {
	metrics.LoginAttempts.WithLabelValues("failure").Inc()
	h.audit.Record(ctx, audit.EventOIDCLogin, audit.Failure, "", slog.String("reason", reason))
}

// postLoginURL is where a successful callback returns the browser: this
// deployment's frontend origin, plus whatever path the login started from.
//
// Carrying that path matters beyond convenience -- an invite link
// (/join/{teamId}/{code}) is redeemed by the frontend from the URL it loads
// on, so dropping it turns "log in with Google to join this team" into a
// login that joins nothing.
func (h *Handler) postLoginURL(returnPath string) string {
	// Re-validated rather than trusted: the cookie is this server's own, but
	// the value inside it came from a query parameter, and the single cheap
	// check here is what keeps that provenance from mattering.
	path := sanitizeReturnPath(returnPath)
	if path == "" {
		path = "/"
	}
	base := h.oidc.Config().PostLoginURL
	if base == "" {
		return path
	}
	return base + path
}

// RateLimitRedirectURL is where a rate-limited OIDC request sends the browser.
// Wired into middleware.PerIPRateLimitRedirect by cmd/server/main.go, so a
// user who trips the limiter lands on the login screen with an explanation
// rather than on a page of raw problem+json.
func (h *Handler) RateLimitRedirectURL() string {
	return h.loginErrorURL(loginErrorRateLimited)
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
