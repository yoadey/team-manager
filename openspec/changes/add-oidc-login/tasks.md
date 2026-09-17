## 1. Spec
- [x] 1.1 Add `GET /auth/oidc/start` and `GET /auth/oidc/callback` to `openapi.yaml` (`security: []`, 302 via a new `OidcRedirect` response, 404 problem+json on start when disabled)
- [x] 1.2 Add optional `icon` (URL) to the `Provider` schema
- [x] 1.3 Run `cd backend && make generate` and repo-root `make generate-ts`; commit generated output

## 2. Backend — configuration
- [x] 2.1 `loadOIDCConfig` in `internal/config/config.go` following the `loadS3Config`/`loadSMTPConfig` shape, with `ErrOIDCConfigRequired` when enabled without issuer/client id/client secret
- [x] 2.2 `OIDC_REDIRECT_URL` defaults to `PUBLIC_BASE_URL` + `/api/v1/auth/oidc/callback`
- [x] 2.3 Validate `OIDC_PROVIDER_ICON`: same-origin path or `https://` only; reject `javascript:`/`data:`

## 3. Backend — OIDC client
- [x] 3.1 Add `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`; `go mod tidy`
- [x] 3.2 `internal/auth/oidc.go`: discovery (lazy retry after a failed boot attempt), authorization URL with PKCE S256 + state + nonce and no `prompt`, code exchange, ID-token verification
- [x] 3.3 Encrypted, short-lived `tv_oidc` state cookie reusing the session cookie's AES-GCM keys; `SameSite=Lax`, callback-scoped path

## 4. Backend — session + account resolution
- [x] 4.1 `repository.go`: `FindUserByOIDCSubject`, `LinkOIDCAccount`; parameterize `CreateSession`'s hardcoded `provider` column; extend the `authRepo` interface and its mocks
- [x] 4.2 `service.go`: `LoginWithOIDC` — reject unverified email, then subject → email-link (marking verified) → provision passwordless, all onto `createSessionAndSign`; map `ErrEmailTaken` to a distinct deleted-account error
- [x] 4.3 `cookie.go`: carry an OIDC state-cookie action in the ctx holder; set the session cookie for the 302 callback response
- [x] 4.4 `handler.go`: `ListProviders` advertises the configured provider; `StartOidcLogin` / `OidcCallback`; validate state before any token exchange; always 302 back with `?login_error=` on failure
- [x] 4.5 `cmd/server/main.go`: register both routes in the public block; rate-limit start with the login limiter
- [x] 4.6 `internal/audit`: `auth.oidc.login` / `auth.oidc.link` / `auth.oidc.provision` constants; reuse the existing `LoginAttempts` metric labels

## 5. Frontend
- [x] 5.1 `serviceLayerReal`: `startProviderLogin(pid)` navigating to the start endpoint; `doLogin` uses it instead of the legacy password POST
- [x] 5.2 Bootstrap reads `?login_error=`, cleans the URL with `history.replaceState`, and surfaces a translated message
- [x] 5.3 `Login.tsx` renders `provider.icon` when present, text glyph otherwise
- [x] 5.4 `scripts/generate-provider-icons.mjs` emitting `public/provider-icons/*.svg` from `@mui/icons-material` path data (baked fill); commit the generated assets and verify each renders correctly
- [x] 5.5 `i18n/de.ts` + `i18n/en.ts` strings for every `login_error` code
- [x] 5.6 Update `Login.test.tsx` / `AppContext.test.tsx` for the redirect-based provider login; cover icon rendering and the error-param path

## 6. Chart + docs
- [x] 6.1 `oidc` section in `values.yaml`, `values.schema.json` and `templates/_env.tpl`
- [x] 6.2 `frontend.extraImgSrc` wired through `config.js.template` / entrypoint / `values.schema.json` into the CSP `img-src`
- [x] 6.3 Backend env table in `CLAUDE.md`; chart `README.md`

## 7. Verification
- [x] 7.1 `backend-openapi-drift`: `make generate` + `make generate-ts` leave no diff
- [x] 7.2 Backend tests: state mismatch short-circuits before token exchange; unverified email rejected; all three resolution paths; deleted-address conflict; session cookie set on the 302; `sessions.provider` recorded
- [x] 7.3 `make lint` + `make test` green; migration gates unaffected (no migration added). govulncheck and the Docker-backed repository tests could not be run in the authoring environment (blocked vuln.go.dev egress / no Docker daemon) — both run in CI.
- [x] 7.4 Frontend `lint` + `typecheck` + `test:coverage` (80/65/75/80) + `build` + `check:bundle` (250 KB/chunk, 600 KB total gzipped) green
- [x] 7.5 `openspec validate add-oidc-login --strict` green
- [ ] 7.6 Independent code review by a fresh subagent with no prior context on the change
