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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/image/draw"
)

// authRepo is the interface the Service relies on. A *Repository satisfies it.
type authRepo interface {
	FindUserByEmail(ctx context.Context, email string) (*UserRow, error)
	FindUserByID(ctx context.Context, id string) (*UserRow, error)
	CreateSession(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) (*SessionRow, error)
	FindSession(ctx context.Context, tokenHash string) (*SessionRow, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	UpdateUserPhoto(ctx context.Context, userID string, data []byte, mime string) error
}

// Service implements authentication logic.
type Service struct {
	repo          authRepo
	jwtPrivateKey *rsa.PrivateKey
	jwtPublicKey  *rsa.PublicKey
	sessionTTL    time.Duration
}

// NewService parses RSA keys from PEM strings and returns a Service.
// If privateKeyPEM and publicKeyPEM are both empty, a throwaway RSA-2048 key
// pair is generated for use in dev/test mode.
func NewService(repo authRepo, privateKeyPEM, publicKeyPEM string, sessionTTL time.Duration) (*Service, error) {
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
			return nil, errors.New("auth.NewService: failed to decode private key PEM")
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
			return nil, errors.New("auth.NewService: private key is not RSA")
		}

		pubBlock, _ := pem.Decode([]byte(publicKeyPEM))
		if pubBlock == nil {
			return nil, errors.New("auth.NewService: failed to decode public key PEM")
		}
		pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("auth.NewService: parse public key: %w", err)
		}
		pub, ok = pubKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("auth.NewService: public key is not RSA")
		}
	}

	return &Service{
		repo:          repo,
		jwtPrivateKey: priv,
		jwtPublicKey:  pub,
		sessionTTL:    sessionTTL,
	}, nil
}

// Login verifies email+password, creates a session, and returns a signed JWT.
func (s *Service) Login(ctx context.Context, email, password string) (token string, user *UserRow, err error) {
	user, err = s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Generate a random token and derive its SHA-256 hash for storage.
	rawToken, tokenHash, err := generateTokenAndHash()
	if err != nil {
		return "", nil, fmt.Errorf("auth.Service.Login: generate token: %w", err)
	}

	expiresAt := time.Now().Add(s.sessionTTL)
	_, err = s.repo.CreateSession(ctx, user.Id.String(), tokenHash, expiresAt)
	if err != nil {
		return "", nil, fmt.Errorf("auth.Service.Login: create session: %w", err)
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
		return "", nil, fmt.Errorf("auth.Service.Login: sign JWT: %w", err)
	}

	return signed, user, nil
}

// ValidateToken verifies the JWT signature, checks the session in DB, and returns the user.
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*UserRow, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtPublicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth.Service.ValidateToken: %w", err)
	}

	// Derive the session hash from the embedded raw token (JWT ID).
	rawToken := claims.RegisteredClaims.ID
	if rawToken == "" {
		return nil, errors.New("auth.Service.ValidateToken: missing jti claim")
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
	return s.repo.DeleteSession(ctx, tokenHash)
}

// HashPassword hashes a plain-text password using bcrypt cost 12.
func (s *Service) HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("auth.Service.HashPassword: %w", err)
	}
	return string(b), nil
}

// UpdatePhoto resizes the image to at most 800×800 px, stores it, and returns
// the refreshed user row.
func (s *Service) UpdatePhoto(ctx context.Context, userID string, data []byte, mime string) (*UserRow, error) {
	resized, err := resizeImage(data, mime)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: resize: %w", err)
	}

	if err := s.repo.UpdateUserPhoto(ctx, userID, resized, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: store: %w", err)
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.UpdatePhoto: refresh user: %w", err)
	}
	return user, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// generateTokenAndHash produces a cryptographically random hex token and its
// SHA-256 hex hash.
func generateTokenAndHash() (rawToken, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
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

// resizeImage decodes a JPEG or PNG, scales it proportionally so neither
// dimension exceeds maxPhotoDim, and re-encodes as JPEG.
func resizeImage(data []byte, mime string) ([]byte, error) {
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
	newW, newH := w, h
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
