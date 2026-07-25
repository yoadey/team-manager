package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// FakeStore is an in-memory ObjectStore for tests. PresignGet returns a
// deterministic "fake://" URL rather than a real signed one.
type FakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	types   map[string]string
}

// NewFakeStore creates an empty FakeStore.
func NewFakeStore() *FakeStore {
	return &FakeStore{objects: map[string][]byte{}, types: map[string]string{}}
}

// Put stores a copy of data under key.
func (f *FakeStore) Put(_ context.Context, key string, data []byte, contentType string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	f.objects[key] = cp
	f.types[key] = contentType
	return nil
}

// PresignGet returns a fake URL identifying key, or ErrObjectNotFound if it
// was never Put (or has since been Deleted).
func (f *FakeStore) PresignGet(_ context.Context, key string, ttl time.Duration) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.objects[key]; !ok {
		return "", ErrObjectNotFound
	}
	return fmt.Sprintf("fake://storage/%s?ttl=%s", key, ttl), nil
}

// Delete removes key. Deleting a non-existent key is not an error.
func (f *FakeStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	delete(f.types, key)
	return nil
}

// Has reports whether key currently exists. Test helper.
func (f *FakeStore) Has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok
}

// Contents returns the bytes stored under key, and whether it exists. Test
// helper -- unlike Get (the ObjectStore interface method), this bypasses the
// ErrObjectNotFound/io.ReadCloser contract so tests can assert on raw bytes
// directly.
func (f *FakeStore) Contents(key string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.objects[key]
	return d, ok
}

// Get returns the bytes stored under key as a stream, along with its stored
// content type. Returns ErrObjectNotFound if key was never Put (or has since
// been Deleted).
func (f *FakeStore) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.objects[key]
	if !ok {
		return nil, "", ErrObjectNotFound
	}
	cp := make([]byte, len(d))
	copy(cp, d)
	return io.NopCloser(bytes.NewReader(cp)), f.types[key], nil
}
