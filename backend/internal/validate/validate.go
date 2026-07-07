// Package validate provides lightweight field-level validation helpers.
// Each function returns a descriptive error when the value violates constraints,
// or nil on success. All functions are pure and safe for concurrent use.
package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxNameLen     = 255
	maxTextLen     = 10_000
	minPasswordLen = 8
	maxPasswordLen = 128
	// maxAmountCents caps a single monetary amount at 1,000,000.00 (in the
	// club's currency, stored as integer cents). Generous for any club
	// transaction, while keeping SUM(amount) aggregates (see
	// finances.Repository.SumTransactions, ListOpenPenaltiesByUser) far below
	// the int64/Postgres BIGINT range even across many rows — without a cap, a
	// single amount near math.MaxInt64 makes those ::BIGINT-cast aggregates
	// overflow and error out, permanently breaking the team's finance overview.
	maxAmountCents = 100_000_000
	// maxUUIDItems caps role-ID-array-shaped request fields (roleIds,
	// nominatedRoleIds, reasonVisibilityRoleIds). Generous for any real
	// club's role list, while preventing a caller from submitting an
	// absurdly large array — e.g. members.SetRoles processes one
	// INSERT per ID sequentially while holding the team's exclusive
	// advisory lock, so an unbounded array would block every other admin
	// mutation on that team for the duration.
	maxUUIDItems = 200
)

// Sentinel errors for static analysis compliance.
var (
	ErrEmailRequired      = errors.New("email is required")
	ErrEmailInvalid       = errors.New("email is not a valid address")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must not exceed 128 characters")
	ErrFieldRequired      = errors.New("is required")
	ErrFieldTooLong       = errors.New("is too long")
	ErrAmountNotPositive  = errors.New("must be greater than zero")
	ErrAmountTooLarge     = errors.New("exceeds the maximum allowed amount")
	ErrTimeOfDayInvalid   = errors.New("must be a 24-hour HH:MM time")
	ErrTooManyItems       = errors.New("has too many items")
	ErrFieldNullByte      = errors.New("must not contain a null byte")
	ErrBirthdayOutOfRange = errors.New("must be between 1900-01-01 and today")
)

// timeOfDayRE matches a 24-hour "HH:MM" time-of-day string, the format the
// events repository passes straight into a Postgres ::time cast.
var timeOfDayRE = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// TimeOfDay validates a 24-hour "HH:MM" time-of-day string (e.g. "09:30").
// Empty strings are rejected — callers should skip validation for optional
// fields that are nil rather than pass "".
func TimeOfDay(s, field string) error {
	if !timeOfDayRE.MatchString(s) {
		return fmt.Errorf("%s %w", field, ErrTimeOfDayInvalid)
	}
	return nil
}

// minBirthday is the earliest accepted birthday -- generous enough for any
// real club member while catching obviously-wrong input (e.g. an
// off-by-a-century typo, or a client bug sending the Unix epoch).
var minBirthday = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

// Birthday validates that t is not in the future and not before minBirthday.
// Unlike every other free-text/date field in the codebase, birthday had no
// range validation at all -- a members:write holder could set an arbitrary,
// nonsensical date with no server-side rejection.
func Birthday(t time.Time) error {
	if t.Before(minBirthday) || t.After(time.Now()) {
		return ErrBirthdayOutOfRange
	}
	return nil
}

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

// MaxLen returns an error when s exceeds n UTF-8 characters or contains a
// null byte. A null byte is valid UTF-8 but Postgres text/varchar columns
// reject it outright, which would otherwise surface as an unhandled 500 at
// the DB layer instead of a clean 400 here.
func MaxLen(s string, n int, field string) error {
	if strings.IndexByte(s, 0) != -1 {
		return fmt.Errorf("%s %w", field, ErrFieldNullByte)
	}
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
// not positive (zero or negative), or exceeds maxAmountCents.
func PositiveAmount(amount int64, field string) error {
	if amount <= 0 {
		return fmt.Errorf("%s %w", field, ErrAmountNotPositive)
	}
	if amount > maxAmountCents {
		return fmt.Errorf("%s %w", field, ErrAmountTooLarge)
	}
	return nil
}

// UUIDItems returns an error when n (the length of a role-ID-shaped array
// field) exceeds maxUUIDItems.
func UUIDItems(n int, field string) error {
	if n > maxUUIDItems {
		return fmt.Errorf("%s %w", field, ErrTooManyItems)
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
