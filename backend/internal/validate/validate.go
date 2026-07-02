// Package validate provides lightweight field-level validation helpers.
// Each function returns a descriptive error when the value violates constraints,
// or nil on success. All functions are pure and safe for concurrent use.
package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	maxNameLen     = 255
	maxTextLen     = 10_000
	minPasswordLen = 8
	maxPasswordLen = 128
)

// Sentinel errors for static analysis compliance.
var (
	ErrEmailRequired     = errors.New("email is required")
	ErrEmailInvalid      = errors.New("email is not a valid address")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong   = errors.New("password must not exceed 128 characters")
	ErrFieldRequired     = errors.New("is required")
	ErrFieldTooLong      = errors.New("is too long")
	ErrAmountNotPositive = errors.New("must be greater than zero")
)

// Email checks that s is a valid RFC 5322 e-mail address.
func Email(s string) error {
	if strings.TrimSpace(s) == "" {
		return ErrEmailRequired
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return ErrEmailInvalid
	}
	return nil
}

// RequireNonEmpty returns an error when s is blank.
func RequireNonEmpty(s, field string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s %w", field, ErrFieldRequired)
	}
	return nil
}

// MaxLen returns an error when s exceeds n UTF-8 characters.
func MaxLen(s string, n int, field string) error {
	if utf8.RuneCountInString(s) > n {
		return fmt.Errorf("%s %w", field, ErrFieldTooLong)
	}
	return nil
}

// Name validates a display-name field: non-empty, max 255 characters.
func Name(s string) error {
	if err := RequireNonEmpty(s, "name"); err != nil {
		return err
	}
	return MaxLen(s, maxNameLen, "name")
}

// Text validates a free-text field: non-empty, max 10 000 characters.
func Text(s, field string) error {
	if err := RequireNonEmpty(s, field); err != nil {
		return err
	}
	return MaxLen(s, maxTextLen, field)
}

// PositiveAmount returns an error when amount (an integer amount in cents) is
// not positive (zero or negative).
func PositiveAmount(amount int64, field string) error {
	if amount <= 0 {
		return fmt.Errorf("%s %w", field, ErrAmountNotPositive)
	}
	return nil
}

// PasswordStrength checks that s is within the accepted length window.
// A minimum of 8 and a maximum of 128 characters is enforced; further
// complexity requirements are deliberately avoided to follow NIST SP 800-63B.
func PasswordStrength(s string) error {
	n := utf8.RuneCountInString(s)
	if n < minPasswordLen {
		return ErrPasswordTooShort
	}
	if n > maxPasswordLen {
		return ErrPasswordTooLong
	}
	return nil
}
