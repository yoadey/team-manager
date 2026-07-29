package events

import (
	"strconv"
	"strings"
	"time"
)

// europeBerlin is the hardcoded timezone every event's wall-clock
// meet/start/end time is interpreted in, matching the frontend's identical
// hardcoded assumption (useCalExportActions.ts's zonedTimeToUtc()) -- neither
// side of this app has a per-team timezone concept yet.
var europeBerlin = mustLoadLocation("Europe/Berlin")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Europe/Berlin is a standard IANA zone; a missing tzdata would be a
		// broken deployment environment, not a runtime condition to recover
		// from.
		panic("events: " + err.Error())
	}
	return loc
}

// parseHHMM parses a "HH:MM" string as produced by selectEventFields's
// TO_CHAR(..., 'HH24:MI').
func parseHHMM(s string) (hour, minute int, ok bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, errH := strconv.Atoi(parts[0])
	m, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil {
		return 0, 0, false
	}
	return h, m, true
}

// ZonedTimeToUTC interprets date's year/month/day combined with hhmm as
// wall-clock time in Europe/Berlin, returning the equivalent UTC instant. An
// empty or unparseable hhmm falls back to 18:00. Shared by calendarfeed.Render
// (the feed's DTSTART/DTEND) and EventStartInstant below, so the two stay
// consistent instead of maintaining separate timezone logic.
func ZonedTimeToUTC(date time.Time, hhmm string) time.Time {
	hour, minute, ok := parseHHMM(hhmm)
	if !ok {
		hour, minute = 18, 0
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, europeBerlin).UTC()
}

// EventStartInstant returns the absolute UTC instant an event starts, from
// its Date plus StartTime (falling back to MeetTime, then ZonedTimeToUTC's
// own 18:00 default, when neither is set) -- the same StartTime→MeetTime
// precedence calendarfeed.Render already uses for the feed's DTSTART. This is
// the "start" a CancelLeadMinutes cutoff counts back from (see
// Service.SetAttendance).
func EventStartInstant(date time.Time, startTime, meetTime *string) time.Time {
	var hhmm string
	switch {
	case startTime != nil:
		hhmm = *startTime
	case meetTime != nil:
		hhmm = *meetTime
	}
	return ZonedTimeToUTC(date, hhmm)
}
