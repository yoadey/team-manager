package auth_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/storage"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	userByEmail      func(ctx context.Context, email string) (*auth.UserRow, error)
	userByID         func(ctx context.Context, id string) (*auth.UserRow, error)
	createSess       func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*auth.SessionRow, error)
	findSess         func(ctx context.Context, tokenHash string) (*auth.SessionRow, error)
	deleteSess       func(ctx context.Context, tokenHash string) error
	updatePhoto      func(ctx context.Context, userID, objectKey string) error
	userPhotoKeyByID func(ctx context.Context, id string) (string, error)
	eraseUser        func(ctx context.Context, userID string) error
	exportUserData   func(ctx context.Context, userID string) (*auth.ExportData, error)
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

func (m *mockRepo) UpdateUserPhoto(ctx context.Context, userID, objectKey string) error {
	return m.updatePhoto(ctx, userID, objectKey)
}

func (m *mockRepo) FindUserPhotoKeyByID(ctx context.Context, id string) (string, error) {
	if m.userPhotoKeyByID != nil {
		return m.userPhotoKeyByID(ctx, id)
	}
	return "", pgx.ErrNoRows
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
	svc, err := auth.NewService(repo, storage.NewFakeStore(), "", "", 24*time.Hour)
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

func TestService_Login_RejectsOverLengthPassword(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		userByEmail: func(_ context.Context, _ string) (*auth.UserRow, error) {
			t.Fatal("repository must not be consulted for an over-length password")
			return nil, nil
		},
	}
	svc := newTestService(t, repo)

	_, _, err := svc.Login(context.Background(), "test@example.com", strings.Repeat("a", 73))
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials, "an over-length password is rejected before any DB lookup")
}

func TestService_HashPassword_RejectsOverLength(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &mockRepo{})

	_, err := svc.HashPassword(strings.Repeat("x", 73))
	assert.ErrorIs(t, err, auth.ErrPasswordTooLong, "a >72-byte password must be rejected, not silently truncated")

	_, err = svc.HashPassword(strings.Repeat("x", 72))
	assert.NoError(t, err, "a 72-byte password is at the limit and accepted")
}

func TestHashEmailForAudit(t *testing.T) {
	t.Parallel()

	h := auth.HashEmailForAudit("User@Example.com")
	assert.Equal(t, auth.HashEmailForAudit("user@example.com"), h, "hashing is case-insensitive (lowercased)")
	assert.Len(t, h, 64, "SHA-256 hex digest is 64 chars")
	assert.NotContains(t, h, "example", "the digest must not contain the plaintext address")
	assert.NotEqual(t, auth.HashEmailForAudit("other@example.com"), h, "different emails hash differently")
}

func TestService_ValidateToken_Expired(t *testing.T) {
	t.Parallel()

	// Use a very short TTL so the token expires instantly.
	repo := &mockRepo{}
	svc, err := auth.NewService(repo, storage.NewFakeStore(), "", "", -time.Second)
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

// Regression test: ValidateToken parsed with no jwt.WithExpirationRequired()
// option. golang-jwt v5's default validator only checks the exp claim IF
// PRESENT -- a token with no exp claim at all passes validation
// unconditionally, never expiring by JWT validation alone. Login always
// sets ExpiresAt, so this was latent, not exploitable via the normal login
// path -- but any future code path minting a JWT without setting exp (or a
// signing-key compromise letting an attacker forge one) would produce a
// token this service accepts forever. This mints a token directly (bypassing
// Login) with every other claim Login sets except ExpiresAt, signed with
// the same key the service validates against, and asserts it's rejected.
func TestService_ValidateToken_RejectsTokenWithNoExpiryClaim(t *testing.T) {
	t.Parallel()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := string(pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: mustMarshalPKCS8(t, privKey),
	}))
	pubPEM := string(pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: mustMarshalPKIXPublicKey(t, &privKey.PublicKey),
	}))

	repo := &mockRepo{
		findSess: func(_ context.Context, _ string) (*auth.SessionRow, error) {
			return &auth.SessionRow{UserId: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}, nil
		},
		userByID: func(_ context.Context, id string) (*auth.UserRow, error) {
			return &auth.UserRow{Id: uuid.MustParse(id)}, nil
		},
	}
	svc, err := auth.NewService(repo, storage.NewFakeStore(), privPEM, pubPEM, 24*time.Hour)
	require.NoError(t, err)

	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ID:       "raw-token-no-exp",
			// ExpiresAt deliberately omitted.
		},
		UserId: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}
	noExpToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
	require.NoError(t, err)

	_, err = svc.ValidateToken(context.Background(), noExpToken)
	assert.Error(t, err, "a token with no exp claim at all must be rejected, not accepted forever")
}

func mustMarshalPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	b, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return b
}

func mustMarshalPKIXPublicKey(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	b, err := x509.MarshalPKIXPublicKey(key)
	require.NoError(t, err)
	return b
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

// fixedJPEG returns a minimal valid 2x2 JPEG for image-processing tests.
func fixedJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func TestService_UpdatePhoto_UploadsAndStoresKey(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	var storedKey string
	repo := &mockRepo{
		updatePhoto: func(_ context.Context, uid, objectKey string) error {
			assert.Equal(t, userID, uid)
			storedKey = objectKey
			return nil
		},
		userByID: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return &auth.UserRow{Id: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}, nil
		},
	}
	store := storage.NewFakeStore()
	svc, err := auth.NewService(repo, store, "", "", 24*time.Hour)
	require.NoError(t, err)

	_, err = svc.UpdatePhoto(context.Background(), userID, fixedJPEG(t), "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, "users/"+userID+"/photo", storedKey)
	data, ok := store.Get(storedKey)
	require.True(t, ok, "resized image must be uploaded to the object store")
	assert.NotEmpty(t, data)
}

func TestService_GetMyPhotoURL_ReturnsPresignedURL(t *testing.T) {
	t.Parallel()

	userID := "user-1"
	key := "users/" + userID + "/photo"
	repo := &mockRepo{
		userPhotoKeyByID: func(_ context.Context, _ string) (string, error) {
			return key, nil
		},
	}
	store := storage.NewFakeStore()
	require.NoError(t, store.Put(context.Background(), key, []byte{1, 2, 3}, "image/jpeg"))
	svc, err := auth.NewService(repo, store, "", "", 24*time.Hour)
	require.NoError(t, err)

	url, err := svc.GetMyPhotoURL(context.Background(), userID)
	require.NoError(t, err)
	assert.Contains(t, url, key)
}

func TestService_GetMyPhotoURL_NoPhotoReturnsErrNoRows(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		userPhotoKeyByID: func(_ context.Context, _ string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}
	svc := newTestService(t, repo)

	_, err := svc.GetMyPhotoURL(context.Background(), "user-1")
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

// EraseAccount (GDPR Art. 17) must delete the underlying object store photo,
// not just null the DB reference to it -- otherwise the image bytes survive
// the "erasure" in the object store indefinitely.
func TestService_EraseAccount_DeletesStoredPhoto(t *testing.T) {
	t.Parallel()

	const accountEmail = "member@example.com"
	userID := "user-1"
	key := "users/" + userID + "/photo"

	repo := &mockRepo{
		userByID: func(_ context.Context, _ string) (*auth.UserRow, error) {
			return &auth.UserRow{Email: accountEmail}, nil
		},
		userPhotoKeyByID: func(_ context.Context, _ string) (string, error) {
			return key, nil
		},
		eraseUser: func(_ context.Context, _ string) error { return nil },
	}
	store := storage.NewFakeStore()
	require.NoError(t, store.Put(context.Background(), key, []byte{1, 2, 3}, "image/jpeg"))
	svc, err := auth.NewService(repo, store, "", "", 24*time.Hour)
	require.NoError(t, err)

	require.NoError(t, svc.EraseAccount(context.Background(), userID, accountEmail))
	assert.False(t, store.Has(key), "erasure must delete the underlying photo object")
}
