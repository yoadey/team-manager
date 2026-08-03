package spielerplus

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// eventsPath is the initial full-page events list. ajaxEventsPath is the
// endpoint the real frontend calls to page further (both forward and back
// into history) - confirmed from a HAR capture of a live session: the page
// loads eventsPath once, then issues POST ajaxEventsPath calls with an
// incrementing "offset" (observed in steps of eventsPageSize) as the user
// scrolls, and an additional "old=true" field once the user switches into
// "past events" mode (the page's own "Ältere Termine laden" button starts
// with offset="0"). Whether "old" must be repeated on every page (vs. only
// once to flip a server-side toggle) is unconfirmed -
// fetchEventsAjaxPage sends "old" on every historical request as the safer
// assumption. ajaxEventsPath's response is JSON (`{"html": "...",
// "count": N}`, confirmed) - "count" is the authoritative number of events
// in that page and is used to detect the end of history.
const (
	eventsPath      = "/events"
	ajaxEventsPath  = "/events/ajaxgetevents"
	eventsPageSize  = 5
	maxHistoryPages = 500 // safety cap: ~2500 events, in case count never reaches 0
)

// Selectors below are confirmed from a HAR capture of a live session's
// /events page and ajaxgetevents fragment (both render the same markup):
// each event is a `<div class="panel" id="event-{type}-{id}">`
// (eventRowSelector below matches on the id prefix), with the title under
// `.panel-heading-text .panel-title` (note `.panel-title` is reused inside
// `.panel-heading-info` for the weekday abbreviation, hence the more
// specific selector) and the year-less date under `.panel-heading-info
// .panel-subtitle` (format "DD.MM", no trailing dot). Up to three
// `.event-time-item` blocks hold the meet/start/end times in that fixed
// order (Treffen/Beginn/Ende in a German-language account) - read
// positionally rather than by label text so this doesn't depend on the
// account's display language. Only "training" events were present in the
// capture; the id-prefix-derived type covers whatever slug SpielerPlus
// uses for other types too (mapEventType below still only recognizes the
// slugs guessed from the "create new event" menu - see design.md).
var eventIDRegexp = regexp.MustCompile(`^event-([a-z]+)-(\d+)$`)

const (
	eventRowSelector      = `div.panel[id^="event-"]`
	eventTitleSelector    = ".panel-heading-text .panel-title"
	eventDateSelector     = ".panel-heading-info .panel-subtitle"
	eventTimeItemSelector = ".event-time .event-time-item .event-time-value"
	eventLocationSelector = ".event-location, .location" // unverified - no location field observed in the capture (see design.md)
)

// ParseEvents parses a full /events page body into structured events. now
// anchors year resolution for the first (year-less, "DD.MM.") date on the
// page; every subsequent date is resolved relative to the previously
// resolved one instead, so the sequence stays consistent however far it
// wanders from now (see resolveDate).
func ParseEvents(body io.Reader, now time.Time) ([]Event, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse events page: %w", err)
	}

	rows := doc.Find(eventRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no event rows matched selector %q - SpielerPlus markup likely changed, inspect the /events page and update eventRowSelector in events.go", eventRowSelector)
	}

	events, skipped := parseEventRows(rows, now)
	if len(events) == 0 {
		return nil, fmt.Errorf("spielerplus: %d event row(s) matched selector %q but none parsed successfully - selectors likely need adjusting, see logged per-row errors above", rows.Length(), eventRowSelector)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d event(s), skipped %d that failed to parse", len(events), skipped)
	}
	return events, nil
}

// ajaxEventsResponse is the JSON envelope ajaxEventsPath responds with,
// confirmed from a HAR capture: {"html": "<the same per-event markup
// ParseEvents parses>", "count": <events in this page>}.
type ajaxEventsResponse struct {
	HTML  string `json:"html"`
	Count int    `json:"count"`
}

// parseEventsFragment parses an ajaxgetevents JSON response. Unlike
// ParseEvents, zero events is not an error here - it's the expected "no
// more pages" signal used to end pagination (via the response's own
// "count", which is authoritative regardless of whether parsing every row
// succeeds).
func parseEventsFragment(body io.Reader, reference time.Time) (events []Event, count int, err error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, 0, fmt.Errorf("spielerplus: read events fragment: %w", err)
	}
	var resp ajaxEventsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, 0, fmt.Errorf("spielerplus: parse events fragment JSON envelope: %w", err)
	}
	if resp.Count == 0 {
		return nil, 0, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.HTML))
	if err != nil {
		return nil, 0, fmt.Errorf("spielerplus: parse events fragment html: %w", err)
	}
	rows := doc.Find(eventRowSelector)
	events, skipped := parseEventRows(rows, reference)
	if skipped > 0 {
		log.Printf("spielerplus: fragment: imported %d event(s), skipped %d that failed to parse", len(events), skipped)
	}
	return events, resp.Count, nil
}

// parseEventRows parses every row matched by a selector in document order,
// threading a "reference" date through them so each date resolves relative
// to the previously resolved one (see resolveDate) rather than always
// against a fixed anchor - required for paginated results that wander far
// from "now" in either direction. A row-specific parse error is logged and
// skipped rather than aborting the batch (see ParseEvents doc).
func parseEventRows(rows *goquery.Selection, initialReference time.Time) (events []Event, skipped int) {
	reference := initialReference
	rows.Each(func(i int, row *goquery.Selection) {
		ev, err := parseEventRow(row, reference)
		if err != nil {
			skipped++
			log.Printf("spielerplus: skipping event row %d: %v", i, err)
			return
		}
		events = append(events, ev)
		if !ev.TimeUnknown {
			reference = ev.Start
		}
	})
	return events, skipped
}

func parseEventRow(row *goquery.Selection, reference time.Time) (Event, error) {
	idAttr, ok := row.Attr("id")
	if !ok {
		return Event{}, fmt.Errorf("missing id")
	}
	m := eventIDRegexp.FindStringSubmatch(idAttr)
	if m == nil {
		return Event{}, fmt.Errorf("id %q doesn't match event-<type>-<id>", idAttr)
	}
	typeSlug, id := m[1], m[2]

	title := strings.TrimSpace(row.Find(eventTitleSelector).First().Text())
	if title == "" {
		return Event{}, fmt.Errorf("missing title (event %s)", id)
	}

	dateText := strings.TrimSpace(row.Find(eventDateSelector).First().Text())
	meet, start, end, endIsEstimated, timeUnknown, err := parseDateTime(dateText, eventTimes(row), reference)
	if err != nil {
		return Event{}, fmt.Errorf("event %s: %w", id, err)
	}

	location := strings.TrimSpace(row.Find(eventLocationSelector).First().Text())

	return Event{
		ID:             id,
		Type:           mapEventType(typeSlug),
		Title:          title,
		Location:       location,
		Start:          start,
		End:            end,
		EndIsEstimated: endIsEstimated,
		TimeUnknown:    timeUnknown,
		MeetTime:       meet,
	}, nil
}

// eventTimes reads the row's .event-time-value elements positionally:
// three values are (meet, start, end); two are (start, end); one is just
// (start). Read positionally rather than by the German "Treffen"/"Beginn"/
// "Ende" label text, so this doesn't depend on the account's display
// language (see the eventRowSelector doc comment).
func eventTimes(row *goquery.Selection) []string {
	var values []string
	row.Find(eventTimeItemSelector).Each(func(_ int, v *goquery.Selection) {
		values = append(values, strings.TrimSpace(v.Text()))
	})
	return values
}

// eventTypeSlug maps our EventType back to the SpielerPlus type identifier
// used in the "event-<type>-<id>" row id and as the "eventtype" form field
// for ajaxgetparticipation. Only "training" is confirmed (from a HAR
// capture); the others are guesses from the "create new event" menu's
// URLs (/game/create, /tournament/create, /event/create).
func eventTypeSlug(t EventType) string {
	switch t {
	case EventTraining:
		return "training"
	case EventGame:
		return "game"
	case EventTournament:
		return "tournament"
	default:
		return "event"
	}
}

func mapEventType(slug string) EventType {
	switch slug {
	case "training":
		return EventTraining
	case "game":
		return EventGame
	case "tournament":
		return EventTournament
	default:
		return EventOther
	}
}

// dayMonthLayout matches SpielerPlus's year-less date display, confirmed
// from a HAR capture: "26.07" (no trailing dot).
const dayMonthLayout = "02.01"

// resolveDate resolves a SpielerPlus "DD.MM" date against reference by
// picking whichever of reference's year, year-1, or year+1 puts the result
// closest to reference. Applied to a chronologically continuous sequence of
// events (each resolved date becomes the next row's reference - see
// parseEventRows), this correctly follows the list forward into the future
// (the plain /events page) or backward into the past (paginated history via
// ajaxgetevents) without needing to know in advance which direction a given
// page is going - unlike a fixed "assume next year once far enough in the
// past" rule, which is only valid for forward/upcoming lists and would
// silently roll genuinely historical dates a year forward.
func resolveDate(dateText string, reference time.Time) (time.Time, error) {
	parsed, err := time.Parse(dayMonthLayout, dateText)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %q: %w", dateText, err)
	}

	var best time.Time
	bestDiff := time.Duration(math.MaxInt64)
	for _, yearOffset := range [3]int{-1, 0, 1} {
		candidate := time.Date(reference.Year()+yearOffset, parsed.Month(), parsed.Day(), 0, 0, 0, 0, reference.Location())
		diff := candidate.Sub(reference)
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = candidate
		}
	}
	return best, nil
}

// parseDateTime resolves a SpielerPlus "DD.MM" date (via resolveDate) plus
// up to three positional time values (see eventTimes) into meet/start/end
// times.
func parseDateTime(dateText string, times []string, reference time.Time) (meet, start, end time.Time, endIsEstimated, timeUnknown bool, err error) {
	if dateText == "" {
		return time.Time{}, time.Time{}, time.Time{}, false, false, fmt.Errorf("missing date")
	}
	day, err := resolveDate(dateText, reference)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, false, false, err
	}

	var meetHM, startHM, endHM string
	switch len(times) {
	case 0:
		// No time information at all: day's time-of-day is a meaningless
		// midnight placeholder, not a real start time.
		return time.Time{}, day, day.Add(2 * time.Hour), true, true, nil
	case 1:
		startHM = times[0]
	case 2:
		startHM, endHM = times[0], times[1]
	default:
		meetHM, startHM, endHM = times[0], times[1], times[2]
	}

	at := func(hm string) (time.Time, error) {
		h, m, perr := parseHM(hm)
		if perr != nil {
			return time.Time{}, fmt.Errorf("parse time %q: %w", hm, perr)
		}
		return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, day.Location()), nil
	}

	if meetHM != "" {
		if meet, err = at(meetHM); err != nil {
			return time.Time{}, time.Time{}, time.Time{}, false, false, err
		}
	}
	if start, err = at(startHM); err != nil {
		return time.Time{}, time.Time{}, time.Time{}, false, false, err
	}
	if endHM == "" {
		// A start time is known but no end time: estimate start + 2h, same
		// fallback the reference implementations use.
		return meet, start, start.Add(2 * time.Hour), true, false, nil
	}
	if end, err = at(endHM); err != nil {
		return time.Time{}, time.Time{}, time.Time{}, false, false, err
	}
	return meet, start, end, false, false, nil
}

func parseHM(s string) (hour, minute int, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected HH:MM, got %q", s)
	}
	hour, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	minute, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return hour, minute, nil
}

// FetchEvents fetches the initial /events page and then pages backward
// through history via ajaxgetevents (see the package-level doc comment on
// ajaxEventsPath) until a page reports a zero count or maxHistoryPages is
// hit.
func (c *Client) FetchEvents(now time.Time) ([]Event, error) {
	body, err := c.get(eventsPath)
	if err != nil {
		return nil, err
	}
	initial, err := ParseEvents(body, now)
	body.Close()
	if err != nil {
		return nil, err
	}

	events := initial
	reference := now
	if len(initial) > 0 {
		reference = initial[len(initial)-1].Start
	}

	for page := 0; page < maxHistoryPages; page++ {
		fragment, count, err := c.fetchEventsAjaxPage(page*eventsPageSize, reference)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			break
		}
		events = append(events, fragment...)
		if len(fragment) > 0 {
			reference = fragment[len(fragment)-1].Start
		}
	}

	return events, nil
}

func (c *Client) fetchEventsAjaxPage(offset int, reference time.Time) ([]Event, int, error) {
	form := url.Values{
		"offset": {strconv.Itoa(offset)},
		"old":    {"true"},
	}
	body, err := c.postForm(ajaxEventsPath, form)
	if err != nil {
		return nil, 0, fmt.Errorf("spielerplus: fetch events page at offset %d: %w", offset, err)
	}
	defer body.Close()
	return parseEventsFragment(body, reference)
}
