package notifications

import (
	"time"

	"github.com/google/uuid"
)

// NotificationRow mirrors the DB notifications table joined with actor user info.
type NotificationRow struct {
	Id         uuid.UUID
	TeamId     uuid.UUID
	Type       string
	ActorId    *uuid.UUID
	Status     *string
	Title      *string
	EventId    *uuid.UUID
	EventTitle *string
	EventDate  *time.Time
	Note       *string
	CreatedAt  time.Time
	// Joined from users (actor)
	ActorName  *string
	ActorColor *string
	PhotoData  []byte
	// Computed
	Unread bool
}

// NotifSeenRow mirrors the DB notif_seen table.
type NotifSeenRow struct {
	TeamId uuid.UUID
	UserId uuid.UUID
	SeenAt time.Time
}
