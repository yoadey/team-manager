package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"

	"github.com/yoadey/team-manager/backend/internal/mailer"
	"github.com/yoadey/team-manager/backend/internal/storage"
)

// Sentinel errors for the auth package.
var (
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrDecodePrivateKeyPEM      = errors.New("auth.NewService: failed to decode private key PEM")
	ErrPrivateKeyNotRSA         = errors.New("auth.NewService: private key is not RSA")
	ErrDecodePublicKeyPEM       = errors.New("auth.NewService: failed to decode public key PEM")
	ErrPublicKeyNotRSA          = errors.New("auth.NewService: public key is not RSA")
	ErrMissingJTIClaim          = errors.New("auth.Service.ValidateToken: missing jti claim")
	ErrUnexpectedSigningMethod  = errors.New("auth.Service.ValidateToken: unexpected signing method")
	ErrImageTooLarge            = errors.New("auth.resizeImage: image dimensions exceed the allowed maximum")
	ErrErasureConfirmation      = errors.New("auth.Service.EraseAccount: confirmation email does not match account")
	ErrPasswordTooLong          = errors.New("password must be at most 72 bytes")
	ErrEmailNotVerified         = errors.New("email not verified")
	ErrSelfRegistrationDisabled = errors.New("self-registration is disabled")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
)

// dummyPasswordHash is a valid bcrypt hash that Login compares against when no
// user matches the supplied email. Running a bcrypt comparison on both the
// "user found" and "user not found" paths keeps Login's timing roughly constant
// regardless of whether the email exists, mitigating user enumeration.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("user-enumeration-timing-equalizer"), 12)

// authRepo is the interface the Service relies on. A *Repository satisfies it.
type authRepo interface {
	FindUserByEmail(ctx context.Context, email string) (*UserRow, error)
	FindUserByID(ctx context.Context, id string) (*UserRow, error)
	CreateSession(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) (*SessionRow, error)
	FindSession(ctx context.Context, tokenHash string) (*SessionRow, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	FindUserPhotoKeyByID(ctx context.Context, id string) (string, error)
	UpdateUserPhoto(ctx context.Context, userID, objectKey string) error
	EraseUser(ctx context.Context, userID string) error
	ExportUserData(ctx context.Context, userID string) (*ExportData, error)
	CreateUnverifiedUser(ctx context.Context, name, email, passwordHash string) (*UserRow, error)
	MarkEmailVerified(ctx context.Context, userID string) error
	CreateEmailVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	FindEmailVerificationToken(ctx context.Context, tokenHash string) (*EmailVerificationTokenRow, error)
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) error
}

// RegistrationConfig configures self-service registration. Grouped into its
// own struct (unlike NewService's other parameters) because four more
// positional bool/string/duration arguments would be easy to transpose by
// mistake at the call site.
type RegistrationConfig struct {
	Mailer                  mailer.Mailer
	PublicBaseURL           string
	EmailVerificationTTL    time.Duration
	SelfRegistrationEnabled bool
}

// Service implements authentication logic.
type Service struct {
	repo                    authRepo
	store                   storage.ObjectStore
	jwtPrivateKey           *rsa.PrivateKey
	jwtPublicKey            *rsa.PublicKey
	sessionTTL              time.Duration
	mailer                  mailer.Mailer
	publicBaseURL           string
	emailVerificationTTL    time.Duration
	selfRegistrationEnabled bool
	logger                  *slog.Logger
}

// NewService parses RSA keys from PEM strings and returns a Service.
// If privateKeyPEM and publicKeyPEM are both empty, a throwaway RSA-2048 key
// pair is generated for use in dev/test mode. logger defaults to
// slog.Default() when nil (used to log a best-effort warning when sending a
// verification email fails -- see issueVerificationToken).
func NewService(repo authRepo, store storage.ObjectStore, privateKeyPEM, publicKeyPEM string, sessionTTL time.Duration, reg RegistrationConfig, logger *slog.Logger) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var priv *rsa.PrivateKey
	var pub *rsa.PublicKey

	if privateKeyPEM == "" && publicKeyPEM == "" {
		generated, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("auth.NewService: generate dev RSA key: %w", err)
		}
		priv = generated
		pub = &generated.PublicKey
	} else {
		privBlock, _ := pem.Decode([]byte(privateKeyPEM))
		if privBlock == nil {
			return nil, ErrDecodePrivateKeyPEM
		}
		privKey, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
		if err != nil {
			// fallback to PKCS1
			privKey, err = x509.ParsePKCS1PrivateKey(privBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("auth.NewService: parse private key: %w", err)
			}
		}
		var ok bool
		priv, ok = privKey.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrPrivateKeyNotRSA
		}

		pubBlock, _ := pem.Decode([]byte(publicKeyPEM))
		if pubBlock == nil {
			return nil, ErrDecodePublicKeyPEM
		}
		pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("auth.NewService: parse public key: %w", err)
		}
		pub, ok = pubKey.(*rsa.PublicKey)
		if !ok {
			return nil, ErrPublicKeyNotRSA
		}
	}

	return &Service{
		repo:                    repo,
		store:                   store,
		jwtPrivateKey:           priv,
		jwtPublicKey:            pub,
		sessionTTL:              sessionTTL,
		mailer:                  reg.Mailer,
		publicBaseURL:           reg.PublicBaseURL,
		emailVerificationTTL:    reg.EmailVerificationTTL,
		selfRegistrationEnabled: reg.SelfRegistrationEnabled,
		logger:                  logger,
	}, nil
}

// userPhotoKey returns the object store key for userID's profile photo.
func userPhotoKey(userID string) string {
	return "users/" + userID + "/photo"
}

// maxPasswordBytes is bcrypt's hard input limit: bytes beyond 72 are silently
// ignored by the algorithm. Rejecting over-length passwords explicitly (rather
// than letting them be truncated) means a passphrase's tail can never be
// dropped without the user knowing, and a login can't succeed on a truncated
// prefix of a longer stored secret.
const maxPasswordBytes = 72

// Login verifies email+password, creates a session, and returns a signed JWT.
func (s *Service) Login(ctx context.Context, email, password string) (token string, user *UserRow, err error) {
	if len(password) > maxPasswordBytes {
		// Never a valid password (HashPassword rejects the same). Burn a compare
		// so timing doesn't distinguish this from a normal wrong password.
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		return "", nil, ErrInvalidCredentials
	}

	user, err = s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		// Compare against a dummy hash so a missing user takes about as long as a
		// wrong password — otherwise response timing reveals which emails exist.
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		return "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// The password is correct, so checking verification status here can't
	// leak anything about *other* emails' existence -- it only confirms what
	// the caller already knows (that this email+password pair is theirs).
	if user.EmailVerifiedAt == nil {
		return "", nil, ErrEmailNotVerified
	}

	signed, err := s.createSessionAndSign(ctx, user)
	if err != nil {
		return "", nil, fmt.Errorf("auth.Service.Login: %w", err)
	}
	return signed, user, nil
}

// createSessionAndSign creates a DB-backed session for user and returns a
// signed JWT carrying it. Shared by Login and VerifyEmail so a successful
// email verification establishes a session identically to a successful
// password login.
func (s *Service) createSessionAndSign(ctx context.Context, user *UserRow) (string, error) {
	// Generate a random token and derive its SHA-256 hash for storage.
	rawToken, tokenHash, err := generateTokenAndHash()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().Add(s.sessionTTL)
	_, err = s.repo.CreateSession(ctx, user.Id.String(), tokenHash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	// Sign a JWT that carries userId and the raw token (so we can look up the session).
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Id.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        rawToken,
		},
		UserId: user.Id.String(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signed, err := jwtToken.SignedString(s.jwtPrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signed, nil
}

// Register creates a new, unverified account and emails a verification link.
// The response is identical whether the email was available, already
// registered and verified, or already registered and still pending
// verification -- see openspec/changes/self-service-registration/design.md
// for the full enumeration-safety rationale. Callers must treat every nil
// error the same way (a generic "check your email" response); the three
// cases are only distinguished internally.
func (s *Service) Register(ctx context.Context, email, password string) error {
	if !s.selfRegistrationEnabled {
		return ErrSelfRegistrationDisabled
	}
	if len(password) > maxPasswordBytes {
		return ErrPasswordTooLong
	}

	// Hash unconditionally, before any DB lookup, so response timing doesn't
	// distinguish "email available" from "email already taken" -- mirroring
	// Login's dummyPasswordHash trick.
	hash, err := s.HashPassword(password)
	if err != nil {
		return err
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	name := displayNameFromEmail(normalizedEmail)

	user, err := s.repo.CreateUnverifiedUser(ctx, name, normalizedEmail, hash)
	if err != nil {
		if !errors.Is(err, ErrEmailTaken) {
			return fmt.Errorf("auth.Service.Register: %w", err)
		}

		// Case 2/3: an account already exists for this email. Never
		// overwrite it or its password -- only find out whether it's
		// already verified (case 2: leave untouched) or still pending
		// (case 3: resend a fresh verification token).
		existing, findErr := s.repo.FindUserByEmail(ctx, normalizedEmail)
		if findErr != nil {
			return fmt.Errorf("auth.Service.Register: %w", findErr)
		}
		if existing.EmailVerifiedAt != nil {
			return nil
		}
		return s.issueVerificationToken(ctx, existing)
	}

	return s.issueVerificationToken(ctx, user)
}

// displayNameFromEmail derives a placeholder display name from the local
// part of an email address (e.g. "jane.doe" from "jane.doe@example.com").
// Self-registration collects no separate name field; the user can set a
// proper one later via their team member profile.
func displayNameFromEmail(email string) string {
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}

// issueVerificationToken generates a fresh single-use verification token for
// user, persists its hash, and emails the verification link. A failure to
// send the email is logged but not returned as an error -- the token row
// already exists, so POST /auth/resend-verification can recover it, and
// Register's response must stay identical to the success case regardless
// (see enumeration-safety notes on Register).
func (s *Service) issueVerificationToken(ctx context.Context, user *UserRow) error {
	rawToken, tokenHash, err := generateTokenAndHash()
	if err != nil {
		return fmt.Errorf("auth.Service.issueVerificationToken: generate token: %w", err)
	}

	expiresAt := time.Now().Add(s.emailVerificationTTL)
	if err := s.repo.CreateEmailVerificationToken(ctx, user.Id.String(), tokenHash, expiresAt); err != nil {
		return fmt.Errorf("auth.Service.issueVerificationToken: %w", err)
	}

	verifyURL := s.publicBaseURL + "/verify-email/" + rawToken
	if err := s.mailer.SendVerificationEmail(ctx, user.Email, verifyURL); err != nil {
		s.logger.WarnContext(ctx, "issueVerificationToken: send mail failed", "err", err)
	}
	return nil
}

// VerifyEmail consumes a single-use verification token, marks the account
// verified, and returns a session identical in shape to a successful Login
// (so the frontend can reuse its normal post-login bootstrap, including
// redeeming any pending team invite).
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) (token string, user *UserRow, err error) {
	tokenHash := sha256Hex(rawToken)

	tokenRow, err := s.repo.FindEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		return "", nil, ErrInvalidVerificationToken
	}
	if err := s.repo.ConsumeEmailVerificationToken(ctx, tokenHash); err != nil {
		return "", nil, ErrInvalidVerificationToken
	}
	if err := s.repo.MarkEmailVerified(ctx, tokenRow.UserId.String()); err != nil {
		return "", nil, fmt.Errorf("auth.Service.VerifyEmail: %w", err)
	}

	user, err = s.repo.FindUserByID(ctx, tokenRow.UserId.String())
	if err != nil {
		return "", nil, fmt.Errorf("auth.Service.VerifyEmail: %w", err)
	}

	signed, err := s.createSessionAndSign(ctx, user)
	if err != nil {
		return "", nil, fmt.Errorf("auth.Service.VerifyEmail: %w", err)
	}
	return signed, user, nil
}

// ResendVerification always succeeds from the caller's perspective,
// regardless of whether email has no account, an already-verified account,
// or a still-unverified account -- only the last case actually issues and
// sends a new token, mirroring Register's enumeration-safety contract.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user, err := s.repo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil
	}
	if user.EmailVerifiedAt != nil {
		return nil
	}
	return s.issueVerificationToken(ctx, user)
}

// ValidateToken verifies the JWT signature, checks the session in DB, and returns the user.
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*UserRow, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, t.Header["alg"])
		}
		return s.jwtPublicKey, nil
	}, jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("auth.Service.ValidateToken: %w", err)
	}

	// Derive the session hash from the embedded raw token (JWT ID).
	rawToken := claims.ID
	if rawToken == "" {
		return nil, ErrMissingJTIClaim
	}
	tokenHash := sha256Hex(rawToken)

	_, err = s.repo.FindSession(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.ValidateToken: session not found: %w", err)
	}

	user, err := s.repo.FindUserByID(ctx, claims.UserId)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.ValidateToken: user not found: %w", err)
	}

	return user, nil
}

// Logout deletes the session associated with the given raw token (or token hash).
// The caller passes the tokenHash directly.
func (s *Service) Logout(ctx context.Context, tokenHash string) error {
	if err := s.repo.DeleteSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("auth.Service.Logout: %w", err)
	}
	return nil
}

// EraseAccount anonymizes the account (GDPR Art. 17) for an already
// authenticated user. To confirm intent the caller must echo the account's own
// email address — this works for every login method (including OIDC accounts
// that have no password) and guards against an accidental or forged blind
// DELETE. Returns ErrErasureConfirmation when the email does not match.
func (s *Service) EraseAccount(ctx context.Context, userID, confirmEmail string) error {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return ErrInvalidCredentials
	}
	if !strings.EqualFold(strings.TrimSpace(confirmEmail), strings.TrimSpace(user.Email)) {
		return ErrErasureConfirmation
	}

	// Fetch the photo key before erasure nulls it out, so the underlying
	// object can be deleted too -- GDPR Art. 17 erasure must remove the
	// actual image bytes, not just the DB reference to them. Best-effort: a
	// delete failure here doesn't roll back the (already-committed)
	// anonymization, mirroring how upload's orphan cleanup is best-effort too.
	photoKey, keyErr := s.repo.FindUserPhotoKeyByID(ctx, userID)

	if err := s.repo.EraseUser(ctx, userID); err != nil {
		return fmt.Errorf("auth.Service.EraseAccount: %w", err)
	}

	if keyErr == nil && photoKey != "" {
		_ = s.store.Delete(ctx, photoKey)
	}
	return nil
}

// ExportUserData returns the authenticated user's full personal-data export
// (GDPR Art. 15).
func (s *Service) ExportUserData(ctx context.Context, userID string) (*ExportData, error) {
	data, err := s.repo.ExportUserData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.ExportUserData: %w", err)
	}
	return data, nil
}

// HashPassword hashes a plain-text password using bcrypt cost 12. Passwords
// longer than bcrypt's 72-byte limit are rejected rather than silently
// truncated, so the stored hash always covers the full input.
func (s *Service) HashPassword(password string) (string, error) {
	if len(password) > maxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("auth.Service.HashPassword: %w", err)
	}
	return string(b), nil
}

// HashEmailForAudit returns a one-way SHA-256 hex digest of the lowercased
// email, used in audit-log attributes instead of the plaintext address. This
// keeps repeated attempts for the same address correlatable (identical input →
// identical digest) without retaining the address itself, so GDPR erasure and
// the audit log's retention window don't leave plaintext PII behind.
func HashEmailForAudit(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(email)))
	return hex.EncodeToString(sum[:])
}

// UpdatePhoto resizes the image to at most 800×800 px, uploads it to the
// object store, and returns the refreshed user row. Upload order is S3 put
// before the DB write; if the DB write fails, the just-uploaded object is
// deleted (best-effort) so it doesn't linger orphaned.
func (s *Service) UpdatePhoto(ctx context.Context, userID string, data []byte, mime string) (*UserRow, error) {
	resized, err := resizeImage(data, mime)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: resize: %w", err)
	}

	key := userPhotoKey(userID)
	if err := s.store.Put(ctx, key, resized, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: upload: %w", err)
	}

	if err := s.repo.UpdateUserPhoto(ctx, userID, key); err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: store: %w", err)
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: refresh user: %w", err)
	}
	return user, nil
}

// GetMyPhotoURL returns a short-lived presigned URL for userID's photo, or
// pgx.ErrNoRows if the user has no photo set.
func (s *Service) GetMyPhotoURL(ctx context.Context, userID string) (string, error) {
	key, err := s.repo.FindUserPhotoKeyByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("auth.Service.GetMyPhotoURL: %w", err)
	}
	url, err := s.store.PresignGet(ctx, key, storage.PresignTTL)
	if err != nil {
		return "", fmt.Errorf("auth.Service.GetMyPhotoURL: %w", err)
	}
	return url, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// generateTokenAndHash produces a cryptographically random hex token and its
// SHA-256 hex hash.
func generateTokenAndHash() (rawToken, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth.generateTokenAndHash: %w", err)
	}
	rawToken = hex.EncodeToString(b)
	tokenHash = sha256Hex(rawToken)
	return rawToken, tokenHash, nil
}

// sha256Hex returns the hex-encoded SHA-256 digest of s.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

const maxPhotoDim = 800

// maxDecodePixels caps the total pixel count of an uploaded image before it is
// fully decoded, bounding peak memory against decompression-bomb inputs. 50 MP
// comfortably covers legitimate phone-camera photos.
const maxDecodePixels = 50_000_000

// resizeImage decodes a JPEG or PNG, scales it proportionally so neither
// dimension exceeds maxPhotoDim, and re-encodes as JPEG.
func resizeImage(data []byte, mime string) ([]byte, error) {
	// Read the header first (cheap) and reject oversized images before a full
	// decode. A small compressed file can declare enormous dimensions and blow
	// up memory when fully decoded (a "decompression bomb"); the 4 MB request
	// body limit does not bound the decoded pixel count.
	var cfg image.Config
	var cfgErr error
	switch mime {
	case "image/png":
		cfg, cfgErr = png.DecodeConfig(bytes.NewReader(data))
	default:
		cfg, cfgErr = jpeg.DecodeConfig(bytes.NewReader(data))
	}
	if cfgErr != nil {
		return nil, fmt.Errorf("decode image config: %w", cfgErr)
	}
	if cfg.Width*cfg.Height > maxDecodePixels {
		return nil, fmt.Errorf("%w (%dx%d)", ErrImageTooLarge, cfg.Width, cfg.Height)
	}

	var src image.Image
	var decodeErr error

	switch mime {
	case "image/png":
		src, decodeErr = png.Decode(bytes.NewReader(data))
	default:
		src, decodeErr = jpeg.Decode(bytes.NewReader(data))
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("decode image: %w", decodeErr)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxPhotoDim && h <= maxPhotoDim {
		// No resize needed — still re-encode as JPEG for consistency.
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encode image: %w", err)
		}
		return buf.Bytes(), nil
	}

	// Compute new dimensions preserving aspect ratio.
	var newW, newH int
	if w > h {
		newH = h * maxPhotoDim / w
		newW = maxPhotoDim
	} else {
		newW = w * maxPhotoDim / h
		newH = maxPhotoDim
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode resized image: %w", err)
	}
	return buf.Bytes(), nil
}
