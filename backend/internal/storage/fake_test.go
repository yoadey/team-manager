package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/storage"
)

func TestFakeStorePutPresignGetDelete(t *testing.T) {
	ctx := context.Background()
	s := storage.NewFakeStore()

	_, err := s.PresignGet(ctx, "teams/t1/photo", time.Minute)
	require.ErrorIs(t, err, storage.ErrObjectNotFound)

	require.NoError(t, s.Put(ctx, "teams/t1/photo", []byte("hello"), "image/jpeg"))
	assert.True(t, s.Has("teams/t1/photo"))
	data, ok := s.Get("teams/t1/photo")
	require.True(t, ok)
	assert.Equal(t, []byte("hello"), data)

	url, err := s.PresignGet(ctx, "teams/t1/photo", time.Minute)
	require.NoError(t, err)
	assert.Contains(t, url, "teams/t1/photo")

	require.NoError(t, s.Delete(ctx, "teams/t1/photo"))
	assert.False(t, s.Has("teams/t1/photo"))
	_, err = s.PresignGet(ctx, "teams/t1/photo", time.Minute)
	require.ErrorIs(t, err, storage.ErrObjectNotFound)

	// Deleting a non-existent key is not an error.
	require.NoError(t, s.Delete(ctx, "nope"))
}

func TestFakeStorePutOverwrites(t *testing.T) {
	ctx := context.Background()
	s := storage.NewFakeStore()

	require.NoError(t, s.Put(ctx, "k", []byte("v1"), "image/jpeg"))
	require.NoError(t, s.Put(ctx, "k", []byte("v2"), "image/png"))
	data, ok := s.Get("k")
	require.True(t, ok)
	assert.Equal(t, []byte("v2"), data)
}
