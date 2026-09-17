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

	token, user, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
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

	_, first, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "user@example.com", "User"))
	require.NoError(t, err)

	_, second, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
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

	_, original, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "old@example.com", "User"))
	require.NoError(t, err)

	_, afterChange, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
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

	require.NoError(t, svc.Register(context.Background(), "existing@example.com", "longenoughpassword"))
	existing, err := repo.FindUserByEmail(context.Background(), "existing@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, existing.PasswordHash)

	_, user, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "existing@example.com", "Existing"))
	require.NoError(t, err)

	assert.Equal(t, existing.Id, user.Id, "linked, not duplicated")
	assert.NotEmpty(t, user.PasswordHash, "linking must not remove the existing password")
	assert.NotNil(t, user.EmailVerifiedAt, "an unverified account becomes verified by the link")
}

func TestLoginWithOIDC_RejectsUnverifiedEmail(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	claims := oidcClaims("sub-1", "user@example.com", "User")
	claims.EmailVerified = false

	_, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider, claims)
	require.ErrorIs(t, err, auth.ErrOIDCEmailUnverified)

	_, findErr := repo.FindUserByEmail(context.Background(), "user@example.com")
	assert.Error(t, findErr, "no account may be created from an unverified assertion")
}

func TestLoginWithOIDC_RejectsMissingEmail(t *testing.T) {
	t.Parallel()
	svc := newRegTestService(t, newRegTestRepo(), time.Hour, true)

	_, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "", "User"))
	require.ErrorIs(t, err, auth.ErrOIDCEmailUnverified)
}

func TestLoginWithOIDC_RejectsMissingSubject(t *testing.T) {
	t.Parallel()
	svc := newRegTestService(t, newRegTestRepo(), time.Hour, true)

	_, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
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

	_, _, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
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

	_, user, err := svc.LoginWithOIDC(context.Background(), testOIDCProvider,
		oidcClaims("sub-1", "user@example.com", "User"))
	require.NoError(t, err)
	assert.NotNil(t, user)
}
