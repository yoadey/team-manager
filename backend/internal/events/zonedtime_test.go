package events_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yoadey/team-manager/backend/internal/events"
)

// TestZonedTimeToUTC_DaylightSavingBoundary covers the timezone-correctness
// concern design.md calls out explicitly: the cutoff must be computed from
// the event's absolute start instant (Europe/Berlin wall-clock -> UTC), not
// a fixed offset -- CET (winter, UTC+1) and CEST (summer, UTC+2) must each
// convert correctly depending on the event's date.
func TestZonedTimeToUTC_DaylightSavingBoundary(t *testing.T) {
	t.Parallel()

	winter := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := events.ZonedTimeToUTC(winter, "12:00")
	assert.Equal(t, time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC), got, "CET is UTC+1")

	summer := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	got = events.ZonedTimeToUTC(summer, "12:00")
	assert.Equal(t, time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC), got, "CEST is UTC+2")
}

// TestZonedTimeToUTC_FallsBackTo1800 covers the documented fallback for an
// empty or unparseable hhmm.
func TestZonedTimeToUTC_FallsBackTo1800(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC), events.ZonedTimeToUTC(date, ""))
	assert.Equal(t, time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC), events.ZonedTimeToUTC(date, "not-a-time"))
}

// TestEventStartInstant_PrecedenceAndFallback covers EventStartInstant's
// StartTime -> MeetTime -> 18:00-default precedence (mirroring
// calendarfeed.Render's identical DTSTART derivation).
func TestEventStartInstant_PrecedenceAndFallback(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	startTime := "10:00"
	meetTime := "09:30"

	// StartTime takes precedence over MeetTime.
	got := events.EventStartInstant(date, &startTime, &meetTime)
	assert.Equal(t, events.ZonedTimeToUTC(date, "10:00"), got)

	// Falls back to MeetTime when StartTime is unset.
	got = events.EventStartInstant(date, nil, &meetTime)
	assert.Equal(t, events.ZonedTimeToUTC(date, "09:30"), got)

	// Falls back to the 18:00 default when neither is set.
	got = events.EventStartInstant(date, nil, nil)
	assert.Equal(t, events.ZonedTimeToUTC(date, ""), got)
}
