package validate_test

import (
	"math"
	"strings"
	"testing"

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

func TestName(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validate.Name("Alice"))
	assert.Error(t, validate.Name(""))
	assert.Error(t, validate.Name(strings.Repeat("x", 256)))
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
	assert.NoError(t, validate.PositiveAmount(0.01, "amount"))
	assert.NoError(t, validate.PositiveAmount(50, "amount"))
	assert.Error(t, validate.PositiveAmount(0, "amount"))
	assert.Error(t, validate.PositiveAmount(-5, "amount"))
	assert.Error(t, validate.PositiveAmount(math.NaN(), "amount"))
	assert.Error(t, validate.PositiveAmount(math.Inf(1), "amount"))
}
