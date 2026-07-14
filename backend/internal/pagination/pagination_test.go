package pagination_test

import (
	"strings"
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

// ─── Paginator tests ─────────────────────────────────────────────────────────

func TestPaginator_NoKey_RoundTrip(t *testing.T) {
	t.Parallel()

	type cur struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	pager := pagination.New(nil)
	token, err := pager.Encode(cur{ID: 42, Name: "bob"})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Without a key the token should be plain base64 (no dot-separated HMAC).
	if strings.Contains(token, ".") {
		t.Errorf("unsigned token should not contain a dot, got %q", token)
	}

	var got cur
	ok, err := pager.Decode(token, &got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !ok || got.ID != 42 || got.Name != "bob" {
		t.Fatalf("round-trip mismatch: ok=%v got=%+v", ok, got)
	}
}

func TestPaginator_NoKey_EmptyIsFirstPage(t *testing.T) {
	t.Parallel()

	pager := pagination.New(nil)
	var dst struct{ ID int }
	ok, err := pager.Decode("", &dst)
	if err != nil {
		t.Fatalf("Decode(\"\"): %v", err)
	}
	if ok {
		t.Fatal("empty token should return ok=false")
	}
}

func TestPaginator_WithKey_RoundTrip(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	pager := pagination.New(key)

	type cur struct {
		Date string `json:"date"`
		ID   string `json:"id"`
	}
	token, err := pager.Encode(cur{Date: "2025-01-15", ID: "abc123"})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// With a key the token should contain exactly one dot.
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("signed token must contain exactly one dot, got %q", token)
	}

	var got cur
	ok, err := pager.Decode(token, &got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !ok || got.Date != "2025-01-15" || got.ID != "abc123" {
		t.Fatalf("round-trip mismatch: ok=%v got=%+v", ok, got)
	}
}

func TestPaginator_WithKey_TamperedCursorDegradesSafely(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	pager := pagination.New(key)

	type cur struct {
		ID int `json:"id"`
	}
	token, err := pager.Encode(cur{ID: 1})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Tamper: replace payload part with a different value.
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected signed token, got %q", token)
	}

	// Build a tampered token: swap in a different payload with the original sig.
	tamperedPayload, _ := pagination.New(nil).Encode(cur{ID: 999})
	tamperedToken := tamperedPayload + "." + parts[1]

	var dst cur
	ok, err := pager.Decode(tamperedToken, &dst)
	// Must degrade safely: no error, ok=false (treat as "start from beginning").
	if err != nil {
		t.Fatalf("tampered cursor should not return error, got: %v", err)
	}
	if ok {
		t.Fatalf("tampered cursor should return ok=false, but decoded to %+v", dst)
	}
}

func TestPaginator_WithKey_UnsignedCursorDegradesSafely(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	pager := pagination.New(key)

	// Build an unsigned token (as if from the old code or a different instance).
	type cur struct {
		ID int `json:"id"`
	}
	unsignedToken, _ := pagination.New(nil).Encode(cur{ID: 1})

	var dst cur
	ok, err := pager.Decode(unsignedToken, &dst)
	if err != nil {
		t.Fatalf("unsigned cursor should not return error, got: %v", err)
	}
	if ok {
		t.Fatalf("unsigned cursor presented to signed paginator should return ok=false")
	}
}

func TestPaginator_WithKey_EmptyIsFirstPage(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	pager := pagination.New(key)

	var dst struct{ ID int }
	ok, err := pager.Decode("", &dst)
	if err != nil {
		t.Fatalf("Decode(\"\"): %v", err)
	}
	if ok {
		t.Fatal("empty token should return ok=false")
	}
}

func TestPaginator_WithKey_WrongKey(t *testing.T) {
	t.Parallel()

	key1 := make([]byte, 32)
	for i := range key1 {
		key1[i] = 0xAA
	}
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = 0xBB
	}

	pager1 := pagination.New(key1)
	pager2 := pagination.New(key2)

	type cur struct {
		ID int `json:"id"`
	}
	token, err := pager1.Encode(cur{ID: 7})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// pager2 uses a different key — should degrade safely.
	var dst cur
	ok, err := pager2.Decode(token, &dst)
	if err != nil {
		t.Fatalf("wrong-key decode should not return error, got: %v", err)
	}
	if ok {
		t.Fatalf("wrong-key decode should return ok=false, decoded to %+v", dst)
	}
}
