package pagination_test

import (
	"errors"
	"testing"

	"github.com/yoadey/team-manager/backend/internal/pagination"
)

func TestParseLimit(t *testing.T) {
	t.Parallel()

	n := 10
	zero := 0
	huge := 10000
	cases := []struct {
		name string
		in   *int
		want int
	}{
		{"nil -> default", nil, pagination.DefaultLimit},
		{"zero -> default", &zero, pagination.DefaultLimit},
		{"explicit", &n, 10},
		{"capped at max", &huge, pagination.MaxLimit},
	}
	for _, tc := range cases {
		if got := pagination.ParseLimit(tc.in); got != tc.want {
			t.Errorf("%s: ParseLimit() = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestEncodeDecodeCursor_RoundTrip(t *testing.T) {
	t.Parallel()

	type cur struct {
		N string `json:"n"`
		I int    `json:"i"`
	}
	token, err := pagination.EncodeCursor(cur{N: "alice", I: 7})
	if err != nil {
		t.Fatalf("EncodeCursor: %v", err)
	}

	var got cur
	ok, err := pagination.DecodeCursor(token, &got)
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if !ok || got.N != "alice" || got.I != 7 {
		t.Fatalf("round-trip mismatch: ok=%v got=%+v", ok, got)
	}
}

func TestDecodeCursor_EmptyIsFirstPage(t *testing.T) {
	t.Parallel()

	var dst struct{}
	ok, err := pagination.DecodeCursor("", &dst)
	if err != nil {
		t.Fatalf("DecodeCursor(\"\"): %v", err)
	}
	if ok {
		t.Fatal("empty token should report ok=false (start from beginning)")
	}
}

func TestDecodeCursor_InvalidReturnsSentinel(t *testing.T) {
	t.Parallel()

	var dst struct {
		N string `json:"n"`
	}
	// "!!!" is not valid base64url.
	if _, err := pagination.DecodeCursor("!!!", &dst); !errors.Is(err, pagination.ErrInvalidCursor) {
		t.Fatalf("bad base64: got %v, want ErrInvalidCursor", err)
	}
	// Valid base64 of non-JSON bytes -> JSON unmarshal failure.
	if _, err := pagination.DecodeCursor("Zm9v", &dst); !errors.Is(err, pagination.ErrInvalidCursor) {
		t.Fatalf("bad json: got %v, want ErrInvalidCursor", err)
	}
}
