// Package pagination provides helpers for limit/offset and keyset (cursor)
// pagination.
package pagination

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

// ErrInvalidCursor is returned by Paginator.Decode when the token is not a
// valid cursor (bad base64 or JSON). Handlers should map it to a 400, not a 500.
var ErrInvalidCursor = errors.New("invalid cursor")

// ParseLimit applies the default (50) and cap (500) to an optional limit param.
func ParseLimit(limit *int) int {
	l := DefaultLimit
	if limit != nil && *limit > 0 {
		l = *limit
		if l > MaxLimit {
			l = MaxLimit
		}
	}
	return l
}

// Parse extracts limit and offset from optional pointer params, applying
// defaults (limit=50, offset=0) and capping limit at MaxLimit.
func Parse(limit, offset *int) (l, o int) {
	l, o = DefaultLimit, 0
	if limit != nil && *limit > 0 {
		l = *limit
		if l > MaxLimit {
			l = MaxLimit
		}
	}
	if offset != nil && *offset > 0 {
		o = *offset
	}
	return l, o
}

// Paginator is a cursor encoder/decoder that optionally signs cursors with
// HMAC-SHA256 to prevent clients from crafting arbitrary cursor values.
//
// When HMACKey is nil or empty, cursors are plain, unsigned base64url tokens
// (dev mode). When HMACKey is set, cursors are formatted as:
//
//	base64url(json) + "." + hex(hmac-sha256(base64url(json), key))
//
// A cursor with an invalid or missing HMAC signature is treated as absent
// (ok=false, nil error) rather than an error, so tampered cursors degrade
// safely to "start from the beginning" rather than leaking internals.
type Paginator struct {
	HMACKey []byte
}

// New returns a Paginator. key may be nil or empty for unsigned (dev) mode.
func New(key []byte) *Paginator {
	return &Paginator{HMACKey: key}
}

// sign computes hex(hmac-sha256(payload, key)).
func (p *Paginator) sign(payload string) string {
	mac := hmac.New(sha256.New, p.HMACKey)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// Encode serializes v into an opaque cursor token. When HMACKey is set the
// token is HMAC-signed; otherwise it is plain base64url.
func (p *Paginator) Encode(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("pagination.Paginator.Encode: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	if len(p.HMACKey) == 0 {
		return payload, nil
	}
	return payload + "." + p.sign(payload), nil
}

// Decode parses a cursor token produced by Encode into dst.
//   - Empty token → (false, nil): start from the beginning.
//   - Invalid HMAC (when key is set) → (false, nil): degrade safely; do NOT
//     return an error so a tampered cursor is indistinguishable from "no cursor".
//   - Bad base64 or JSON (only reached when no HMAC key is set, or after
//     successful HMAC verification) → (false, ErrInvalidCursor).
func (p *Paginator) Decode(token string, dst any) (bool, error) {
	if token == "" {
		return false, nil
	}

	payload := token
	if len(p.HMACKey) > 0 {
		// Split at the last dot: payload.signature
		idx := strings.LastIndex(token, ".")
		if idx < 0 {
			// No dot → unsigned token presented to signed endpoint; degrade safely.
			return false, nil
		}
		payload = token[:idx]
		sig := token[idx+1:]
		expected := p.sign(payload)
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			// Tampered or forged cursor; degrade safely.
			return false, nil
		}
	}

	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return false, fmt.Errorf("pagination.Paginator.Decode: %w: %v", ErrInvalidCursor, err) //nolint:errorlint // err is context only; ErrInvalidCursor is the matchable sentinel
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return false, fmt.Errorf("pagination.Paginator.Decode: %w: %v", ErrInvalidCursor, err) //nolint:errorlint // err is context only; ErrInvalidCursor is the matchable sentinel
	}
	return true, nil
}
