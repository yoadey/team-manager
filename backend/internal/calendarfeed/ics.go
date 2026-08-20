package calendarfeed

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/events"
)

const (
	icsDateTimeFormat = "20060102T150405Z"
	icsDateFormat     = "20060102"
)

// Birthday is a member birthday to render as a yearly, all-day recurring
// VEVENT. MemberID anchors the VEVENT's UID so regenerating the feed
// updates the same calendar entry rather than duplicating it.
type Birthday struct {
	MemberID uuid.UUID
	Name     string
	Date     time.Time
}

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
// NOTE: buildIcs()'s fold() counts UTF-16 code units (JS string indexing),
// not UTF-8 octets, so for any non-ASCII content the two implementations
// don't actually fold at the same point despite sharing this width.
const icsFoldWidth = 73

// icsFold folds a content line at icsFoldWidth octets, continuation lines
// prefixed with a single space per RFC 5545 §3.1. The split point is backed
// off to the nearest preceding rune boundary so a multi-byte UTF-8 character
// (umlauts, ß, ...) is never split across two lines, which would produce
// invalid UTF-8 -- RFC 5545 §3.1 explicitly forbids breaking a line within a
// UTF-8 multi-octet sequence.
func icsFold(line string) string {
	if len(line) <= icsFoldWidth {
		return line
	}
	var b strings.Builder
	for len(line) > icsFoldWidth {
		cut := icsFoldWidth
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}
		b.WriteString(line[:cut])
		b.WriteString("\r\n ")
		line = line[cut:]
	}
	b.WriteString(line)
	return b.String()
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

// Render builds an iCalendar (RFC 5545) document for teamName's evts plus
// birthdays, mirroring useCalExportActions.ts's buildIcs(): cancelled events
// are excluded, and each VEVENT carries a UID stable across regenerations
// (so a calendar client updates the same entry rather than duplicating it)
// plus a DTSTAMP set to render time. Callers are expected to have already
// filtered evts/birthdays down to the feed's selected content and the
// caller's visibility.
func Render(teamName string, evts []events.EventRow, birthdays []Birthday) []byte {
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

		start := events.EventStartInstant(e.Date, e.StartTime, e.MeetTime)

		// A multi-day event's DTEND is anchored to its last day (EndDate),
		// not its first (Date) -- otherwise a 3-day camp would render as a
		// single ~2h block on day one, with no calendar entry at all for the
		// remaining days it actually covers.
		endDate := e.Date
		if e.EndDate != nil {
			endDate = *e.EndDate
		}

		var end time.Time
		switch {
		case e.EndTime != nil:
			end = events.ZonedTimeToUTC(endDate, *e.EndTime)
		case e.EndDate != nil:
			// No explicit end-of-day time on a multi-day span -- cover
			// through the end of the last day rather than the single-day
			// 2h-default, which would truncate the span to almost nothing.
			end = events.ZonedTimeToUTC(endDate, "23:59")
		default:
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

	for _, b := range birthdays {
		lines = append(
			lines,
			"BEGIN:VEVENT",
			"UID:birthday-"+b.MemberID.String()+"@teamverwaltung.app",
			"DTSTAMP:"+now.Format(icsDateTimeFormat),
			"DTSTART;VALUE=DATE:"+b.Date.Format(icsDateFormat),
			"RRULE:FREQ=YEARLY",
			icsFold("SUMMARY:"+icsEscape("Geburtstag: "+b.Name)),
			"END:VEVENT",
		)
	}

	lines = append(lines, "END:VCALENDAR")
	return []byte(strings.Join(lines, "\r\n"))
}
