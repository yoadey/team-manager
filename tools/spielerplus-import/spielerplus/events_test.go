package spielerplus

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// panelHTML mirrors the confirmed markup from a HAR capture of a live
// /events page / ajaxgetevents fragment. timeValues holds 0-3
// .event-time-value texts in the real page's fixed order (Treffen, Beginn,
// Ende).
func panelHTML(id, eventType, title, dateDDMM string, timeValues ...string) string {
	var times strings.Builder
	for _, v := range timeValues {
		times.WriteString(`<div class="event-time-item"><div class="event-time-value">` + v + `</div></div>`)
	}
	return `<div class="panel" id="event-` + eventType + `-` + id + `">
		<div class="panel-heading-info"><div class="panel-title">Mo.</div><div class="panel-subtitle">` + dateDDMM + `</div></div>
		<div class="panel-heading-text"><div class="panel-title">` + title + `</div></div>
		<div class="event-time">` + times.String() + `</div>
	</div>`
}

func TestParseEvents(t *testing.T) {
	html := panelHTML("101", "training", "Training", "12.08", "18:50", "19:00", "20:00") +
		panelHTML("102", "game", "Heimspiel", "19.08")

	now := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	events, err := ParseEvents(strings.NewReader(html), now)
	if err != nil {
		t.Fatalf("ParseEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	training := events[0]
	if training.ID != "101" || training.Type != EventTraining || training.Title != "Training" {
		t.Errorf("training event = %+v", training)
	}
	wantStart := time.Date(2026, time.August, 12, 19, 0, 0, 0, time.UTC)
	if !training.Start.Equal(wantStart) {
		t.Errorf("training.Start = %v, want %v", training.Start, wantStart)
	}
	wantEnd := time.Date(2026, time.August, 12, 20, 0, 0, 0, time.UTC)
	if !training.End.Equal(wantEnd) || training.EndIsEstimated {
		t.Errorf("training.End = %v (estimated=%v), want %v (estimated=false)", training.End, training.EndIsEstimated, wantEnd)
	}
	wantMeet := time.Date(2026, time.August, 12, 18, 50, 0, 0, time.UTC)
	if !training.MeetTime.Equal(wantMeet) {
		t.Errorf("training.MeetTime = %v, want %v", training.MeetTime, wantMeet)
	}
	if training.TimeUnknown {
		t.Error("training.TimeUnknown = true, want false (times were on the page)")
	}

	game := events[1]
	if game.ID != "102" || game.Type != EventGame {
		t.Errorf("game event = %+v", game)
	}
	if !game.EndIsEstimated {
		t.Errorf("game.EndIsEstimated = false, want true (no time on page)")
	}
	if !game.TimeUnknown {
		t.Error("game.TimeUnknown = false, want true (no time information at all on the page)")
	}
	wantGameEnd := game.Start.Add(2 * time.Hour)
	if !game.End.Equal(wantGameEnd) {
		t.Errorf("game.End = %v, want %v", game.End, wantGameEnd)
	}
}

func TestParseEvents_TwoTimesMeansNoMeetTime(t *testing.T) {
	html := panelHTML("1", "training", "Training", "12.08", "17:00", "20:00")
	events, err := ParseEvents(strings.NewReader(html), time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ParseEvents() error = %v", err)
	}
	ev := events[0]
	if !ev.MeetTime.IsZero() {
		t.Errorf("MeetTime = %v, want zero (only 2 time values means start/end, no meet)", ev.MeetTime)
	}
	wantStart := time.Date(2026, time.August, 12, 17, 0, 0, 0, time.UTC)
	if !ev.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", ev.Start, wantStart)
	}
}

func TestParseEvents_PlaceholderTimeValues(t *testing.T) {
	// Confirmed against a live account: SpielerPlus renders an explicit
	// "-:-" placeholder for an unconfirmed kickoff time, rather than
	// omitting the .event-time-value element.
	cases := []struct {
		name          string
		times         []string
		wantUnknown   bool
		wantEstimated bool
		wantMeetZero  bool
	}{
		{"all placeholders", []string{"-:-", "-:-", "-:-"}, true, true, true},
		{"start placeholder", []string{"18:50", "-:-", "20:00"}, true, true, true},
		{"end placeholder", []string{"18:50", "19:00", "-:-"}, false, true, false},
		{"meet placeholder", []string{"-:-", "19:00", "20:00"}, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := panelHTML("1", "training", "Training", "12.08", tc.times...)
			events, err := ParseEvents(strings.NewReader(html), time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("ParseEvents() error = %v, want placeholder \"-:-\" values handled without a parse error", err)
			}
			ev := events[0]
			if ev.TimeUnknown != tc.wantUnknown {
				t.Errorf("TimeUnknown = %v, want %v", ev.TimeUnknown, tc.wantUnknown)
			}
			if ev.EndIsEstimated != tc.wantEstimated {
				t.Errorf("EndIsEstimated = %v, want %v", ev.EndIsEstimated, tc.wantEstimated)
			}
			if ev.MeetTime.IsZero() != tc.wantMeetZero {
				t.Errorf("MeetTime = %v, want zero=%v", ev.MeetTime, tc.wantMeetZero)
			}
		})
	}
}

func TestParseEvents_TrailingDateOnTimeValue(t *testing.T) {
	// Confirmed against a live account: a multi-day event's end time
	// renders with a trailing "am DD.MM." (e.g. a tournament ending the
	// next day), not a bare "HH:MM" - End and EndDate must both reflect the
	// later day, not silently collapse back onto the start day.
	html := panelHTML("1", "training", "Training", "16.11", "17:00", "18:00", "17:00 am 17.11.")
	events, err := ParseEvents(strings.NewReader(html), time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ParseEvents() error = %v, want the trailing date on a time value handled without a parse error", err)
	}
	ev := events[0]
	wantEnd := time.Date(2026, time.November, 17, 17, 0, 0, 0, time.UTC)
	if !ev.End.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", ev.End, wantEnd)
	}
	wantEndDate := time.Date(2026, time.November, 17, 0, 0, 0, 0, time.UTC)
	if !ev.EndDate.Equal(wantEndDate) {
		t.Errorf("EndDate = %v, want %v", ev.EndDate, wantEndDate)
	}
}

func TestParseEvents_SameDayTrailingDateLeavesEndDateUnset(t *testing.T) {
	// A trailing date that (for whatever reason) resolves to the same day
	// as the start must not set EndDate - only a genuinely later day makes
	// an event "multi-day".
	html := panelHTML("1", "training", "Training", "16.11", "17:00", "18:00", "19:00 am 16.11.")
	events, err := ParseEvents(strings.NewReader(html), time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ParseEvents() error = %v", err)
	}
	ev := events[0]
	if !ev.EndDate.IsZero() {
		t.Errorf("EndDate = %v, want zero (same-day trailing date)", ev.EndDate)
	}
}

func TestParseHM_IgnoresTrailingText(t *testing.T) {
	h, m, err := parseHM("17:00 am 17.11.")
	if err != nil {
		t.Fatalf("parseHM() error = %v", err)
	}
	if h != 17 || m != 0 {
		t.Errorf("parseHM() = %d:%d, want 17:00", h, m)
	}
}

// eventDetailAddressHTML mirrors the confirmed markup from a HAR capture of
// a live event detail page (e.g. /training/view?id=...): a single
// `.info-area` block labeled "Adresse".
func eventDetailAddressHTML(address string) string {
	return fmt.Sprintf(`<div class="info-area"><div class="info-area-icon"></div>
		<a href="javascript:void(0);"><div class="info-area-content"><h4>Adresse</h4><small>%s</small></div></a>
	</div>`, address)
}

func TestParseEventLocation(t *testing.T) {
	location, err := ParseEventLocation(strings.NewReader(eventDetailAddressHTML("Musterstraße 1, 12345 Musterstadt, Deutschland")))
	if err != nil {
		t.Fatalf("ParseEventLocation() error = %v", err)
	}
	if location != "Musterstraße 1, 12345 Musterstadt, Deutschland" {
		t.Errorf("location = %q", location)
	}
}

func TestParseEventLocation_NotSet(t *testing.T) {
	location, err := ParseEventLocation(strings.NewReader(`<html><body>no info-area here</body></html>`))
	if err != nil {
		t.Fatalf("ParseEventLocation() error = %v, want no error for an event with no address set", err)
	}
	if location != "" {
		t.Errorf("location = %q, want empty", location)
	}
}

func TestParseEventLocation_IgnoresOtherInfoAreaBlocks(t *testing.T) {
	html := `<div class="info-area"><div class="info-area-content"><h4>Wetter</h4><small>Sonnig, 22°C</small></div></div>` +
		eventDetailAddressHTML("Sportplatz 1")
	location, err := ParseEventLocation(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseEventLocation() error = %v", err)
	}
	if location != "Sportplatz 1" {
		t.Errorf("location = %q, want only the Adresse block's value", location)
	}
}

func TestNormalizeTimeValue(t *testing.T) {
	cases := map[string]string{
		"18:00": "18:00",
		"-:-":   "",
		"--:--": "",
		"-":     "",
		"":      "",
	}
	for in, want := range cases {
		if got := normalizeTimeValue(in); got != want {
			t.Errorf("normalizeTimeValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseEvents_BadRowSkippedNotFatal(t *testing.T) {
	html := panelHTML("1", "training", "Training", "12.08") +
		`<div class="panel" id="event-training-2"><div class="panel-heading-info"><div class="panel-subtitle">19.08</div></div></div>` // no title: should be skipped, not fatal

	now := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	events, err := ParseEvents(strings.NewReader(html), now)
	if err != nil {
		t.Fatalf("ParseEvents() error = %v, want a good row alongside a bad one to succeed", err)
	}
	if len(events) != 1 || events[0].ID != "1" {
		t.Fatalf("events = %+v, want only event 1 (event 2's bad row skipped)", events)
	}
}

func TestParseEvents_YearRollover(t *testing.T) {
	// A date far enough from "now" resolves to whichever year keeps it
	// closest - here, forward into next year.
	html := panelHTML("1", "training", "Training", "05.01")
	now := time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC)

	events, err := ParseEvents(strings.NewReader(html), now)
	if err != nil {
		t.Fatalf("ParseEvents() error = %v", err)
	}
	want := time.Date(2027, time.January, 5, 0, 0, 0, 0, time.UTC)
	if !events[0].Start.Equal(want) {
		t.Errorf("Start = %v, want %v (year rollover)", events[0].Start, want)
	}
}

func TestParseEvents_NoRowsMatched(t *testing.T) {
	_, err := ParseEvents(strings.NewReader(`<html><body>no events here</body></html>`), time.Now())
	if err == nil {
		t.Fatal("expected an error when no event rows match the selector")
	}
}

func TestParseEvents_MissingRequiredField(t *testing.T) {
	html := `<div class="panel" id="event-training-1"><div class="panel-heading-info"><div class="panel-subtitle">12.08</div></div></div>` // no title
	_, err := ParseEvents(strings.NewReader(html), time.Now())
	if err == nil {
		t.Fatal("expected an error for a row missing its title")
	}
}

func TestMapEventType(t *testing.T) {
	cases := map[string]EventType{
		"training":   EventTraining,
		"game":       EventGame,
		"tournament": EventTournament,
		"event":      EventOther,
		"unknown":    EventOther,
	}
	for slug, want := range cases {
		if got := mapEventType(slug); got != want {
			t.Errorf("mapEventType(%q) = %q, want %q", slug, got, want)
		}
	}
}
