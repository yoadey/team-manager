package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
)

func TestEraseAccount(t *testing.T) {
	const accountEmail = "member@example.com"
	userByID := func(_ context.Context, _ string) (*auth.UserRow, error) {
		return &auth.UserRow{Email: accountEmail}, nil
	}

	t.Run("erases when the confirmation email matches (case-insensitive)", func(t *testing.T) {
		erasedID := ""
		repo := &mockRepo{
			userByID: userByID,
			eraseUser: func(_ context.Context, userID string) error {
				erasedID = userID
				return nil
			},
		}
		svc := newTestService(t, repo)

		require.NoError(t, svc.EraseAccount(context.Background(), "user-1", "  Member@Example.com "))
		assert.Equal(t, "user-1", erasedID, "EraseUser should be called with the user id")
	})

	t.Run("rejects a mismatched confirmation email without erasing", func(t *testing.T) {
		repo := &mockRepo{
			userByID: userByID,
			eraseUser: func(_ context.Context, _ string) error {
				t.Fatal("EraseUser must not be called when the email does not match")
				return nil
			},
		}
		svc := newTestService(t, repo)

		err := svc.EraseAccount(context.Background(), "user-1", "someone-else@example.com")
		assert.ErrorIs(t, err, auth.ErrErasureConfirmation)
	})

	t.Run("propagates ErrSoleSettingsAdmin from the repository", func(t *testing.T) {
		repo := &mockRepo{
			userByID: userByID,
			eraseUser: func(_ context.Context, _ string) error {
				return auth.ErrSoleSettingsAdmin
			},
		}
		svc := newTestService(t, repo)

		err := svc.EraseAccount(context.Background(), "user-1", "member@example.com")
		assert.ErrorIs(t, err, auth.ErrSoleSettingsAdmin)
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

		err := svc.EraseAccount(context.Background(), "user-1", "member@example.com")
		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})
}
