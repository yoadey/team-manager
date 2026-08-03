package spielerplus

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// eventsPath is the known events list endpoint, confirmed against the
// christianwehe/calendar-sync reference implementation.
const eventsPath = "/events"

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

// ParseEvents parses the /events page body into structured events. now is
// used to resolve SpielerPlus's year-less dates (it displays e.g. "Di,
// 12.08." without a year) the same way the reference implementations do:
// assume the list is in chronological order relative to "now" and infer the
// year from that.
func ParseEvents(body io.Reader, now time.Time) ([]Event, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse events page: %w", err)
	}

	rows := doc.Find(eventRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no event rows matched selector %q - SpielerPlus markup likely changed, inspect the /events page and update eventRowSelector in events.go", eventRowSelector)
	}

	var events []Event
	var parseErrs []string
	rows.Each(func(i int, row *goquery.Selection) {
		ev, err := parseEventRow(row, now)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Sprintf("row %d: %v", i, err))
			return
		}
		events = append(events, ev)
	})

	if len(parseErrs) > 0 {
		return events, fmt.Errorf("spielerplus: failed to parse %d/%d event row(s): %s", len(parseErrs), rows.Length(), strings.Join(parseErrs, "; "))
	}
	return events, nil
}

func parseEventRow(row *goquery.Selection, now time.Time) (Event, error) {
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
	start, endIsEstimated, end, err := parseDateTime(dateText, timeText, now)
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
	}, nil
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

// parseDateTime resolves a SpielerPlus "DD.MM." date plus an optional
// "HH:MM - HH:MM" (or single "HH:MM") time string into a start/end time. The
// year is inferred relative to now: if the resulting date would be more than
// ~2 months in the past, it's assumed to belong to next year, matching how
// the reference implementations handle SpielerPlus's upcoming-only event
// list. For historical imports where dates are further in the past, prefer
// a date source that includes the year if/when one is found on the site.
func parseDateTime(dateText, timeText string, now time.Time) (start time.Time, endIsEstimated bool, end time.Time, err error) {
	if dateText == "" {
		return time.Time{}, false, time.Time{}, fmt.Errorf("missing date")
	}
	parsed, err := time.Parse(dayMonthLayout, dateText)
	if err != nil {
		return time.Time{}, false, time.Time{}, fmt.Errorf("parse date %q: %w", dateText, err)
	}

	year := now.Year()
	candidate := time.Date(year, parsed.Month(), parsed.Day(), 0, 0, 0, 0, now.Location())
	if candidate.Before(now.AddDate(0, -2, 0)) {
		candidate = candidate.AddDate(1, 0, 0)
	}

	startHM, endHM, hasTime := splitTimeRange(timeText)
	if hasTime {
		sh, sm, perr := parseHM(startHM)
		if perr != nil {
			return time.Time{}, false, time.Time{}, fmt.Errorf("parse start time %q: %w", timeText, perr)
		}
		start = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), sh, sm, 0, 0, now.Location())
	} else {
		start = candidate
	}

	if endHM != "" {
		eh, em, perr := parseHM(endHM)
		if perr != nil {
			return time.Time{}, false, time.Time{}, fmt.Errorf("parse end time %q: %w", timeText, perr)
		}
		end = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), eh, em, 0, 0, now.Location())
		return start, false, end, nil
	}

	// No end time on the page: estimate start + 2h, same fallback the
	// reference implementations use.
	return start, true, start.Add(2 * time.Hour), nil
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

// FetchEvents fetches and parses the current /events listing. SpielerPlus's
// default view only shows upcoming events (see package docs); reaching
// historical events for the migration requires whatever archive/filter view
// is discovered during live testing (tasks.md 2.2) - this method is the
// integration point to extend once that's known.
func (c *Client) FetchEvents(now time.Time) ([]Event, error) {
	body, err := c.get(eventsPath)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return ParseEvents(body, now)
}
