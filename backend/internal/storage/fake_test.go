package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestFakeStore_PutPresignDelete(t *testing.T) {
	ctx := context.Background()
	f := NewFakeStore()

	if _, err := f.PresignGet(ctx, "teams/t1/photo", time.Minute); err != ErrNotFound {
		t.Fatalf("PresignGet before Put: got err %v, want ErrNotFound", err)
	}

	if err := f.Put(ctx, "teams/t1/photo", strings.NewReader("jpeg-bytes"), 10, "image/jpeg"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !f.Has("teams/t1/photo") {
		t.Fatal("Has: expected key to be present after Put")
	}
	if got := string(f.Get("teams/t1/photo")); got != "jpeg-bytes" {
		t.Fatalf("Get: got %q, want %q", got, "jpeg-bytes")
	}

	url, err := f.PresignGet(ctx, "teams/t1/photo", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet after Put: %v", err)
	}
	if !strings.Contains(url, "teams/t1/photo") {
		t.Fatalf("PresignGet URL %q does not reference the key", url)
	}

	if err := f.Delete(ctx, "teams/t1/photo"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if f.Has("teams/t1/photo") {
		t.Fatal("Has: expected key to be gone after Delete")
	}

	// Deleting an absent key is not an error.
	if err := f.Delete(ctx, "teams/t1/photo"); err != nil {
		t.Fatalf("Delete (already absent): %v", err)
	}
}
