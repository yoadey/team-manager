package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/yoadey/team-manager/backend/internal/auth"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	userByEmail    func(ctx context.Context, email string) (*auth.UserRow, error)
	userByID       func(ctx context.Context, id string) (*auth.UserRow, error)
	createSess     func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*auth.SessionRow, error)
	findSess       func(ctx context.Context, tokenHash string) (*auth.SessionRow, error)
	deleteSess     func(ctx context.Context, tokenHash string) error
	updatePhoto    func(ctx context.Context, userID string, data []byte, mime string) error
	userPhotoByID  func(ctx context.Context, id string) ([]byte, error)
	eraseUser      func(ctx context.Context, userID string) error
	exportUserData func(ctx context.Context, userID string) (*auth.ExportData, error)
}

func (m *mockRepo) FindUserByEmail(ctx context.Context, email string) (*auth.UserRow, error) {
	return m.userByEmail(ctx, email)
}

func (m *mockRepo) FindUserByID(ctx context.Context, id string) (*auth.UserRow, error) {
	return m.userByID(ctx, id)
}

func (m *mockRepo) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*auth.SessionRow, error) {
	return m.createSess(ctx, userID, tokenHash, expiresAt)
}

func (m *mockRepo) FindSession(ctx context.Context, tokenHash string) (*auth.SessionRow, error) {
	return m.findSess(ctx, tokenHash)
}

func (m *mockRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	return m.deleteSess(ctx, tokenHash)
}

func (m *mockRepo) UpdateUserPhoto(ctx context.Context, userID string, data []byte, mime string) error {
	return m.updatePhoto(ctx, userID, data, mime)
}

func (m *mockRepo) FindUserPhotoByID(ctx context.Context, id string) ([]byte, error) {
	if m.userPhotoByID != nil {
		return m.userPhotoByID(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) EraseUser(ctx context.Context, userID string) error {
	return m.eraseUser(ctx, userID)
}

func (m *mockRepo) ExportUserData(ctx context.Context, userID string) (*auth.ExportData, error) {
	if m.exportUserData != nil {
		return m.exportUserData(ctx, userID)
	}
	return &auth.ExportData{}, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

func newTestService(t *testing.T, repo *mockRepo) *auth.Service {
	t.Helper()
	svc, err := auth.NewService(repo, "", "", 24*time.Hour)
	require.NoError(t, err)
	return svc
}

func makeUserWithPassword(t *testing.T, password string) *auth.UserRow {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return &auth.UserRow{
		Id:           uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:         "Test User",
		Email:        "test@example.com",
		AvatarColor:  "#6366f1",
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
}

func makeSession() *auth.SessionRow {
	return &auth.SessionRow{
		Id:        uuid.New(),
		UserId:    uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		TokenHash: "somehash",
		Provider:  "password",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_Login_Success(t *testing.T) {
	t.Parallel()

	user := makeUserWithPassword(t, "secret")
	sess := makeSession()

	repo := &mockRepo{
		userByEmail: func(_ context.Context, email string) (*auth.UserRow, error) {
			assert.Equal(t, "test@example.com", email)
			return user, nil
		},
		createSess: func(_ context.Context, _, _ string, _ time.Time) (*auth.SessionRow, error) {
			return sess, nil
		},
		findSess: func(_ context.Context, hash string) (*auth.SessionRow, error) {
			return sess, nil
		},
		userByID: func(_ context.Context, id string) (*auth.UserRow, error) {
			return user, nil
		},
	}

	svc := newTestService(t, repo)
	ctx := context.Background()

	token, gotUser, err := svc.Login(ctx, "test@example.com", "secret")
	require.NoError(t, err)
	assert.NotEmpty(t, token, "token should not be empty")
	assert.Equal(t, "Test User", gotUser.Name)

	// The token must be a valid JWT that ValidateToken accepts.
	validatedUser, err := svc.ValidateToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, user.Id, validatedUser.Id)
}

func TestService_Login_WrongPassword(t *testing.T) {
	t.Parallel()

	user := makeUserWithPassword(t, "correct")
	repo := &mockRepo{
		userByEmail: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return user, nil
		},
	}

	svc := newTestService(t, repo)
	_, _, err := svc.Login(context.Background(), "test@example.com", "wrong")
	assert.Error(t, err)
}

func TestService_Login_UserNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		userByEmail: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return nil, errors.New("no rows")
		},
	}

	svc := newTestService(t, repo)
	_, _, err := svc.Login(context.Background(), "nobody@example.com", "pass")
	assert.Error(t, err)
}

func TestService_ValidateToken_Expired(t *testing.T) {
	t.Parallel()

	// Use a very short TTL so the token expires instantly.
	repo := &mockRepo{}
	svc, err := auth.NewService(repo, "", "", -time.Second)
	require.NoError(t, err)

	user := makeUserWithPassword(t, "pw")
	repo.userByEmail = func(_ context.Context, _ string) (*auth.UserRow, error) { return user, nil }
	repo.createSess = func(_ context.Context, _, _ string, _ time.Time) (*auth.SessionRow, error) {
		return makeSession(), nil
	}

	token, _, err := svc.Login(context.Background(), "test@example.com", "pw")
	require.NoError(t, err)

	_, err = svc.ValidateToken(context.Background(), token)
	assert.Error(t, err, "expired token should be rejected")
}

func TestService_HashPassword(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &mockRepo{})
	hash, err := svc.HashPassword("mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("mypassword"))
	assert.NoError(t, err, "hash should verify against original password")
}
