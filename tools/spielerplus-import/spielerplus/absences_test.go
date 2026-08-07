package spielerplus

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// absenceRowHTML mirrors the confirmed markup from a HAR capture of a live
// /absence page: an absence-id link, a member-id link, a "DD.MM - DD.MM.YY"
// date range under .list-value, and a reason under the first .list-sublabel.
func absenceRowHTML(absenceID, memberID, dateRange, reason string) string {
	return fmt.Sprintf(`<div class="list-item wrapmode">
		<a class="list-item-link" href="/absence/update?id=%s"></a>
		<div class="member-icon-container"><a href="/user/view?id=%s"></a></div>
		<div class="list-content-label-sublabel">
			<div class="list-sublabel-section hidden-xs"><div class="list-sublabel">%s</div></div>
			<div class="list-sublabel-section visible-xs"><div class="list-sublabel">%s<br>%s</div></div>
		</div>
		<div class="list-value hidden-xs">%s</div>
	</div>`, absenceID, memberID, reason, dateRange, reason, dateRange)
}

// tabPaneHTML wraps rows the way the real /absence page nests each type's
// list inside a `<div id="absence-tab{n}" class="tab-pane">`.
func tabPaneHTML(tabNum int, rows ...string) string {
	return fmt.Sprintf(`<div id="absence-tab%d" class="tab-pane">%s</div>`, tabNum, strings.Join(rows, ""))
}

func TestParseAbsences_NonRecurring(t *testing.T) {
	html := tabPaneHTML(0, absenceRowHTML("999", "999", "01.06 - 10.06.26", "Urlaub")) + // tab0 is skipped
		tabPaneHTML(1, absenceRowHTML("9", "1", "01.06 - 10.06.26", "Urlaub"))

	raws, err := parseAbsences(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseAbsences() error = %v", err)
	}
	if len(raws) != 1 {
		t.Fatalf("got %d raw absences, want 1 (tab0 must be skipped): %+v", len(raws), raws)
	}
	r := raws[0]
	if r.id != "9" || r.memberID != "1" || r.reason != "Urlaub" || r.recurring {
		t.Errorf("raw absence = %+v", r)
	}
	if !r.from.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("from = %v", r.from)
	}
	if !r.to.Equal(time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("to = %v", r.to)
	}
}

func TestParseAbsences_DateRangeCrossesYearBoundary(t *testing.T) {
	html := tabPaneHTML(1, absenceRowHTML("9", "1", "28.12 - 05.01.26", "Urlaub"))
	raws, err := parseAbsences(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseAbsences() error = %v", err)
	}
	want := time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)
	if !raws[0].from.Equal(want) {
		t.Errorf("from = %v, want %v (year before the Jan 26 end date)", raws[0].from, want)
	}
}

func TestParseAbsences_RecurringTabMarked(t *testing.T) {
	html := tabPaneHTML(5, absenceRowHTML("9", "1", "01.06 - 30.06.26", "Immer montags"))
	raws, err := parseAbsences(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseAbsences() error = %v", err)
	}
	if !raws[0].recurring {
		t.Error("expected a row in absence-tab5 to be marked recurring")
	}
}

func TestParseAbsences_BadRowSkippedNotFatal(t *testing.T) {
	html := tabPaneHTML(1,
		absenceRowHTML("1", "1", "01.06 - 10.06.26", "Urlaub"),
		absenceRowHTML("2", "2", "not-a-date-range", "Urlaub"), // should be skipped, not fatal
	)

	raws, err := parseAbsences(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseAbsences() error = %v, want a good row alongside a bad one to succeed", err)
	}
	if len(raws) != 1 || raws[0].id != "1" {
		t.Fatalf("raws = %+v, want only absence 1 (absence 2's bad row skipped)", raws)
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

func TestExpandAbsences_RecurringWeekdayDetectedFromReason(t *testing.T) {
	raws := []rawAbsence{{
		id:        "9",
		memberID:  "1",
		from:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), // Monday
		to:        time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		reason:    "Immer montags verhindert",
		recurring: true,
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
	raws := []rawAbsence{{
		id:        "9",
		memberID:  "1",
		from:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		to:        time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		reason:    "montags",
		recurring: true,
	}}
	until := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	out := expandAbsences(raws, until)
	if len(out) == 0 {
		t.Fatal("expected at least one occurrence")
	}
	for _, occ := range out {
		if occ.From.After(until) {
			t.Errorf("occurrence %v is after the until cutoff %v", occ.From, until)
		}
	}
}

func TestExpandAbsences_RecurringNoWeekdayDetected(t *testing.T) {
	// No confirmed real example exists for a recurring entry's markup (see
	// recurringTabID's doc comment) - if we can't find a weekday in the
	// reason, fall back to importing the literal range rather than
	// guessing wrong or dropping the record.
	raws := []rawAbsence{{
		id:        "9",
		memberID:  "1",
		from:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		to:        time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		reason:    "keine Ahnung welcher Tag",
		recurring: true,
	}}
	out := expandAbsences(raws, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	if len(out) != 1 || out[0].ID != "9" || !out[0].From.Equal(raws[0].from) || !out[0].To.Equal(raws[0].to) {
		t.Fatalf("expandAbsences() = %+v, want a single literal-range fallback", out)
	}
}

func TestDetectWeekday(t *testing.T) {
	cases := map[string]time.Weekday{
		"Immer montags":       time.Monday,
		"Di.":                 time.Tuesday,
		"jeden Mittwoch frei": time.Wednesday,
		"keine Angabe":        0, // sentinel, checked separately below
	}
	for reason, want := range cases {
		wd, ok := detectWeekday(reason)
		if reason == "keine Angabe" {
			if ok {
				t.Errorf("detectWeekday(%q) = %v, ok=true, want ok=false", reason, wd)
			}
			continue
		}
		if !ok || wd != want {
			t.Errorf("detectWeekday(%q) = %v, ok=%v, want %v, ok=true", reason, wd, ok, want)
		}
	}
}
