## Why

`CLAUDE.md` describes the auth module as "OIDC-ready", but no OIDC integration
exists. What exists is scaffolding only: an empty `oidc_accounts` table whose own
migration comment says "no OIDC integration exists yet and no code path writes
here" (`backend/internal/db/migrations/00001_init.sql:285-293`), a nullable
`users.password_hash`, and a `GET /auth/providers` endpoint that returns exactly
one hardcoded `password` provider (`backend/internal/auth/handler.go:72-85`).
The frontend still carries the matching dead path: `doLogin(pid)` POSTs the
provider id into the `email` field (`frontend/src/services/serviceLayerReal.ts:194-202`),
which the real backend rejects at `validate.Email()`.

The first concrete deployment need is a "Sign in with Google" button, brokered
through a self-hosted ZITADEL instance that already has Google configured as an
identity provider — with ZITADEL's own login screen skipped entirely, so users go
straight from the app to Google.

Password login stays: it is the fallback when the identity provider is
unreachable, and the only way in for accounts without a Google address.

## What Changes

- Add a real OIDC Authorization Code + PKCE login flow to the backend: a start
  endpoint that redirects to the provider, and a callback endpoint that verifies
  the ID token and establishes the ordinary session.
- Advertise the configured provider through `GET /auth/providers` alongside
  `password`, so the existing provider-button UI picks it up.
- Give `Provider` an optional `icon` URL, and ship the common providers' icons as
  static frontend assets generated from `@mui/icons-material`, so an operator can
  point `OIDC_PROVIDER_ICON` at a bundled icon (or any other URL).
- Account resolution on callback: match an existing `oidc_accounts` row by
  subject; else match an existing user by verified email and link; else create a
  new passwordless, email-verified account. A new account has no team until it is
  invited, exactly like self-registration.
- Make the authorization request's scopes configurable, including extra scopes,
  so a deployment can pass a provider-specific hint. This is what lets ZITADEL
  skip its own login screen: `urn:zitadel:iam:org:idp:id:<idpID>` makes it start
  the IdP flow directly. No vendor-specific string is baked into the app.
- Add the matching `oidc` section to the Helm chart.

## Capabilities

### New Capabilities
- `oidc-login`: sign-in through an external OpenID Connect provider, with
  account linking and provisioning, alongside the existing password login.

## Impact

- Spec: `backend/openapi/openapi.yaml` gains `GET /auth/oidc/start` and
  `GET /auth/oidc/callback` (both `security: []`, both 302) and an optional
  `icon` on `Provider`. Regenerate both clients.
- Backend: new `backend/internal/auth/oidc.go` (discovery, PKCE, ID-token
  verification) with new dependencies `github.com/coreos/go-oidc/v3` and
  `golang.org/x/oauth2`; `service.go` gains `LoginWithOIDC` on top of the
  existing `createSessionAndSign`; `repository.go` gains `FindUserByOIDCSubject`
  / `LinkOIDCAccount` and parameterizes the already-existing but hardcoded
  `sessions.provider` column; `cookie.go` gains the short-lived OIDC state
  cookie and a `applyCookie` case for the 302 callback; `config.go` gains
  `loadOIDCConfig`; `cmd/server/main.go` registers both routes in the public
  block; `internal/audit` gains OIDC event constants. **No migration** — both
  tables and columns already exist.
- Frontend: `doLogin` becomes a full-page navigation to the start endpoint;
  bootstrap reads a `?login_error=` param on the way back; `Login.tsx` renders
  `icon` when present; new `frontend/public/provider-icons/*.svg` plus the
  generator that produces them; `i18n/{de,en}.ts`.
- Chart: `oidc.*` values + schema + env rendering; `frontend.extraImgSrc` so a
  non-same-origin icon URL is not silently blocked by the app's CSP.
- Docs: backend env table in `CLAUDE.md`, chart `README.md`.
- CI: openapi-drift, backend lint/test/govulncheck, frontend
  lint/typecheck/coverage/build/bundle budget.
