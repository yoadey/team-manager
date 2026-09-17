## ADDED Requirements

### Requirement: Configured OIDC provider is offered as a login option
When an OIDC provider is configured, `GET /auth/providers` MUST list it in
addition to the password provider, carrying the operator-configured display name,
subtitle and icon URL. When no provider is configured, the response MUST be
unchanged from today's password-only list.

#### Scenario: Provider configured
- **WHEN** OIDC is enabled and a client requests the provider list
- **THEN** the response contains both the `password` provider and the configured
  OIDC provider with its configured name, subtitle and icon URL

#### Scenario: Provider not configured
- **WHEN** OIDC is disabled and a client requests the provider list
- **THEN** the response contains only the `password` provider

#### Scenario: Start endpoint with OIDC disabled
- **WHEN** OIDC is disabled and a client requests the OIDC start endpoint
- **THEN** the request is rejected with a 404 problem+json response rather than a
  redirect

### Requirement: Login starts with an Authorization Code + PKCE request
The OIDC start endpoint MUST redirect the browser to the provider's
authorization endpoint using the Authorization Code flow with PKCE, carrying a
single-use `state` and `nonce`, and MUST persist the `state`, `nonce` and PKCE
verifier in a short-lived, encrypted, `HttpOnly` cookie. The request MUST NOT
send a `prompt` parameter.

#### Scenario: Start redirects to the provider
- **WHEN** a client requests the OIDC start endpoint while OIDC is enabled
- **THEN** it receives a 302 to the provider's authorization endpoint with
  `response_type=code`, a `code_challenge` using S256, a `state`, a `nonce`, and
  no `prompt` parameter, and the response sets the encrypted state cookie

#### Scenario: Configured scopes are sent
- **WHEN** base scopes and extra scopes are both configured
- **THEN** the authorization request's `scope` parameter contains the base scopes
  followed by the extra scopes

#### Scenario: The openid scope cannot be configured away
- **WHEN** an operator overrides the base scopes without including `openid`
- **THEN** the authorization request still requests `openid`, exactly once

### Requirement: Callback verifies the provider's response before any token exchange
The callback MUST reject a request whose `state` does not match the state cookie,
and MUST do so before exchanging the authorization code. It MUST verify the ID
token's signature, issuer, audience, expiry and `nonce`.

#### Scenario: State mismatch
- **WHEN** the callback receives a `state` that does not match the state cookie
- **THEN** the login fails and no token exchange request is made to the provider

#### Scenario: Missing state cookie
- **WHEN** the callback receives a request with no state cookie
- **THEN** the login fails and no token exchange request is made to the provider

#### Scenario: Provider returned an error
- **WHEN** the callback receives an `error` parameter from the provider
- **THEN** the login fails and no token exchange request is made to the provider

### Requirement: Unverified email addresses are rejected
The callback MUST refuse to establish a session when the ID token does not assert
a verified email address.

#### Scenario: Email not verified by the provider
- **WHEN** the ID token carries `email_verified` false or omits it
- **THEN** no session is established, no account is created or linked, and the
  user is returned to the login screen with an error

### Requirement: An existing link is resolved by subject
When an `oidc_accounts` row already exists for the provider and the ID token's
subject, the callback MUST establish a session for that user, regardless of the
email address in the token.

#### Scenario: Known subject
- **WHEN** a user has previously signed in through the provider and signs in again
- **THEN** a session is established for the same account

#### Scenario: Email changed at the provider
- **WHEN** a linked user's email address at the provider has changed since the
  link was made
- **THEN** a session is established for the same account, matched by subject

### Requirement: A first sign-in links to an existing account by verified email
When no link exists and an account with the token's email address exists, the
callback MUST link the OIDC subject to that account and establish a session. If
that account's address was not yet verified, it MUST become verified.

#### Scenario: Existing password account with the same address
- **WHEN** a user whose account was created with a password signs in through the
  provider for the first time with the same verified address
- **THEN** the subject is linked to that account and a session is established for
  it, with the account's password left intact

#### Scenario: Existing account was not yet verified
- **WHEN** the matched account had never completed email verification
- **THEN** the account is marked verified as part of the link, and any password
  it was carrying is discarded

#### Scenario: A password parked on an unverified account cannot be armed
- **WHEN** someone self-registers another person's address with a password of
  their choosing and never verifies it, and that address's real owner later
  signs in through the provider
- **THEN** the account is adopted and marked verified, but the parked password
  no longer grants a password login

### Requirement: An unknown address creates a passwordless account
When no link and no account exist for the verified address, the callback MUST
create a new account with no password, mark it verified, link the subject to it,
and establish a session. The new account MUST NOT be granted membership in any
team.

#### Scenario: Brand-new user
- **WHEN** a user with no existing account signs in through the provider
- **THEN** an account is created with no password set, marked verified, linked to
  the subject, and a session is established

#### Scenario: New account has no team
- **WHEN** a newly provisioned account reaches the app
- **THEN** it sees the no-team state and gains access only once invited

#### Scenario: Address belongs to a deleted account
- **WHEN** the verified address is still held by a soft-deleted account
- **THEN** no account is created, no session is established, and the user is
  returned to the login screen with a distinct error

### Requirement: A successful callback establishes an ordinary session
A successful callback MUST issue the same session as password login — the
encrypted `HttpOnly` session cookie — record the originating provider on the
session, and redirect the browser back to the application.

#### Scenario: Session cookie is set on the redirect
- **WHEN** the callback succeeds
- **THEN** the 302 response sets the session cookie, and the session token is
  never present in a response body

#### Scenario: Session records its provider
- **WHEN** a session is established through the OIDC flow
- **THEN** the stored session records the OIDC provider rather than `password`

### Requirement: Failures return the user to the login screen
A failed callback MUST redirect to the application with a coarse error code
rather than rendering an error page, and the application MUST show a translated
message and remove the code from the URL. Error codes MUST NOT reveal whether an
account exists for a given address.

#### Scenario: Failure redirects with a code
- **WHEN** the callback fails for any reason
- **THEN** the browser is redirected to the application with an error code in the
  query string and no session cookie is set

#### Scenario: Application surfaces and clears the error
- **WHEN** the application loads with an OIDC error code in the URL
- **THEN** it shows the corresponding translated message on the login screen and
  replaces the URL without the code

### Requirement: Password login remains available
Enabling OIDC MUST NOT disable or alter password login, self-registration,
password reset, or existing sessions.

#### Scenario: Password login alongside OIDC
- **WHEN** OIDC is enabled and a user submits valid email and password
- **THEN** they are logged in exactly as before

#### Scenario: Linked account keeps its password
- **WHEN** an account whose address was already verified, and which has a
  password, is linked to an OIDC subject
- **THEN** that account can still be used to log in with the password

### Requirement: A login keeps the page it started from
The start endpoint MUST accept the path the login began on and return the
browser there after a successful callback, so an invite link still redeems. The
value MUST travel in the encrypted state cookie rather than through the
provider, and MUST be restricted to a root-relative path within this
application.

#### Scenario: Invite link survives the round trip
- **WHEN** a user opens an invite link and signs in through the provider
- **THEN** the callback returns them to that invite path, which the application
  redeems as it would after a password login

#### Scenario: An off-site return path is refused
- **WHEN** the start endpoint is given a return path that is an absolute URL, a
  protocol-relative `//host`, or any value that a browser would resolve off this
  origin
- **THEN** it is discarded and a successful login lands on the application root

### Requirement: Browser-facing endpoints answer navigations with redirects
The OIDC start and callback endpoints are reached by navigating, so every
outcome -- including exceeding the rate limit -- MUST be a redirect back to the
login screen rather than a problem+json document.

#### Scenario: Rate limit exceeded
- **WHEN** a client exceeds the login rate limit on either OIDC endpoint
- **THEN** it receives a 302 back to the login screen carrying a login-error
  code, not an error document

### Requirement: Provider buttons can show an icon
`Provider` MUST carry an optional icon URL, and the login screen MUST render it
when present and fall back to the existing text glyph when absent. Icons for the
common providers MUST be served by the application itself rather than fetched
from a third-party host.

#### Scenario: Provider with an icon
- **WHEN** the provider list contains a provider with an icon URL
- **THEN** the login screen renders that image in the provider's button

#### Scenario: Provider without an icon
- **WHEN** a provider has no icon URL
- **THEN** the login screen renders the provider's text glyph, which is a single
  character so that it fits the button's badge

#### Scenario: Bundled icons are same-origin
- **WHEN** an operator selects one of the bundled provider icons
- **THEN** it is served from the application's own origin and renders under the
  application's content security policy without further configuration
