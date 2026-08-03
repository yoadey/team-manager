package spielerplus

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
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
// "past events" mode. The capture didn't include response bodies, so the
// exact end-of-history signal and whether "old" must be repeated on every
// page (vs. only once to flip a server-side toggle) are unconfirmed -
// fetchEventsAjaxPage sends "old" on every historical request as the safer
// assumption, and pagination stops once a page parses zero events.
const (
	eventsPath      = "/events"
	ajaxEventsPath  = "/events/ajaxgetevents"
	eventsPageSize  = 5
	maxHistoryPages = 500 // safety cap: ~2500 events, in case "end of history" never surfaces as zero rows
)

// Selectors below are a best-effort starting point derived from the
// reference implementations' documented behavior (event rows carry a type,
// a title, a date without a year, and an optional time/end time). They have
// NOT been validated against a live SpielerPlus account and are the most
// likely thing to need adjustment once real markup is available - see
// tasks.md 2.2/2.6. ParseEvents fails loudly (rather than returning zero
// events) when nothing matches, so a selector drift is caught immediately
// instead of silently importing an empty history.
const (
	eventRowSelector      = "[data-event-id], .event-list-item, .events-list .event"
	eventTypeAttr         = "data-event-type"
	eventTitleSelector    = ".event-title, .title"
	eventDateSelector     = ".event-date, .date"
	eventTimeSelector     = ".event-time, .time"
	eventLocationSelector = ".event-location, .location"
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

// parseEventsFragment parses an ajaxgetevents response fragment. Unlike
// ParseEvents, zero matched rows is not an error here - it's the expected
// "no more pages" signal used to end pagination.
func parseEventsFragment(body io.Reader, reference time.Time) (events []Event, err error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse events fragment: %w", err)
	}
	rows := doc.Find(eventRowSelector)
	events, skipped := parseEventRows(rows, reference)
	if skipped > 0 {
		log.Printf("spielerplus: fragment: imported %d event(s), skipped %d that failed to parse", len(events), skipped)
	}
	return events, nil
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
	id, ok := row.Attr("data-event-id")
	if !ok || id == "" {
		return Event{}, fmt.Errorf("missing data-event-id")
	}

	title := strings.TrimSpace(row.Find(eventTitleSelector).First().Text())
	if title == "" {
		return Event{}, fmt.Errorf("missing title (event %s)", id)
	}

	dateText := strings.TrimSpace(row.Find(eventDateSelector).First().Text())
	timeText := strings.TrimSpace(row.Find(eventTimeSelector).First().Text())
	start, endIsEstimated, end, timeUnknown, err := parseDateTime(dateText, timeText, reference)
	if err != nil {
		return Event{}, fmt.Errorf("event %s: %w", id, err)
	}

	typeAttr, _ := row.Attr(eventTypeAttr)
	location := strings.TrimSpace(row.Find(eventLocationSelector).First().Text())

	return Event{
		ID:             id,
		Type:           mapEventType(typeAttr),
		Title:          title,
		Location:       location,
		Start:          start,
		End:            end,
		EndIsEstimated: endIsEstimated,
		TimeUnknown:    timeUnknown,
	}, nil
}

// eventTypeSlug maps our EventType back to the SpielerPlus type identifier
// used as the "eventtype" form field for ajaxgetparticipation. Only
// "training" is confirmed (from a HAR capture); the others are carried over
// from mapEventType's guesses and unverified.
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

func mapEventType(raw string) EventType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "training":
		return EventTraining
	case "game", "match", "spiel":
		return EventGame
	case "tournament", "turnier":
		return EventTournament
	default:
		return EventOther
	}
}

// dayMonthLayout matches SpielerPlus's year-less date display (e.g.
// "12.08."), which the reference implementations also had to work around.
const dayMonthLayout = "02.01."

// resolveDate resolves a SpielerPlus "DD.MM." date against reference by
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

// parseDateTime resolves a SpielerPlus "DD.MM." date plus an optional
// "HH:MM - HH:MM" (or single "HH:MM") time string into a start/end time,
// using resolveDate (see its doc) to pick the year.
func parseDateTime(dateText, timeText string, reference time.Time) (start time.Time, endIsEstimated bool, end time.Time, timeUnknown bool, err error) {
	if dateText == "" {
		return time.Time{}, false, time.Time{}, false, fmt.Errorf("missing date")
	}
	candidate, err := resolveDate(dateText, reference)
	if err != nil {
		return time.Time{}, false, time.Time{}, false, err
	}

	startHM, endHM, hasTime := splitTimeRange(timeText)
	if !hasTime {
		// No time information at all on the page: Start's time-of-day is a
		// meaningless midnight placeholder, not a real start time - flag it
		// so callers (db.InsertEvent) don't write out a fake 00:00 slot.
		return candidate, true, candidate.Add(2 * time.Hour), true, nil
	}

	sh, sm, perr := parseHM(startHM)
	if perr != nil {
		return time.Time{}, false, time.Time{}, false, fmt.Errorf("parse start time %q: %w", timeText, perr)
	}
	start = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), sh, sm, 0, 0, candidate.Location())

	if endHM != "" {
		eh, em, perr := parseHM(endHM)
		if perr != nil {
			return time.Time{}, false, time.Time{}, false, fmt.Errorf("parse end time %q: %w", timeText, perr)
		}
		end = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), eh, em, 0, 0, candidate.Location())
		return start, false, end, false, nil
	}

	// A start time is known but no end time on the page: estimate start +
	// 2h, same fallback the reference implementations use.
	return start, true, start.Add(2 * time.Hour), false, nil
}

func splitTimeRange(s string) (start, end string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", false
	}
	parts := strings.SplitN(s, "-", 2)
	start = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		end = strings.TrimSpace(parts[1])
	}
	return start, end, true
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
// ajaxEventsPath) until a page yields no events or maxHistoryPages is hit.
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
		fragment, err := c.fetchEventsAjaxPage(page*eventsPageSize, reference)
		if err != nil {
			return nil, err
		}
		if len(fragment) == 0 {
			break
		}
		events = append(events, fragment...)
		reference = fragment[len(fragment)-1].Start
	}

	return events, nil
}

func (c *Client) fetchEventsAjaxPage(offset int, reference time.Time) ([]Event, error) {
	form := url.Values{
		"offset": {strconv.Itoa(offset)},
		"old":    {"true"},
	}
	body, err := c.postForm(ajaxEventsPath, form)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: fetch events page at offset %d: %w", offset, err)
	}
	defer body.Close()
	return parseEventsFragment(body, reference)
}
