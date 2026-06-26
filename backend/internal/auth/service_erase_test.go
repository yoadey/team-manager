package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/yoadey/team-manager/backend/internal/auth"
)

func TestEraseAccount(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-horse"), bcrypt.MinCost)
	require.NoError(t, err)

	t.Run("erases on correct password", func(t *testing.T) {
		erasedID := ""
		repo := &mockRepo{
			userByID: func(_ context.Context, _ string) (*auth.UserRow, error) {
				return &auth.UserRow{PasswordHash: string(hash)}, nil
			},
			eraseUser: func(_ context.Context, userID string) error {
				erasedID = userID
				return nil
			},
		}
		svc := newTestService(t, repo)

		require.NoError(t, svc.EraseAccount(context.Background(), "user-1", "correct-horse"))
		assert.Equal(t, "user-1", erasedID, "EraseUser should be called with the user id")
	})

	t.Run("rejects a wrong password without erasing", func(t *testing.T) {
		repo := &mockRepo{
			userByID: func(_ context.Context, _ string) (*auth.UserRow, error) {
				return &auth.UserRow{PasswordHash: string(hash)}, nil
			},
			eraseUser: func(_ context.Context, _ string) error {
				t.Fatal("EraseUser must not be called when the password is wrong")
				return nil
			},
		}
		svc := newTestService(t, repo)

		err := svc.EraseAccount(context.Background(), "user-1", "wrong")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("rejects an unknown user without erasing", func(t *testing.T) {
		repo := &mockRepo{
			userByID: func(_ context.Context, _ string) (*auth.UserRow, error) {
				return nil, errors.New("not found")
			},
			eraseUser: func(_ context.Context, _ string) error {
				t.Fatal("EraseUser must not be called when the user is missing")
				return nil
			},
		}
		svc := newTestService(t, repo)

		err := svc.EraseAccount(context.Background(), "user-1", "whatever")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})
}
