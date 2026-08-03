package spielerplus

import (
	"encoding/json"
	"strings"
	"testing"
)

// participationFixture wraps html the same way a real ajaxgetparticipation
// response does ({"html": "..."}) - mirrors what a HAR capture of a live
// session showed.
func participationFixture(t *testing.T, html string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{"html": html})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseAttendance(t *testing.T) {
	html := `
	<div class="collapse in" id="99-parti-collapse">
		<div class="participation-list-user">
			<a class="participation-list-user-photo" href="/user/view?id=1"></a>
			<div class="participation-list-user-infos"><div class="participation-list-user-name">No Response</div></div>
		</div>
	</div>
	<div class="collapse in" id="1-parti-collapse">
		<div class="participation-list-user">
			<a class="participation-list-user-photo" href="/user/view?id=2"></a>
			<div class="participation-list-user-infos"><div class="participation-list-user-name">Accepted</div></div>
		</div>
	</div>
	<div class="collapse in" id="2-parti-collapse">
		<div class="participation-list-user">
			<a class="participation-list-user-photo" href="/user/view?id=3"></a>
			<div class="participation-list-user-infos"><div class="participation-list-user-name">Unsure</div></div>
		</div>
	</div>
	<div class="collapse in" id="0-parti-collapse">
		<div class="participation-list-user">
			<a class="participation-list-user-photo" href="/user/view?id=4"></a>
			<div class="participation-list-user-infos">
				<div class="participation-list-user-name">Declined</div>
				<div class="participation-list-user-reason reason-training-42-4"><div class="reason-text">Urlaub</div></div>
			</div>
		</div>
	</div>
	<div class="collapse in" id="3-parti-collapse">
		<div class="participation-list-user">
			<a class="participation-list-user-photo" href="/user/view?id=5"></a>
			<div class="participation-list-user-infos"><div class="participation-list-user-name">Not Nominated</div></div>
		</div>
	</div>`

	records, err := ParseAttendance(strings.NewReader(participationFixture(t, html)), "42")
	if err != nil {
		t.Fatalf("ParseAttendance() error = %v", err)
	}
	if len(records) != 5 {
		t.Fatalf("got %d records, want 5: %+v", len(records), records)
	}

	want := map[string]ParticipationStatus{
		"1": ParticipationNoResponse,
		"2": ParticipationAccepted,
		"3": ParticipationUnsure,
		"4": ParticipationDeclined,
		"5": ParticipationNotNominated,
	}
	for _, rec := range records {
		if rec.EventID != "42" {
			t.Errorf("record %+v: EventID = %q, want 42", rec, rec.EventID)
		}
		if rec.Status != want[rec.MemberID] {
			t.Errorf("member %s: status = %q, want %q", rec.MemberID, rec.Status, want[rec.MemberID])
		}
	}

	for _, rec := range records {
		if rec.MemberID == "4" && rec.Reason != "Urlaub" {
			t.Errorf("member 4: Reason = %q, want Urlaub", rec.Reason)
		}
		if rec.MemberID != "4" && rec.Reason != "" {
			t.Errorf("member %s: Reason = %q, want empty", rec.MemberID, rec.Reason)
		}
	}
}

func TestParseAttendance_EmptyGroupsAreFine(t *testing.T) {
	html := `<div class="collapse in" id="1-parti-collapse"></div>`
	records, err := ParseAttendance(strings.NewReader(participationFixture(t, html)), "42")
	if err != nil {
		t.Fatalf("ParseAttendance() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
}

func TestParseAttendance_NoRowsMatched(t *testing.T) {
	_, err := ParseAttendance(strings.NewReader(participationFixture(t, `<html><body>empty</body></html>`)), "1")
	if err == nil {
		t.Fatal("expected an error when no participation-status groups match the selector")
	}
}

func TestParseAttendance_InvalidJSON(t *testing.T) {
	_, err := ParseAttendance(strings.NewReader("not json"), "1")
	if err == nil {
		t.Fatal("expected an error for a non-JSON response body")
	}
}
