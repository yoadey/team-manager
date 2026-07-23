package calendarfeed

import (
	"strconv"
	"strings"
	"time"

	"github.com/yoadey/team-manager/backend/internal/events"
)

// europeBerlin is the hardcoded timezone every event's wall-clock
// start/end time is interpreted in, matching the frontend's identical
// hardcoded assumption (useCalExportActions.ts's buildIcs()) -- neither side
// of this app has a per-team timezone concept yet.
var europeBerlin = mustLoadLocation("Europe/Berlin")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Europe/Berlin is a standard IANA zone; a missing tzdata would be a
		// broken deployment environment, not a runtime condition to recover
		// from.
		panic("calendarfeed: " + err.Error())
	}
	return loc
}

const icsDateTimeFormat = "20060102T150405Z"

// icsEscape escapes text per RFC 5545 §3.3.11, matching buildIcs()'s esc().
func icsEscape(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		"\r\n", `\n`,
		"\r", `\n`,
		"\n", `\n`,
		`,`, `\,`,
		`;`, `\;`,
	)
	return r.Replace(s)
}

// icsFoldWidth matches buildIcs()'s fold() -- 73 octets, not the RFC 5545
// §3.1-recommended 75, so the two renderers stay recognizably equivalent.
const icsFoldWidth = 73

// icsFold folds a content line at icsFoldWidth octets, continuation lines
// prefixed with a single space per RFC 5545 §3.1.
func icsFold(line string) string {
	if len(line) <= icsFoldWidth {
		return line
	}
	var b strings.Builder
	for len(line) > icsFoldWidth {
		b.WriteString(line[:icsFoldWidth])
		b.WriteString("\r\n ")
		line = line[icsFoldWidth:]
	}
	b.WriteString(line)
	return b.String()
}

// parseHHMM parses a "HH:MM" string as produced by
// events.selectEventFields' TO_CHAR(..., 'HH24:MI').
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

// zonedTimeToUTC interprets date's year/month/day combined with hhmm as
// wall-clock time in Europe/Berlin, returning the equivalent UTC instant --
// the Go stdlib equivalent of the frontend's zonedTimeToUtc(date, hhmm,
// 'Europe/Berlin'). An empty or unparseable hhmm falls back to 18:00,
// matching buildIcs()'s own '18:00' fallback.
func zonedTimeToUTC(date time.Time, hhmm string) time.Time {
	hour, minute, ok := parseHHMM(hhmm)
	if !ok {
		hour, minute = 18, 0
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, europeBerlin).UTC()
}

// eventTypeLabel mirrors the frontend's eventType.* i18n strings. The
// backend has no i18n system (locale is a client-side-only preference) and
// the unauthenticated feed route in particular has no per-request locale
// signal at all, so this defaults to German, matching the project's
// dominant language elsewhere in generated user-facing text.
func eventTypeLabel(eventType string) string {
	switch eventType {
	case "training":
		return "Training"
	case "auftritt":
		return "Auftritt / Turnier"
	default:
		return "Team-Event"
	}
}

// Render builds an iCalendar (RFC 5545) document for teamName's evts,
// mirroring useCalExportActions.ts's buildIcs(): cancelled events are
// excluded, and each VEVENT carries a UID stable across regenerations (so a
// calendar client updates the same entry rather than duplicating it) plus a
// DTSTAMP set to render time.
func Render(teamName string, evts []events.EventRow) []byte {
	now := time.Now().UTC()

	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Teamverwaltung//Termine//DE",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"X-WR-CALNAME:" + icsEscape(teamName),
		"X-WR-TIMEZONE:Europe/Berlin",
	}

	for _, e := range evts {
		if e.Status == "cancelled" {
			continue
		}

		startHHMM := ""
		switch {
		case e.StartTime != nil:
			startHHMM = *e.StartTime
		case e.MeetTime != nil:
			startHHMM = *e.MeetTime
		}
		start := zonedTimeToUTC(e.Date, startHHMM)

		var end time.Time
		if e.EndTime != nil {
			end = zonedTimeToUTC(e.Date, *e.EndTime)
		} else {
			end = start.Add(2 * time.Hour)
		}

		descParts := []string{}
		if e.MeetTime != nil && *e.MeetTime != "" {
			descParts = append(descParts, "Treffpunkt: "+*e.MeetTime)
		}
		if e.Note != nil && *e.Note != "" {
			descParts = append(descParts, *e.Note)
		}
		descParts = append(descParts, "Termintyp: "+eventTypeLabel(e.Type))

		lines = append(
			lines,
			"BEGIN:VEVENT",
			"UID:"+e.Id.String()+"@teamverwaltung.app",
			"DTSTAMP:"+now.Format(icsDateTimeFormat),
			"DTSTART:"+start.Format(icsDateTimeFormat),
			"DTEND:"+end.Format(icsDateTimeFormat),
			icsFold("SUMMARY:"+icsEscape(e.Title)),
		)
		if e.Location != nil && *e.Location != "" {
			lines = append(lines, icsFold("LOCATION:"+icsEscape(*e.Location)))
		}
		lines = append(lines, icsFold("DESCRIPTION:"+icsEscape(strings.Join(descParts, "\n"))), "END:VEVENT")
	}

	lines = append(lines, "END:VCALENDAR")
	return []byte(strings.Join(lines, "\r\n"))
}
