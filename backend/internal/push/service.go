package push

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// subscriptionRepo is the interface Service relies on.
type subscriptionRepo interface {
	Upsert(ctx context.Context, userID uuid.UUID, sub Subscription) error
	Delete(ctx context.Context, userID uuid.UUID, endpoint string) error
}

// Service implements push-subscription management business logic.
type Service struct {
	repo subscriptionRepo
}

// NewService creates a new Service.
func NewService(repo subscriptionRepo) *Service {
	return &Service{repo: repo}
}

// Register upserts userID's subscription.
func (s *Service) Register(ctx context.Context, userID uuid.UUID, sub Subscription) error {
	if err := s.repo.Upsert(ctx, userID, sub); err != nil {
		return fmt.Errorf("push.Service.Register: %w", err)
	}
	return nil
}

// Unregister removes userID's subscription for endpoint, if any.
func (s *Service) Unregister(ctx context.Context, userID uuid.UUID, endpoint string) error {
	if err := s.repo.Delete(ctx, userID, endpoint); err != nil {
		return fmt.Errorf("push.Service.Unregister: %w", err)
	}
	return nil
}
