// Package pagination provides helpers for limit/offset and keyset (cursor)
// pagination.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

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

// EncodeCursor serializes a keyset cursor value into an opaque base64url token.
func EncodeCursor(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("pagination.EncodeCursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeCursor parses an opaque token produced by EncodeCursor into dst. An
// empty token returns (false, nil), meaning "start from the beginning".
func DecodeCursor(token string, dst any) (bool, error) {
	if token == "" {
		return false, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false, fmt.Errorf("pagination.DecodeCursor: %w", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return false, fmt.Errorf("pagination.DecodeCursor: %w", err)
	}
	return true, nil
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
