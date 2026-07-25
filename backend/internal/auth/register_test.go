package auth_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/mailer"
	"github.com/yoadey/team-manager/backend/internal/storage"
)

// ─── in-memory fake repository ─────────────────────────────────────────────
//
// Register/VerifyEmail/ResendVerification exercise several repository calls
// together with real interdependent state (create -> conflict -> lookup ->
// issue token -> consume token -> mark verified), which the per-call-closure
// mockRepo in service_test.go isn't a good fit for. regTestRepo is a small,
// self-contained in-memory implementation of the authRepo interface instead.

var errRegTestNotFound = errors.New("auth_test: not found")

type regTestRepo struct {
	mu        sync.Mutex
	users     map[string]*auth.UserRow // keyed by normalized (lowercased) email
	usersByID map[string]*auth.UserRow
	tokens    map[string]*auth.EmailVerificationTokenRow // keyed by token hash
}

func newRegTestRepo() *regTestRepo {
	return &regTestRepo{
		users:     map[string]*auth.UserRow{},
		usersByID: map[string]*auth.UserRow{},
		tokens:    map[string]*auth.EmailVerificationTokenRow{},
	}
}

func (r *regTestRepo) FindUserByEmail(_ context.Context, email string) (*auth.UserRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, errRegTestNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *regTestRepo) FindUserByID(_ context.Context, id string) (*auth.UserRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return nil, errRegTestNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *regTestRepo) CreateSession(_ context.Context, userID, tokenHash string, expiresAt time.Time) (*auth.SessionRow, error) {
	return &auth.SessionRow{
		Id: uuid.New(), UserId: uuid.MustParse(userID), TokenHash: tokenHash,
		Provider: "password", ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func (r *regTestRepo) FindSession(_ context.Context, _ string) (*auth.SessionRow, error) {
	return nil, errRegTestNotFound
}

func (r *regTestRepo) DeleteSession(_ context.Context, _ string) error { return nil }

func (r *regTestRepo) FindUserPhotoKeyByID(_ context.Context, _ string) (string, error) {
	return "", pgx.ErrNoRows
}

func (r *regTestRepo) UpdateUserPhoto(_ context.Context, _, _ string) error { return nil }

func (r *regTestRepo) EraseUser(_ context.Context, _ string) error { return nil }

func (r *regTestRepo) ExportUserData(_ context.Context, _ string) (*auth.ExportData, error) {
	return &auth.ExportData{}, nil
}

func (r *regTestRepo) CreateUnverifiedUser(_ context.Context, name, email, passwordHash string) (*auth.UserRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	normalized := strings.ToLower(strings.TrimSpace(email))
	if _, exists := r.users[normalized]; exists {
		return nil, auth.ErrEmailTaken
	}
	u := &auth.UserRow{
		Id:           uuid.New(),
		Name:         name,
		Email:        normalized,
		AvatarColor:  "#6366f1",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	r.users[normalized] = u
	r.usersByID[u.Id.String()] = u
	cp := *u
	return &cp, nil
}

func (r *regTestRepo) MarkEmailVerified(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[userID]
	if !ok {
		return errRegTestNotFound
	}
	now := time.Now()
	u.EmailVerifiedAt = &now
	return nil
}

func (r *regTestRepo) CreateEmailVerificationToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[tokenHash] = &auth.EmailVerificationTokenRow{
		Id: uuid.New(), UserId: uuid.MustParse(userID), TokenHash: tokenHash,
		ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}
	return nil
}

func (r *regTestRepo) FindEmailVerificationToken(_ context.Context, tokenHash string) (*auth.EmailVerificationTokenRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[tokenHash]
	if !ok || t.ConsumedAt != nil || time.Now().After(t.ExpiresAt) {
		return nil, pgx.ErrNoRows
	}
	cp := *t
	return &cp, nil
}

func (r *regTestRepo) ConsumeEmailVerificationToken(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[tokenHash]
	if !ok || t.ConsumedAt != nil {
		return pgx.ErrNoRows
	}
	now := time.Now()
	t.ConsumedAt = &now
	return nil
}

func (r *regTestRepo) countUsers() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func newRegTestService(t *testing.T, repo *regTestRepo, fm *mailer.FakeMailer, ttl time.Duration, enabled bool) *auth.Service {
	t.Helper()
	svc, err := auth.NewService(repo, storage.NewFakeStore(), "", "", 24*time.Hour, auth.RegistrationConfig{
		Mailer:                  fm,
		PublicBaseURL:           "https://example.com",
		EmailVerificationTTL:    ttl,
		SelfRegistrationEnabled: enabled,
	}, nil)
	require.NoError(t, err)
	return svc
}

// tokenFromLink extracts the raw token from a "https://.../verify-email/<token>" link.
func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	const marker = "/verify-email/"
	idx := strings.Index(link, marker)
	require.GreaterOrEqual(t, idx, 0, "link must contain /verify-email/: %s", link)
	return link[idx+len(marker):]
}

// ─── Register: enumeration-safety matrix ───────────────────────────────────

func TestService_Register_EmailAvailable_CreatesUnverifiedUserAndSendsEmail(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	err := svc.Register(context.Background(), "New@Example.com", "longenoughpassword")
	require.NoError(t, err)

	user, err := repo.FindUserByEmail(context.Background(), "new@example.com")
	require.NoError(t, err)
	assert.Nil(t, user.EmailVerifiedAt, "newly registered account must start unverified")
	assert.NotEmpty(t, user.PasswordHash)

	to, link := fm.LastSentTo()
	assert.Equal(t, "new@example.com", to)
	assert.Contains(t, link, "/verify-email/")
}

func TestService_Register_EmailTaken_Verified_LeavesAccountUntouched(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	existing, err := repo.CreateUnverifiedUser(context.Background(), "Existing", "taken@example.com", "original-hash")
	require.NoError(t, err)
	require.NoError(t, repo.MarkEmailVerified(context.Background(), existing.Id.String()))

	err = svc.Register(context.Background(), "taken@example.com", "attackerpassword123")
	require.NoError(t, err, "Register must report generic success even though the email is already taken")

	refetched, err := repo.FindUserByEmail(context.Background(), "taken@example.com")
	require.NoError(t, err)
	assert.Equal(t, "original-hash", refetched.PasswordHash, "an already-verified account's password must never be overwritten")
	assert.Equal(t, 0, fm.SentCount(), "no email is sent for an already-verified account in this design")
}

func TestService_Register_EmailTaken_Unverified_ResendsFreshTokenWithoutOverwritingPassword(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	_, err := repo.CreateUnverifiedUser(context.Background(), "Pending", "pending@example.com", "original-hash")
	require.NoError(t, err)

	err = svc.Register(context.Background(), "pending@example.com", "attackerpassword123")
	require.NoError(t, err)

	refetched, err := repo.FindUserByEmail(context.Background(), "pending@example.com")
	require.NoError(t, err)
	assert.Equal(t, "original-hash", refetched.PasswordHash, "a pending registration's password must never be overwritten by a re-registration attempt")
	assert.Equal(t, 1, fm.SentCount(), "a fresh verification token must be (re)sent for a still-pending registration")

	to, _ := fm.LastSentTo()
	assert.Equal(t, "pending@example.com", to)
}

func TestService_Register_AllThreeCases_ReturnIdenticalSuccess(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	verified, err := repo.CreateUnverifiedUser(context.Background(), "V", "verified@example.com", "hash")
	require.NoError(t, err)
	require.NoError(t, repo.MarkEmailVerified(context.Background(), verified.Id.String()))
	_, err = repo.CreateUnverifiedUser(context.Background(), "P", "pending@example.com", "hash")
	require.NoError(t, err)

	// Available, already-verified, and still-pending must all report the
	// exact same nil-error "generic success" outcome to the caller -- the
	// handler builds one fixed response body from this, regardless of case.
	assert.NoError(t, svc.Register(context.Background(), "available@example.com", "longenoughpassword"))
	assert.NoError(t, svc.Register(context.Background(), "verified@example.com", "longenoughpassword"))
	assert.NoError(t, svc.Register(context.Background(), "pending@example.com", "longenoughpassword"))
}

func TestService_Register_DisabledFeatureFlag_RejectsAndCreatesNoAccount(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, false)

	err := svc.Register(context.Background(), "new@example.com", "longenoughpassword")
	require.ErrorIs(t, err, auth.ErrSelfRegistrationDisabled)
	assert.Equal(t, 0, repo.countUsers())
	assert.Equal(t, 0, fm.SentCount())
}

func TestService_Register_OverLongPassword_Rejected(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	err := svc.Register(context.Background(), "new@example.com", strings.Repeat("x", 73))
	require.ErrorIs(t, err, auth.ErrPasswordTooLong)
	assert.Equal(t, 0, repo.countUsers())
}

// ─── VerifyEmail ────────────────────────────────────────────────────────────

func TestService_VerifyEmail_ValidToken_VerifiesAndEstablishesSession(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	require.NoError(t, svc.Register(context.Background(), "new@example.com", "longenoughpassword"))
	_, link := fm.LastSentTo()
	rawToken := tokenFromLink(t, link)

	sessionToken, user, err := svc.VerifyEmail(context.Background(), rawToken)
	require.NoError(t, err)
	assert.NotEmpty(t, sessionToken)
	assert.Equal(t, "new@example.com", user.Email)

	refetched, err := repo.FindUserByEmail(context.Background(), "new@example.com")
	require.NoError(t, err)
	require.NotNil(t, refetched.EmailVerifiedAt, "account must be marked verified after a successful VerifyEmail")
}

func TestService_VerifyEmail_TokenReuse_Rejected(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	require.NoError(t, svc.Register(context.Background(), "new@example.com", "longenoughpassword"))
	_, link := fm.LastSentTo()
	rawToken := tokenFromLink(t, link)

	_, _, err := svc.VerifyEmail(context.Background(), rawToken)
	require.NoError(t, err)

	_, _, err = svc.VerifyEmail(context.Background(), rawToken)
	require.ErrorIs(t, err, auth.ErrInvalidVerificationToken, "a consumed token must not verify a second time")
}

func TestService_VerifyEmail_ExpiredToken_Rejected(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	// A negative TTL means the token is already expired the instant it's issued.
	svc := newRegTestService(t, repo, fm, -time.Minute, true)

	require.NoError(t, svc.Register(context.Background(), "new@example.com", "longenoughpassword"))
	_, link := fm.LastSentTo()
	rawToken := tokenFromLink(t, link)

	_, _, err := svc.VerifyEmail(context.Background(), rawToken)
	require.ErrorIs(t, err, auth.ErrInvalidVerificationToken)
}

func TestService_VerifyEmail_UnknownToken_Rejected(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	_, _, err := svc.VerifyEmail(context.Background(), "totally-bogus-token")
	require.ErrorIs(t, err, auth.ErrInvalidVerificationToken)
}

// ─── ResendVerification ─────────────────────────────────────────────────────

func TestService_ResendVerification_UniformAcrossAllAccountStates(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	verified, err := repo.CreateUnverifiedUser(context.Background(), "V", "verified@example.com", "hash")
	require.NoError(t, err)
	require.NoError(t, repo.MarkEmailVerified(context.Background(), verified.Id.String()))
	_, err = repo.CreateUnverifiedUser(context.Background(), "P", "pending@example.com", "hash")
	require.NoError(t, err)

	// No account at all: no-op, no error.
	require.NoError(t, svc.ResendVerification(context.Background(), "nobody@example.com"))
	assert.Equal(t, 0, fm.SentCount(), "resend for a nonexistent account must not send mail")

	// Already verified: no-op, no error, no mail.
	require.NoError(t, svc.ResendVerification(context.Background(), "verified@example.com"))
	assert.Equal(t, 0, fm.SentCount(), "resend for an already-verified account must not send mail")

	// Still pending: succeeds and actually sends a fresh token.
	require.NoError(t, svc.ResendVerification(context.Background(), "pending@example.com"))
	assert.Equal(t, 1, fm.SentCount(), "resend for a still-unverified account must send a fresh verification email")
}

// ─── Login gating on verification status ───────────────────────────────────

func TestService_Login_UnverifiedAccount_Rejected(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	require.NoError(t, svc.Register(context.Background(), "new@example.com", "longenoughpassword"))

	_, _, err := svc.Login(context.Background(), "new@example.com", "longenoughpassword")
	require.ErrorIs(t, err, auth.ErrEmailNotVerified, "correct credentials for an unverified account must be rejected distinctly from wrong credentials")
}

func TestService_Login_VerifiedAccount_Succeeds(t *testing.T) {
	t.Parallel()
	repo := newRegTestRepo()
	fm := mailer.NewFakeMailer(nil)
	svc := newRegTestService(t, repo, fm, time.Hour, true)

	require.NoError(t, svc.Register(context.Background(), "new@example.com", "longenoughpassword"))
	_, link := fm.LastSentTo()
	_, _, err := svc.VerifyEmail(context.Background(), tokenFromLink(t, link))
	require.NoError(t, err)

	token, user, err := svc.Login(context.Background(), "new@example.com", "longenoughpassword")
	require.NoError(t, err, "login must succeed once the account is verified")
	assert.NotEmpty(t, token)
	assert.Equal(t, "new@example.com", user.Email)
}
