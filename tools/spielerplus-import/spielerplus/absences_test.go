package spielerplus

import (
	"strings"
	"testing"
	"time"
)

func TestParseAbsences_NonRecurring(t *testing.T) {
	html := `
	<html><body>
	<div data-absence-id="9" data-user-id="1">
		<span class="from">01.06.2026</span>
		<span class="to">10.06.2026</span>
		<span class="reason">Urlaub</span>
	</div>
	</body></html>`

	raws, err := parseAbsences(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseAbsences() error = %v", err)
	}
	if len(raws) != 1 {
		t.Fatalf("got %d raw absences, want 1", len(raws))
	}
	r := raws[0]
	if r.id != "9" || r.memberID != "1" || r.reason != "Urlaub" || r.recurringWeekday != nil {
		t.Errorf("raw absence = %+v", r)
	}
	if !r.from.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("from = %v", r.from)
	}
	if !r.to.Equal(time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("to = %v", r.to)
	}
}

func TestExpandAbsences_NonRecurringPassesThrough(t *testing.T) {
	raws := []rawAbsence{{
		id:       "9",
		memberID: "1",
		from:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		to:       time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		reason:   "Urlaub",
	}}
	out := expandAbsences(raws, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	if len(out) != 1 || out[0].ID != "9" {
		t.Fatalf("expandAbsences() = %+v", out)
	}
}

func TestExpandAbsences_RecurringWeekday(t *testing.T) {
	monday := time.Monday
	raws := []rawAbsence{{
		id:               "9",
		memberID:         "1",
		from:             time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), // Monday
		to:               time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		reason:           "Immer montags verhindert",
		recurringWeekday: &monday,
	}}
	until := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	out := expandAbsences(raws, until)

	// Mondays in June 2026: 1, 8, 15, 22, 29
	if len(out) != 5 {
		t.Fatalf("got %d occurrences, want 5: %+v", len(out), out)
	}
	for _, occ := range out {
		if occ.From.Weekday() != time.Monday {
			t.Errorf("occurrence %+v is not a Monday", occ)
		}
		if !occ.From.Equal(occ.To) {
			t.Errorf("occurrence %+v: From != To for a single-day expansion", occ)
		}
		if occ.MemberID != "1" || occ.Reason != "Immer montags verhindert" {
			t.Errorf("occurrence %+v: unexpected member/reason", occ)
		}
	}
	if out[0].ID == out[1].ID {
		t.Errorf("expanded occurrences must have distinct IDs, got %q twice", out[0].ID)
	}
}

func TestExpandAbsences_RecurringCappedAtUntil(t *testing.T) {
	monday := time.Monday
	raws := []rawAbsence{{
		id:               "9",
		memberID:         "1",
		from:             time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		to:               time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		recurringWeekday: &monday,
	}}
	until := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	out := expandAbsences(raws, until)
	for _, occ := range out {
		if occ.From.After(until) {
			t.Errorf("occurrence %v is after the until cutoff %v", occ.From, until)
		}
	}
}
