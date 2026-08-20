package calendarfeed_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

	out := string(calendarfeed.Render("Test Team", []events.EventRow{active, cancelled}, nil))

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

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}, nil))

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

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}, nil))

	assert.Contains(t, out, "DTSTART:20260112T170000Z") // 18:00 CET -> 17:00 UTC
	assert.Contains(t, out, "DTEND:20260112T190000Z")   // +2h
}

func TestRender_MultiDayEvent_DTENDAnchoredToLastDayEndTime(t *testing.T) {
	t.Parallel()

	end := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	e := events.EventRow{
		Id:        uuid.New(),
		Type:      "training",
		Title:     "Trainingslager",
		Date:      time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC),
		EndDate:   &end,
		StartTime: ptr("09:00"),
		EndTime:   ptr("16:00"),
		Status:    "active",
	}

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}, nil))

	// 09:00 Europe/Berlin in August (CEST, UTC+2) is 07:00 UTC on the first day.
	assert.Contains(t, out, "DTSTART:20260814T070000Z")
	// 16:00 CEST on the LAST day (Aug 16), not the first.
	assert.Contains(t, out, "DTEND:20260816T140000Z")
}

func TestRender_MultiDayEvent_NoEndTime_CoversThroughEndOfLastDay(t *testing.T) {
	t.Parallel()

	end := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	e := events.EventRow{
		Id:      uuid.New(),
		Type:    "event",
		Title:   "Vereinsfahrt",
		Date:    time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC),
		EndDate: &end,
		Status:  "active",
	}

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}, nil))

	// 23:59 CEST on the last day (Aug 16) is 21:59 UTC.
	assert.Contains(t, out, "DTEND:20260816T215900Z")
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

	out := string(calendarfeed.Render("Test Team", []events.EventRow{e}, nil))
	assert.Contains(t, out, `Comma\, semicolon\; back\\slash`)
}

func TestRender_IncludesBirthdaysAsYearlyAllDayEvents(t *testing.T) {
	t.Parallel()

	memberID := uuid.New()
	birthdays := []calendarfeed.Birthday{
		{MemberID: memberID, Name: "Ada Lovelace", Date: time.Date(2000, 5, 17, 0, 0, 0, 0, time.UTC)},
	}

	out := string(calendarfeed.Render("Test Team", nil, birthdays))

	assert.Contains(t, out, "UID:birthday-"+memberID.String()+"@teamverwaltung.app")
	assert.Contains(t, out, "DTSTART;VALUE=DATE:20000517")
	assert.Contains(t, out, "RRULE:FREQ=YEARLY")
	assert.Contains(t, out, "SUMMARY:Geburtstag: Ada Lovelace")
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

	out := calendarfeed.Render("Test Team", []events.EventRow{e}, nil)
	for _, line := range strings.Split(string(out), "\r\n") {
		assert.LessOrEqual(t, len(line), 74, "no unfolded line may exceed 73 octets plus the leading space on continuations")
	}
	assert.Contains(t, string(out), "\r\n A") // continuation line prefixed with a single space
}

// TestRender_FoldsMultiByteUTF8WithoutSplittingRunes guards against folding
// SUMMARY/LOCATION/DESCRIPTION mid-rune. The padding lengths below are
// chosen (verified with a byte-offset calculation, not by trial and error at
// test time) so that under the *old* line[:73]/line[73:] byte-slicing logic,
// the fold cutoff lands exactly one byte into a following two-byte UTF-8
// rune (an "ä", 0xC3 0xA4) -- i.e. right after its leading byte -- which
// splits that rune's bytes across two output lines and produces invalid
// UTF-8. This test fails under the old logic and passes once folding
// backs off to a rune boundary.
func TestRender_FoldsMultiByteUTF8WithoutSplittingRunes(t *testing.T) {
	t.Parallel()

	title := strings.Repeat("A", 64) + "äöüß Vereinsausflug ins Grüne mit Übernachtung und Frühstück"
	location := strings.Repeat("A", 63) + "äöüß Größenstraße direkt gegenüber vom Vereinsheim"
	note := strings.Repeat("A", 60) + "äöüß Bitte über Änderungen der Abfahrtszeit Bescheid geben"

	e := events.EventRow{
		Id:       uuid.New(),
		Type:     "training",
		Title:    title,
		Date:     time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		Location: &location,
		Note:     &note,
		Status:   "active",
	}

	out := calendarfeed.Render("Test Team", []events.EventRow{e}, nil)

	require.True(t, utf8.Valid(out), "rendered ICS must be valid UTF-8 -- folding must never split a multi-byte rune")

	// Unfolding (removing every CRLF+space continuation marker) must
	// reconstruct each field's original text exactly.
	unfolded := strings.ReplaceAll(string(out), "\r\n ", "")
	assert.Contains(t, unfolded, "SUMMARY:"+title)
	assert.Contains(t, unfolded, "LOCATION:"+location)
	assert.Contains(t, unfolded, note)
}
