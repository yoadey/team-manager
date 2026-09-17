package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIDCState_RoundTrip(t *testing.T) {
	t.Parallel()
	state, err := newOIDCState()
	require.NoError(t, err)

	assert.NotEmpty(t, state.State)
	assert.NotEmpty(t, state.Nonce)
	assert.NotEmpty(t, state.Verifier)
	assert.NotEqual(t, state.State, state.Nonce, "state and nonce must be independent secrets")

	payload, err := state.encode()
	require.NoError(t, err)

	decoded, err := decodeOIDCState(payload)
	require.NoError(t, err)
	assert.Equal(t, state.State, decoded.State)
	assert.Equal(t, state.Nonce, decoded.Nonce)
	assert.Equal(t, state.Verifier, decoded.Verifier)
}

// The cookie's own Max-Age is only a client-side hint -- a client is free to
// keep sending an expired cookie -- so the deadline has to be re-checked
// server-side.
func TestDecodeOIDCState_RejectsExpired(t *testing.T) {
	t.Parallel()
	state, err := newOIDCState()
	require.NoError(t, err)
	state.IssuedAt = time.Now().Add(-OIDCStateTTL - time.Minute).Unix()

	payload, err := state.encode()
	require.NoError(t, err)

	_, err = decodeOIDCState(payload)
	assert.ErrorIs(t, err, ErrOIDCState)
}

func TestDecodeOIDCState_RejectsMalformedOrIncomplete(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"empty":          "",
		"not json":       "nonsense",
		"missing fields": `{"s":"abc"}`,
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeOIDCState(payload)
			assert.ErrorIs(t, err, ErrOIDCState)
		})
	}
}

func TestFlexBool(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw  string
		want bool
	}{
		{`true`, true},
		{`false`, false},
		{`"true"`, true},
		{`"false"`, false},
		// Anything unexpected must decode as false: for email_verified that is
		// the safe direction.
		{`null`, false},
		{`1`, false},
		{`"yes"`, false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			var b flexBool
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &b))
			assert.Equal(t, tc.want, bool(b))
		})
	}
}

// users.name is NOT NULL, so a provisioned account must never end up nameless
// however sparse the provider's claims are.
func TestDisplayName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, given, family, email, want string
	}{
		{"Full Name", "Given", "Family", "a@b.com", "Full Name"},
		{"  ", "Given", "Family", "a@b.com", "Given Family"},
		{"", "Given", "", "a@b.com", "Given"},
		{"", "", "", "local.part@b.com", "local.part"},
		{"", "", "", "", "Unbenannt"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, displayName(tc.name, tc.given, tc.family, tc.email))
		})
	}
}

// A provider-supplied name reaches users.name without ever passing through a
// request handler, so it is the one name in the system that no validate.Name
// call has seen.
func TestSanitizeOIDCName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"keeps an ordinary name", "Erika Mustermann", "Erika Mustermann"},
		{"trims surrounding whitespace", "  Erika  ", "Erika"},
		{"strips a null byte Postgres would reject", "Eri\x00ka", "Erika"},
		{"strips newlines and other control characters", "Erika\r\n\tMustermann", "ErikaMustermann"},
		{"falls back when nothing usable is left", "\x00\x01\x02", oidcFallbackName},
		{"falls back on an empty claim", "   ", oidcFallbackName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, sanitizeOIDCName(tc.in))
		})
	}
}

// Length is counted in runes, not bytes: a multi-byte name must not be cut
// mid-character, and the result must stay inside validate.Name's bound.
func TestSanitizeOIDCName_ClampsLength(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("ä", maxOIDCNameLen+50)

	got := sanitizeOIDCName(long)

	assert.Equal(t, maxOIDCNameLen, utf8.RuneCountInString(got))
	assert.True(t, utf8.ValidString(got), "truncation must not split a rune")
}
