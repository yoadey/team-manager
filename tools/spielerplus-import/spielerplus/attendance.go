package spielerplus

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// eventDetailPath is the per-event page. The exact path segment is a
// best-effort guess (SpielerPlus's event list links to a per-event page;
// the reference community projects only ever read/write the *viewer's own*
// participation status via the ajax-participation-form endpoint, since
// they're personal calendar tools, not trainer/admin tools). A full-team
// migration needs the trainer-facing participant list for each event
// instead, which has no existing reference and must be confirmed against a
// live account (tasks.md 2.3).
const eventDetailPathFmt = "/events/view/%s"

// Selectors below are unverified against a live account, same caveat as
// events.go - see tasks.md 2.3/2.6.
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

// ParseAttendance parses an event detail page's participant list into one
// Attendance record per member. Members with no explicit status are
// reported as ParticipationNoResonse.
func ParseAttendance(body io.Reader, eventID string) ([]Attendance, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse event %s participant list: %w", eventID, err)
	}

	rows := doc.Find(participantRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no participant rows matched selector %q for event %s - SpielerPlus markup likely differs from what this parser expects, inspect the event page and update participantRowSelector in attendance.go", participantRowSelector, eventID)
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

// FetchAttendance fetches and parses the participant list for eventID.
func (c *Client) FetchAttendance(eventID string) ([]Attendance, error) {
	body, err := c.get(fmt.Sprintf(eventDetailPathFmt, eventID))
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return ParseAttendance(body, eventID)
}
