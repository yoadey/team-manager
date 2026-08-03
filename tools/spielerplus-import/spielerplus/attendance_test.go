package spielerplus

import (
	"strings"
	"testing"
)

func TestParseAttendance(t *testing.T) {
	html := `
	<html><body>
	<div data-user-id="1"><span class="selected" title="Zugesagt">x</span></div>
	<div data-user-id="2"><span class="selected" title="Unsicher">x</span></div>
	<div data-user-id="3"><span class="selected" title="Absagen / Abwesend">x</span></div>
	<div data-user-id="4"></div>
	</body></html>`

	records, err := ParseAttendance(strings.NewReader(html), "42")
	if err != nil {
		t.Fatalf("ParseAttendance() error = %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("got %d records, want 4", len(records))
	}

	want := map[string]ParticipationStatus{
		"1": ParticipationAccepted,
		"2": ParticipationUnsure,
		"3": ParticipationDeclined,
		"4": ParticipationNoResonse,
	}
	for _, rec := range records {
		if rec.EventID != "42" {
			t.Errorf("record %+v: EventID = %q, want 42", rec, rec.EventID)
		}
		if rec.Status != want[rec.MemberID] {
			t.Errorf("member %s: status = %q, want %q", rec.MemberID, rec.Status, want[rec.MemberID])
		}
	}
}

func TestParseAttendance_NoRowsMatched(t *testing.T) {
	_, err := ParseAttendance(strings.NewReader(`<html><body>empty</body></html>`), "1")
	if err == nil {
		t.Fatal("expected an error when no participant rows match the selector")
	}
}
