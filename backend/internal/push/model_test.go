package push_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yoadey/team-manager/backend/internal/push"
)

func TestNotificationCategory(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"attendance":        "attendance",
		"event_created":     "events",
		"event_updated":     "events",
		"event_cancelled":   "events",
		"event_reactivated": "events",
		"event_deleted":     "events",
		"news":              "news",
		"poll":              "polls",
		"absence":           "absence",
		"something_unknown": "",
	}
	for notifType, want := range cases {
		assert.Equal(t, want, push.NotificationCategory(notifType), "notifType=%s", notifType)
	}
}

func TestDefaultCategoryPreferences_AllowsEverything(t *testing.T) {
	t.Parallel()

	p := push.DefaultCategoryPreferences()
	for _, category := range []string{"attendance", "events", "news", "polls", "absence"} {
		assert.True(t, p.Allows(category), "category=%s", category)
	}
}

func TestCategoryPreferences_Allows(t *testing.T) {
	t.Parallel()

	p := push.CategoryPreferences{Attendance: true, Events: false, News: true, Polls: false, Absence: true}
	assert.True(t, p.Allows("attendance"))
	assert.False(t, p.Allows("events"))
	assert.True(t, p.Allows("news"))
	assert.False(t, p.Allows("polls"))
	assert.True(t, p.Allows("absence"))
}

func TestCategoryPreferences_Allows_EmptyCategoryAlwaysAllowed(t *testing.T) {
	t.Parallel()

	p := push.CategoryPreferences{}
	assert.True(t, p.Allows(""), "an empty category (unmapped notification type) must never be gated")
}

func TestCategoryPreferences_Allows_UnknownCategoryFailsClosed(t *testing.T) {
	t.Parallel()

	p := push.DefaultCategoryPreferences()
	assert.False(t, p.Allows("something_unknown"), "an unrecognized category must fail closed, even with everything else enabled")
}
