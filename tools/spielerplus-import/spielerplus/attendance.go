package spielerplus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ajaxParticipationPath is confirmed from a HAR capture of a live session:
// POST with form fields "eventid" and "eventtype" (e.g. "training"),
// X-Requested-With: XMLHttpRequest, triggered by the "Teilnehmer anzeigen"
// (show participants) button - as opposed to the separate per-user
// "showParticipationForm" widget used to set the *caller's own* status.
// The response is JSON-wrapped like ajaxgetevents ({"html": "..."}) and its
// html lists every team member grouped by status, confirmed by inspecting a
// real response body: this is the full-roster source this importer needs,
// not a self-only one.
const ajaxParticipationPath = "/events/ajaxgetparticipation"

// Selectors below are confirmed from a HAR capture of a live
// ajaxgetparticipation response. Members are grouped under
// `<div class="collapse in" id="{code}-parti-collapse">` - one group per
// ParticipationStatus code (see types.go) - each containing zero or more
// `.participation-list-user` blocks:
//
//	<div class="participation-list-user">
//	    <a class="participation-list-user-photo" href="/user/view?id={userID}">...</a>
//	    <div class="participation-list-user-infos">
//	        <div class="participation-list-user-name">{Name}</div>
//	        <div class="participation-list-user-reason ...">
//	            <div class="reason-text">{Reason}</div>   <!-- absent if none given -->
//	        </div>
//	    </div>
//	    ...
//	</div>
const (
	participationGroupSelector = `div[id$="-parti-collapse"]`
	participantRowSelector     = ".participation-list-user"
	participantPhotoSelector   = ".participation-list-user-photo"
	participantReasonSelector  = ".participation-list-user-reason .reason-text"
)

var participationGroupIDRegexp = regexp.MustCompile(`^(\d+)-parti-collapse$`)

// ajaxParticipationResponse is the JSON envelope ajaxParticipationPath
// responds with (same shape as ajaxEventsResponse, minus "count").
type ajaxParticipationResponse struct {
	HTML string `json:"html"`
}

// ParseAttendance parses an ajaxgetparticipation JSON response into one
// Attendance record per member.
func ParseAttendance(body io.Reader, eventID string) ([]Attendance, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: read event %s participant list: %w", eventID, err)
	}
	var resp ajaxParticipationResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("spielerplus: parse event %s participant list JSON envelope: %w", eventID, err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.HTML))
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse event %s participant list html: %w", eventID, err)
	}

	groups := doc.Find(participationGroupSelector)
	if groups.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no participation-status groups matched selector %q for event %s - SpielerPlus markup likely differs from what this parser expects, inspect the participation fragment and update participationGroupSelector in attendance.go", participationGroupSelector, eventID)
	}

	var records []Attendance
	groups.Each(func(_ int, group *goquery.Selection) {
		groupID, _ := group.Attr("id")
		m := participationGroupIDRegexp.FindStringSubmatch(groupID)
		if m == nil {
			return
		}
		status := ParticipationStatus(m[1])

		group.Find(participantRowSelector).Each(func(_ int, row *goquery.Selection) {
			href, _ := row.Find(participantPhotoSelector).First().Attr("href")
			userID := userIDFromHref(href)
			if userID == "" {
				return
			}
			reason := strings.TrimSpace(row.Find(participantReasonSelector).First().Text())
			records = append(records, Attendance{
				EventID:  eventID,
				MemberID: userID,
				Status:   status,
				Reason:   reason,
			})
		})
	})
	return records, nil
}

// userIDFromHref extracts the "id" query parameter from a
// "/user/view?id=123" style href.
func userIDFromHref(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return u.Query().Get("id")
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
