package spielerplus

import (
	"strings"
	"testing"
	"time"
)

func TestParseEvents(t *testing.T) {
	html := `
	<html><body>
	<div class="events-list">
		<div data-event-id="101" data-event-type="training">
			<span class="title">Training</span>
			<span class="date">12.08.</span>
			<span class="time">18:00 - 19:30</span>
			<span class="location">Sportplatz</span>
		</div>
		<div data-event-id="102" data-event-type="game">
			<span class="title">Heimspiel</span>
			<span class="date">19.08.</span>
			<span class="location">Stadion</span>
		</div>
	</div>
	</body></html>`

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
	wantStart := time.Date(2026, time.August, 12, 18, 0, 0, 0, time.UTC)
	if !training.Start.Equal(wantStart) {
		t.Errorf("training.Start = %v, want %v", training.Start, wantStart)
	}
	wantEnd := time.Date(2026, time.August, 12, 19, 30, 0, 0, time.UTC)
	if !training.End.Equal(wantEnd) || training.EndIsEstimated {
		t.Errorf("training.End = %v (estimated=%v), want %v (estimated=false)", training.End, training.EndIsEstimated, wantEnd)
	}

	game := events[1]
	if game.ID != "102" || game.Type != EventGame {
		t.Errorf("game event = %+v", game)
	}
	if !game.EndIsEstimated {
		t.Errorf("game.EndIsEstimated = false, want true (no time on page)")
	}
	wantGameEnd := game.Start.Add(2 * time.Hour)
	if !game.End.Equal(wantGameEnd) {
		t.Errorf("game.End = %v, want %v", game.End, wantGameEnd)
	}
}

func TestParseEvents_YearRollover(t *testing.T) {
	// A date far enough in the past relative to "now" is assumed to belong
	// to next year (SpielerPlus's list is upcoming-only).
	html := `<div data-event-id="1"><span class="title">Training</span><span class="date">05.01.</span></div>`
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
	html := `<div data-event-id="1"><span class="date">12.08.</span></div>` // no title
	_, err := ParseEvents(strings.NewReader(html), time.Now())
	if err == nil {
		t.Fatal("expected an error for a row missing its title")
	}
}
