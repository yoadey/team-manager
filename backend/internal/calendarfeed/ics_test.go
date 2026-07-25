package calendarfeed_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/calendarfeed"
	"github.com/yoadey/team-manager/backend/internal/events"
)

func ptr[T any](v T) *T { return &v }

func TestRender_ExcludesCancelledEvents(t *testing.T) {
	t.Parallel()

	active := events.EventRow{Id: uuid.New(), Type: "training", Title: "Aktives Training", Date: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), Status: "active"}
	cancelled := events.EventRow{Id: uuid.New(), Type: "training", Title: "Abgesagtes Training", Date: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC), Status: "cancelled"}

	out := string(calendarfeed.Render("Test Team", []events.EventRow{active, cancelled}))

	assert.Contains(t, out, "Aktives Training")
	assert.NotContains(t, out, "Abgesagtes Training")
	assert.Equal(t, 1, strings.Count(out, "BEGIN:VEVENT"))
}

func TestRender_ProducesValidVCalendarStructure(t *testing.T) {
	t.Parallel()

	e := events.EventRow{
		Id:        uuid.New(),
		Type:      "auftritt",
		Title:     "Turnier",
		Date:      time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		StartTime: ptr("18:00"),
		EndTime:   ptr("20:00"),
		Location:  ptr("Sporthalle"),
		Status:    "active",
	}

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}))

	require.True(t, strings.HasPrefix(out, "BEGIN:VCALENDAR\r\n"))
	assert.True(t, strings.HasSuffix(out, "END:VCALENDAR"))
	assert.Contains(t, out, "VERSION:2.0")
	assert.Contains(t, out, "X-WR-CALNAME:Test Team")
	assert.Contains(t, out, "UID:"+e.Id.String()+"@teamverwaltung.app")
	assert.Contains(t, out, "LOCATION:Sporthalle")
	// 18:00 Europe/Berlin in August (CEST, UTC+2) is 16:00 UTC.
	assert.Contains(t, out, "DTSTART:20260803T160000Z")
	assert.Contains(t, out, "DTEND:20260803T180000Z")
}

func TestRender_DefaultsToEighteenHundredAndTwoHourDuration(t *testing.T) {
	t.Parallel()

	e := events.EventRow{
		Id:     uuid.New(),
		Type:   "training",
		Title:  "Ohne Zeitangabe",
		Date:   time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC), // winter -> CET, UTC+1
		Status: "active",
	}

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}))

	assert.Contains(t, out, "DTSTART:20260112T170000Z") // 18:00 CET -> 17:00 UTC
	assert.Contains(t, out, "DTEND:20260112T190000Z")   // +2h
}

func TestRender_EscapesSpecialCharacters(t *testing.T) {
	t.Parallel()

	e := events.EventRow{
		Id:     uuid.New(),
		Type:   "event",
		Title:  "Comma, semicolon; back\\slash",
		Date:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Status: "active",
	}

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}))
	assert.Contains(t, out, `Comma\, semicolon\; back\\slash`)
}

func TestRender_FoldsLongLines(t *testing.T) {
	t.Parallel()

	e := events.EventRow{
		Id:     uuid.New(),
		Type:   "event",
		Title:  strings.Repeat("A", 200),
		Date:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Status: "active",
	}

	out := calendarfeed.Render("Test Team", []events.EventRow{e})
	for _, line := range strings.Split(string(out), "\r\n") {
		assert.LessOrEqual(t, len(line), 74, "no unfolded line may exceed 73 octets plus the leading space on continuations")
	}
	assert.Contains(t, string(out), "\r\n A") // continuation line prefixed with a single space
}
