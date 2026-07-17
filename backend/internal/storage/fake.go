package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// FakeStore is an in-memory ObjectStore for tests. It is safe for concurrent
// use. PresignGet returns a fake URL of the form "fake://<bucket>/<key>"
// rather than a real signed URL, sufficient for asserting redirect targets in
// handler tests without a real S3/MinIO backend.
type FakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

// NewFakeStore creates an empty FakeStore.
func NewFakeStore() *FakeStore {
	return &FakeStore{objects: make(map[string][]byte)}
}

func (f *FakeStore) Put(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("storage.FakeStore.Put: %w", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = b
	return nil
}

func (f *FakeStore) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.objects[key]; !ok {
		return "", ErrNotFound
	}
	return "fake://object/" + key, nil
}

func (f *FakeStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	return nil
}

// Has reports whether key is currently stored, for test assertions.
func (f *FakeStore) Has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok
}

// Get returns the stored bytes for key, for test assertions.
func (f *FakeStore) Get(key string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return bytes.Clone(f.objects[key])
}
