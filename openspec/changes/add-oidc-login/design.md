## Context

Sessions today are minted in exactly one place: `Service.createSessionAndSign`
(`backend/internal/auth/service.go:231`), documented as "Shared by Login and
VerifyEmail". `VerifyEmail` (`service.go:352`) already establishes a session with
no password involved, which makes it the structural precedent for an OIDC
callback. The session itself is an RS256 JWT whose `jti` is the raw session
token, encrypted into the `tv_session` cookie by `SessionCookieCodec`
(`cookie.go`), and the `sessions` table already carries a `provider` column that
Go code hardcodes to `'password'` (`repository.go:175`).

Deployment target: frontend and backend share one origin (`/` → frontend,
`/api` → backend behind one Ingress), so every cookie in this flow is
first-party.

## Goals / Non-Goals

**Goals:**
- Authorization Code flow with PKCE against one configured OIDC provider.
- Reuse the existing session/cookie machinery rather than a parallel one.
- Deterministic, safe account resolution: subject → verified email → new account.
- Keep password login fully functional.
- Configuration-only support for skipping ZITADEL's login screen.

**Non-Goals:**
- More than one OIDC provider at a time. The `oidc_accounts.provider` column and
  the `{provider}`-free URL shape leave room for it; the config does not.
- RP-initiated logout / single logout. Logging out clears the local session; the
  provider session is untouched, so signing back in is one click.
- Migrating existing password users onto OIDC, or disabling password login.

## Decisions

**Flow shape.** `GET /auth/oidc/start` 302s to the provider's authorization
endpoint. `GET /auth/oidc/callback` exchanges the code, verifies the ID token,
resolves the account, sets `tv_session`, and 302s back to the frontend. Both are
plain top-level navigations, so no CORS and no `connect-src` change is involved.

**Why a 302 and not JSON.** The browser must leave the app; a fetch cannot
follow a cross-origin authorization redirect. The generated strict-server
interface already supports 302 responses with a `Location` header — four photo
endpoints use that shape (`components/responses/PhotoRedirect`) — so this needs
no new codegen mechanism.

**Cookie plumbing.** `applyCookie` (`cookie.go:183`) currently decides whether to
set the session cookie by switching on the operation id *and* asserting a JSON
200 response type. A 302 matches no case, so the callback would silently not set
a cookie. Rather than special-casing, the ctx holder is extended to carry both
the session token and an optional OIDC state-cookie action, and the middleware
applies both. This keeps handlers free of `http.ResponseWriter`, which is what
the strict-server interface is for.

**State cookie.** A separate short-lived `tv_oidc` cookie holds `state`, `nonce`
and the PKCE verifier, encrypted with the same AES-256-GCM keys as the session
cookie (`COOKIE_ENCRYPTION_KEYS`), 10-minute lifetime, `HttpOnly`, `Secure`,
`SameSite=Lax`, scoped to the callback path. `Lax` is required and sufficient:
the return from the provider is a top-level GET navigation, which `Lax` permits
and `Strict` would not. Keeping this state in a cookie rather than server-side
means no new table and no cleanup job.

**Account resolution.** In order:
1. `oidc_accounts` row for `(provider, subject)` → that user. Subject is the
   stable identifier; email changes at the provider must not orphan an account.
2. Otherwise a user with that email (`FindUserByEmail` already lowercases) →
   link, and mark the address verified if it was not. The provider has just
   proven control of the address, which is exactly what the emailed verification
   link proves.
3. Otherwise create a passwordless account (`CreateUnverifiedUser(..., "")` — the
   empty hash is already the established "OIDC-only account" marker, asserted by
   `register_test.go:565`), mark it verified, and link it.

An unverified email in the ID token is rejected outright at step 0: without it,
anyone who can register an arbitrary address at any federated IdP could claim an
existing account by email. This is the single most important check in the flow.

**Deleted accounts.** `FindUserByEmail` filters `deleted_at IS NULL`, but the
unique index does not — a soft-deleted user still holds the address, so step 3
hits `ON CONFLICT DO NOTHING` and returns `ErrEmailTaken`. That is a distinct
user-facing outcome ("this address belonged to a deleted account"), not a server
error, and is mapped to its own `login_error` code.

**Scopes.** `OIDC_SCOPES` holds the base scopes (`openid profile email`);
`OIDC_EXTRA_SCOPES` is appended verbatim. The ZITADEL auto-redirect is then
purely deployment configuration:
`OIDC_EXTRA_SCOPES=urn:zitadel:iam:org:idp:id:<idpID>`. ZITADEL reads that
reserved scope and starts the IdP flow directly instead of rendering its login
screen — implemented in both of its login UIs (`internal/domain/request.go`'s
`SelectIDPScope` and `apps/login/src/lib/server/flow-initiation.ts`). The app
itself stays vendor-neutral.

Consequence worth recording: the authorization request must **not** send
`prompt=login`. ZITADEL's V1 flow returns its normal login step whenever a
prompt other than `select_account` is present, which would defeat the whole
point. No `prompt` is sent.

**Discovery failure is not a startup failure.** Provider discovery is attempted
at boot with a short timeout; on failure the error is logged and retried lazily
on first use. A single-node homelab should not have a backend that refuses to
start because the IdP is briefly down — password login would be exactly what is
needed at that moment.

**Error reporting.** The callback never renders an error page; it always 302s to
the frontend, on failure with `?login_error=<code>`. The frontend maps the code
to a translated message and cleans the URL, mirroring how the verify-email
bootstrap path already behaves. Codes are coarse on purpose (`oidc_failed`,
`oidc_email_unverified`, `oidc_account_deleted`) — enough to be actionable,
not enough to be an account-enumeration oracle.

**Icons.** `Provider.icon` is a URL rather than a symbolic name so operators can
point it anywhere. The bundled set is generated from `@mui/icons-material`'s own
path data into static SVGs under `public/provider-icons/`, checked in so the
build does not depend on the generator. The fill is baked in, because
`currentColor` does not resolve inside an `<img>`. The app's CSP is
`img-src 'self' data: blob:`, so an icon on a foreign host would silently fail
to render; `frontend.extraImgSrc` extends the directive through the same
envsubst mechanism that already injects `${API_BASE_URL}` into `connect-src`.

## Risks / Trade-offs

- **Email-based linking is the flow's trust anchor.** It is only as strong as the
  `email_verified` claim. Mitigated by rejecting unverified claims and by the
  operator controlling which IdPs the broker accepts.
- **A misconfigured `OIDC_REDIRECT_URL`** produces a provider-side error before
  the app is ever reached. The value defaults to `PUBLIC_BASE_URL` +
  `/api/v1/auth/oidc/callback` so the common case needs no configuration, and it
  must match the URI registered at the provider exactly.
- **Two new dependencies** in a deliberately dependency-light backend.
  Justification: JWKS retrieval, key rotation, caching, algorithm allow-listing
  and `iss`/`aud`/`nonce`/`exp` validation are security-critical and a poor
  candidate for a hand-rolled implementation. `github.com/coreos/go-oidc/v3` is
  the de-facto standard client and `golang.org/x/oauth2` is already in the module
  graph.
- **The callback performs outbound HTTP.** State is validated against the cookie
  before the token exchange, so an unsolicited callback never reaches the
  provider. That bounds a single request, not a series of them: a scripted
  client can call the start endpoint once, keep the `tv_oidc` cookie it was
  issued, and replay the callback with it until the state's ten minutes run
  out — the cookie's `Max-Age` only constrains a browser. Each replay would
  reach the token exchange, so the callback carries the same per-IP login rate
  limit as the start endpoint. A server-side single-use record would close the
  window completely, at the cost of a table and a cleanup job; the rate limit
  plus the short TTL is the proportionate trade for now.
- **Changing `doLogin` to a navigation** retires the legacy provider-id-as-email
  path. The MSW demo backend keeps advertising only `password`, so demo mode is
  unaffected; the tests that drove the old path are updated.
