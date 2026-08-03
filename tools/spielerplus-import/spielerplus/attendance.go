package spielerplus

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ajaxParticipationPath is confirmed from a HAR capture of a live session:
// POST with form fields "eventid" and "eventtype" (e.g. "training"),
// X-Requested-With: XMLHttpRequest. The capture only exercised it for a
// training event reached from its /training/view?id=... detail page, so it
// is unconfirmed whether the returned fragment lists every team member's
// status (what this importer needs) or only the caller's own - the known
// community reference clients only ever read/write their own status via
// this same family of endpoint. If it turns out to be self-only, this
// importer needs a different, trainer-facing source for full-roster
// attendance (tasks.md 2.3).
const ajaxParticipationPath = "/events/ajaxgetparticipation"

// Selectors below are unverified against a live account - see tasks.md
// 2.3/2.6 (the HAR capture this endpoint is grounded in didn't include
// response bodies).
const (
	participantRowSelector  = "[data-user-id]"
	participantStatusAttr   = "title"
	participantSelectedFlag = "selected"
)

// spielerPlusStatusTitles maps the title text on a participation status
// element (as seen on both the self-service button in the reference
// projects and, presumably, the trainer-facing participant list) to our
// ParticipationStatus vocabulary.
var spielerPlusStatusTitles = map[string]ParticipationStatus{
	"Zugesagt":           ParticipationAccepted,
	"Unsicher":           ParticipationUnsure,
	"Absagen / Abwesend": ParticipationDeclined,
	"Absagen/Abwesend":   ParticipationDeclined,
	"Abgesagt":           ParticipationDeclined,
	"Keine Rückmeldung":  ParticipationNoResonse,
}

// ParseAttendance parses an ajaxgetparticipation response fragment into one
// Attendance record per member. Members with no explicit status are
// reported as ParticipationNoResonse.
func ParseAttendance(body io.Reader, eventID string) ([]Attendance, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse event %s participant list: %w", eventID, err)
	}

	rows := doc.Find(participantRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no participant rows matched selector %q for event %s - SpielerPlus markup likely differs from what this parser expects, inspect the participation fragment and update participantRowSelector in attendance.go", participantRowSelector, eventID)
	}

	var records []Attendance
	rows.Each(func(_ int, row *goquery.Selection) {
		userID, ok := row.Attr("data-user-id")
		if !ok || userID == "" {
			return
		}
		status := ParticipationNoResonse
		selected := row.Find("." + participantSelectedFlag).First()
		if selected.Length() > 0 {
			if title, ok := selected.Attr(participantStatusAttr); ok {
				if mapped, known := spielerPlusStatusTitles[strings.TrimSpace(title)]; known {
					status = mapped
				}
			}
		}
		records = append(records, Attendance{
			EventID:  eventID,
			MemberID: userID,
			Status:   status,
		})
	})
	return records, nil
}

// FetchAttendance fetches and parses the participation fragment for
// eventID/eventType (eventType is SpielerPlus's own type identifier, e.g.
// "training" - see eventTypeSlug).
func (c *Client) FetchAttendance(eventID string, eventType EventType) ([]Attendance, error) {
	form := url.Values{
		"eventid":   {eventID},
		"eventtype": {eventTypeSlug(eventType)},
	}
	body, err := c.postForm(ajaxParticipationPath, form)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: fetch attendance for event %s: %w", eventID, err)
	}
	defer body.Close()
	return ParseAttendance(body, eventID)
}
