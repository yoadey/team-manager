package push

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionRow mirrors the push_subscriptions DB table.
type SubscriptionRow struct {
	Id         uuid.UUID
	UserId     uuid.UUID
	Endpoint   string
	P256dh     string
	AuthKey    string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// SubscriptionForUser pairs a Subscription (and its row ID, needed to prune
// it later) with the user it belongs to -- returned by
// Repository.ListForTeamExcludingUser, which joins across team membership,
// so callers don't need a second round-trip to know who each subscription
// is for.
type SubscriptionForUser struct {
	Id           uuid.UUID
	UserId       uuid.UUID
	Subscription Subscription
}
