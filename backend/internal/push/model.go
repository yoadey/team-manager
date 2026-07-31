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

// CategoryPreferences is a member's per-team Web Push opt-out, one boolean
// per notification category. It gates delivery independently of (and in
// addition to) the recipient's module-read permission: permission decides
// what a member is allowed to see at all, preference decides what they've
// asked to be interrupted for.
type CategoryPreferences struct {
	Attendance bool
	Events     bool
	News       bool
	Polls      bool
	Absence    bool
	// EventReminderEnabled and EventReminderHoursBefore configure a
	// separate, time-triggered push sent shortly before an upcoming event
	// starts -- independent of the Events category above, which only
	// covers event lifecycle notifications (created/updated/cancelled/...).
	// See jobs.EventReminderWorker.
	EventReminderEnabled     bool
	EventReminderHoursBefore int16
}

// DefaultEventReminderHoursBefore is the lead time a member gets reminded
// before an event's start when they've never customized it.
const DefaultEventReminderHoursBefore = 6

// MinEventReminderHoursBefore and MaxEventReminderHoursBefore bound the
// caller-configurable lead time -- enforced both by the DB CHECK constraint
// on push_preferences.event_reminder_hours_before and by request validation
// in the handler, so an out-of-range value is rejected with 400 rather than
// a generic 500 from the DB constraint violation.
const (
	MinEventReminderHoursBefore = 1
	MaxEventReminderHoursBefore = 72
)

// DefaultCategoryPreferences returns every category enabled -- the implicit
// preference for a member who has never called SetPreferences, so a missing
// push_preferences row behaves exactly like today's unconditional delivery.
func DefaultCategoryPreferences() CategoryPreferences {
	return CategoryPreferences{
		Attendance:               true,
		Events:                   true,
		News:                     true,
		Polls:                    true,
		Absence:                  true,
		EventReminderEnabled:     true,
		EventReminderHoursBefore: DefaultEventReminderHoursBefore,
	}
}

// Allows reports whether category is enabled. An empty category (a
// notification type NotificationCategory doesn't recognize) is always
// allowed, mirroring notifications.HasReadAccess's treatment of an empty
// module -- there's nothing to gate.
func (p CategoryPreferences) Allows(category string) bool {
	switch category {
	case "attendance":
		return p.Attendance
	case "events":
		return p.Events
	case "news":
		return p.News
	case "polls":
		return p.Polls
	case "absence":
		return p.Absence
	case "":
		return true
	default:
		// Fail closed on an unrecognized category, mirroring
		// notifications.HasReadAccess's default case -- a future category
		// NotificationCategory returns that this switch hasn't been taught
		// about yet must not silently bypass the preference gate.
		return false
	}
}

// NotificationCategory returns the push-preference category a
// gen.NotificationType string belongs to. Deliberately independent of
// notifications.NotificationModule: that function collapses "attendance"
// into the "events" RBAC module (there's no separate events:attendance
// permission), but the preference UI wants attendance responses
// separately toggleable from event-lifecycle changes, matching the
// notification feed's own "attendance" vs "events" filter chips
// (frontend NotificationsSheet.tsx).
func NotificationCategory(notifType string) string {
	switch notifType {
	case "attendance":
		return "attendance"
	case "event_created", "event_updated", "event_cancelled", "event_reactivated", "event_deleted":
		return "events"
	case "news":
		return "news"
	case "poll":
		return "polls"
	case "absence":
		return "absence"
	default:
		return ""
	}
}
