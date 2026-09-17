package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
)

const testOIDCProvider = "google"

func oidcClaims(subject, email, name string) auth.OIDCClaims {
	return auth.OIDCClaims{Subject: subject, Email: email, EmailVerified: true, Name: name}
}

func TestLoginWithOIDC_ProvisionsNewAccount(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	token, user, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "New@Example.com", "New User"))
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	require.NotNil(t, user)

	assert.Equal(t, "new@example.com", user.Email, "email is normalized on the way in")
	assert.Equal(t, "New User", user.Name)
	assert.Empty(t, user.PasswordHash, "a provisioned OIDC account has no password")
	assert.NotNil(t, user.EmailVerifiedAt, "the provider already verified the address")

	// The session must record the provider it came from, not "password".
	sess := repo.sessionsForUser(user.Id.String())
	require.Len(t, sess, 1)
	assert.Equal(t, testOIDCProvider, sess[0].Provider)
}

func TestLoginWithOIDC_SecondLoginReusesTheSameAccount(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	_, first, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "user@example.com", "User"))
	require.NoError(t, err)

	_, second, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "user@example.com", "User"))
	require.NoError(t, err)

	assert.Equal(t, first.Id, second.Id)
}

// The subject, not the address, is the stable identity: a user who changes
// their email at the provider must land on the same local account.
func TestLoginWithOIDC_MatchesBySubjectAfterEmailChange(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	_, original, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "old@example.com", "User"))
	require.NoError(t, err)

	_, afterChange, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "new@example.com", "User"))
	require.NoError(t, err)

	assert.Equal(t, original.Id, afterChange.Id)
	assert.Equal(t, "old@example.com", afterChange.Email,
		"matching by subject must not silently rewrite the local address")
}

func TestLoginWithOIDC_LinksExistingPasswordAccount(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)
	ctx := context.Background()

	require.NoError(t, svc.Register(ctx, "existing@example.com", "longenoughpassword"))
	existing, err := repo.FindUserByEmail(ctx, "existing@example.com")
	require.NoError(t, err)
	require.NoError(t, repo.MarkEmailVerified(ctx, existing.Id.String()))
	require.NotEmpty(t, existing.PasswordHash)

	_, user, _, err := svc.LoginWithOIDC(ctx, testOIDCProvider,
		oidcClaims("sub-1", "existing@example.com", "Existing"))
	require.NoError(t, err)

	assert.Equal(t, existing.Id, user.Id, "linked, not duplicated")
	assert.NotEmpty(t, user.PasswordHash,
		"linking must not disturb a password its owner already proved they control")
	assert.NotNil(t, user.EmailVerifiedAt)
}

// Account squatting: self-registration accepts any address from an
// unauthenticated request, so anyone can park a row on somebody else's
// address carrying a password of their choosing. It stays inert only because
// Login refuses an unverified account. The OIDC link path then marks that row
// verified -- correctly, the provider just proved control of the address --
// which would arm the parked password and hand the squatter a working
// password login on the real owner's account. Adopting a never-verified row
// therefore has to drop whatever password it was carrying.
func TestLoginWithOIDC_DropsPasswordParkedOnAnUnverifiedAccount(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)
	ctx := context.Background()

	// The squatter registers the victim's address and never verifies it.
	require.NoError(t, svc.Register(ctx, "victim@example.com", "squatterpassword"))
	parked, err := repo.FindUserByEmail(ctx, "victim@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, parked.PasswordHash)
	require.Nil(t, parked.EmailVerifiedAt)

	// The real owner signs in through the provider, which adopts that row.
	_, user, outcome, err := svc.LoginWithOIDC(ctx, testOIDCProvider,
		oidcClaims("sub-1", "victim@example.com", "Victim"))
	require.NoError(t, err)
	assert.Equal(t, auth.OIDCLoginLinked, outcome)
	assert.Equal(t, parked.Id, user.Id)
	assert.NotNil(t, user.EmailVerifiedAt)
	assert.Empty(t, user.PasswordHash, "the parked password must not survive the link")

	// And the squatter's password must not work now that the row is verified.
	_, _, err = svc.Login(ctx, "victim@example.com", "squatterpassword")
	require.Error(t, err, "the parked password must not become a usable login")
}

// A provider-supplied name is free text: it never passes through a request
// handler, so nothing else bounds it before it reaches users.name. A null
// byte alone would fail the INSERT as an unhandled 500 mid-login.
func TestLoginWithOIDC_SanitizesTheProvidedName(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	_, user, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "user@example.com", "Erika\x00\tMustermann"))
	require.NoError(t, err)

	assert.Equal(t, "ErikaMustermann", user.Name)
}

func TestLoginWithOIDC_RejectsUnverifiedEmail(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	claims := oidcClaims("sub-1", "user@example.com", "User")
	claims.EmailVerified = false

	_, _, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider, claims)
	require.ErrorIs(t, err, auth.ErrOIDCEmailUnverified)

	_, findErr := repo.FindUserByEmail(context.Background(), "user@example.com")
	assert.Error(t, findErr, "no account may be created from an unverified assertion")
}

func TestLoginWithOIDC_RejectsMissingEmail(t *testing.T) {
	t.Parallel()
	svc := newRegTestService(t, newRegTestRepo(), time.Hour, true)

	_, _, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "", "User"))
	require.ErrorIs(t, err, auth.ErrOIDCEmailUnverified)
}

func TestLoginWithOIDC_RejectsMissingSubject(t *testing.T) {
	t.Parallel()
	svc := newRegTestService(t, newRegTestRepo(), time.Hour, true)

	_, _, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("", "user@example.com", "User"))
	require.ErrorIs(t, err, auth.ErrOIDCState)
}

// A soft-deleted account still holds its address in the unique index, so the
// provision path conflicts. That has to surface as its own explainable
// outcome rather than a generic failure.
func TestLoginWithOIDC_AddressHeldByDeletedAccount(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	require.NoError(t, svc.Register(context.Background(), "gone@example.com", "longenoughpassword"))
	repo.softDelete("gone@example.com")

	_, _, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "gone@example.com", "Gone"))
	require.ErrorIs(t, err, auth.ErrOIDCAccountDeleted)
}

// Self-registration being switched off must not also switch off OIDC login:
// they are separate doors, and an operator who closes public signup still
// expects their identity provider to work.
func TestLoginWithOIDC_WorksWhileSelfRegistrationIsDisabled(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, false)

	_, user, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "user@example.com", "User"))
	require.NoError(t, err)
	assert.NotNil(t, user)
}

// The three resolution paths must be distinguishable by the caller, which is
// what lets the handler audit a first-time link or provision separately from
// routine repeat logins.
func TestLoginWithOIDC_ReportsWhichPathItTook(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)
	ctx := context.Background()

	_, _, provisioned, err := svc.LoginWithOIDC(ctx, testOIDCProvider,
		oidcClaims("sub-new", "fresh@example.com", "Fresh"))
	require.NoError(t, err)
	assert.Equal(t, auth.OIDCLoginProvisioned, provisioned)

	_, _, repeat, err := svc.LoginWithOIDC(ctx, testOIDCProvider,
		oidcClaims("sub-new", "fresh@example.com", "Fresh"))
	require.NoError(t, err)
	assert.Equal(t, auth.OIDCLoginExisting, repeat)

	require.NoError(t, svc.Register(ctx, "haspassword@example.com", "longenoughpassword"))
	_, _, linked, err := svc.LoginWithOIDC(ctx, testOIDCProvider,
		oidcClaims("sub-linked", "haspassword@example.com", "Has Password"))
	require.NoError(t, err)
	assert.Equal(t, auth.OIDCLoginLinked, linked)
}

// Two first-time logins for the same new address can race (a double-clicked
// button, two tabs): one insert wins, the other comes back ErrEmailTaken. That
// must resolve to the account that won, not to "this address belonged to a
// deleted account" -- which is what a blanket ErrEmailTaken mapping reported,
// and is simply untrue for a live account.
func TestLoginWithOIDC_ConcurrentFirstLoginIsNotReportedAsDeleted(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)
	ctx := context.Background()

	// Stand in for the racing winner: the row exists by the time this login's
	// own insert runs, exactly as if a concurrent request had just created it.
	repo.simulateEmailTakenOnce("raced@example.com")

	_, user, outcome, err := svc.LoginWithOIDC(ctx, testOIDCProvider,
		oidcClaims("sub-race", "raced@example.com", "Raced"))
	require.NoError(t, err, "a lost race must still log the user in")
	require.NotNil(t, user)
	assert.Equal(t, "raced@example.com", user.Email)
	assert.Equal(t, auth.OIDCLoginLinked, outcome)
}
