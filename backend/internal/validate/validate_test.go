package validate_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/validate"
)

func TestEmail(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		ok    bool
	}{
		{"user@example.com", true},
		{"User Name <user@example.com>", true},
		{"", false},
		{"   ", false},
		{"notanemail", false},
		{"missing@", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			err := validate.Email(tc.input)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRequireNonEmpty(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.RequireNonEmpty("hello", "field"))
	assert.Error(t, validate.RequireNonEmpty("", "field"))
	assert.Error(t, validate.RequireNonEmpty("   ", "field"))
}

func TestMaxLen(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.MaxLen("abc", 3, "f"))
	assert.Error(t, validate.MaxLen("abcd", 3, "f"))
	// Emoji = 1 rune, not 4 bytes
	assert.NoError(t, validate.MaxLen("abc😀", 4, "f"))
	assert.Error(t, validate.MaxLen("abc😀!", 4, "f"))
}

func TestMaxLen_RejectsNullByte(t *testing.T) {
	t.Parallel()
	err := validate.MaxLen("abc\x00def", 100, "f")
	require.Error(t, err)
	assert.ErrorIs(t, err, validate.ErrFieldNullByte)
}

func TestName(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.Name("Alice"))
	assert.Error(t, validate.Name(""))
	assert.Error(t, validate.Name(strings.Repeat("x", 256)))
}

func TestBirthday(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.Birthday(time.Date(1990, 5, 1, 0, 0, 0, 0, time.UTC)))
	assert.NoError(t, validate.Birthday(time.Now()))
	assert.Error(t, validate.Birthday(time.Now().AddDate(0, 0, 1)), "future birthday must be rejected")
	assert.Error(t, validate.Birthday(time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)), "birthday before 1900 must be rejected")
}

func TestPasswordStrength(t *testing.T) {
	t.Parallel()
	assert.Error(t, validate.PasswordStrength("short"))
	assert.NoError(t, validate.PasswordStrength("longenough"))
	assert.Error(t, validate.PasswordStrength(strings.Repeat("x", 129)))
	assert.NoError(t, validate.PasswordStrength(strings.Repeat("x", 128)))
}

func TestPositiveAmount(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.PositiveAmount(1, "amount"))
	assert.NoError(t, validate.PositiveAmount(5000, "amount"))
	assert.Error(t, validate.PositiveAmount(0, "amount"))
	assert.Error(t, validate.PositiveAmount(-5, "amount"))
}

func TestPositiveAmount_RejectsAboveMax(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.PositiveAmount(100_000_000, "amount"))
	assert.ErrorIs(t, validate.PositiveAmount(100_000_001, "amount"), validate.ErrAmountTooLarge)
	assert.Error(t, validate.PositiveAmount(math.MaxInt64, "amount"))
}
