package spielerplus

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// catalogRowHTML mirrors the confirmed markup from a HAR capture of a live
// /punishment-catalog/index page.
func catalogRowHTML(id, label, value string) string {
	return fmt.Sprintf(`<div class="list-item small" data-key="%s">
		<a class="list-item-link" href="/punishment-catalog/index"></a>
		<div class="list-content"><div class="list-label wrapmode">%s</div></div>
		<div class="list-value no-ellipsis">%s</div>
	</div>`, id, label, value)
}

func TestParsePenaltyCatalog(t *testing.T) {
	html := catalogRowHTML("1", "Trainingsregeln", "1,00 €") +
		catalogRowHTML("2", "Pünktlichkeit", "1,00 €")
	entries, err := ParsePenaltyCatalog(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParsePenaltyCatalog() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0] != (PenaltyCatalogEntry{ID: "1", Label: "Trainingsregeln", AmountCents: 100}) {
		t.Errorf("entries[0] = %+v", entries[0])
	}
}

func TestParsePenaltyCatalog_NoRows(t *testing.T) {
	if _, err := ParsePenaltyCatalog(strings.NewReader(`<html><body>empty</body></html>`)); err == nil {
		t.Fatal("expected an error when no catalog rows match the selector")
	}
}

// paymentRowHTML mirrors the confirmed markup from a HAR capture of a live
// /punishments/index page: the member is identified by name only, no
// id/link anywhere (see PenaltyAssignment's doc comment).
func paymentRowHTML(id, memberName, dateDDMMYYYY, reason, value, iconClass string) string {
	return fmt.Sprintf(`<div class="list-item" data-key="%s">
		<a class="btnOpenModal list-item-link" href="/punishments/index" data-modal-action="/punishments/view?id=%s"></a>
		<div class="list-icon size-32"><button class="btn-payment"><i class="%s sp fa-fw fa-lg"></i></button></div>
		<div class="list-content-label-sublabel">
			<div class="list-label-section"><div class="list-label">%s</div></div>
			<div class="list-sublabel-section"><div class="list-sublabel"><b>%s</b><span class="dot"></span>%s</div></div>
		</div>
		<div class="list-value no-ellipsis"><span>%s</span></div>
	</div>`, id, id, iconClass, memberName, dateDDMMYYYY, reason, value)
}

func TestParsePenaltyAssignments(t *testing.T) {
	html := paymentRowHTML("1", "Anna Test", "14.07.2026", "Organisatorisches", "1,00 €", "fa-circle") +
		paymentRowHTML("2", "Ben Beispiel", "10.06.2026", "Pünktlichkeit", "2,50 €", "fa-circle-check")

	doc := mustDoc(t, html)
	assignments, err := parsePaymentsPage(doc)
	if err != nil {
		t.Fatalf("parsePaymentsPage() error = %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("got %d assignments, want 2", len(assignments))
	}

	unpaid := assignments[0]
	if unpaid.ID != "1" || unpaid.MemberName != "Anna Test" || unpaid.Reason != "Organisatorisches" || unpaid.Paid || unpaid.AmountCents != 100 {
		t.Errorf("unpaid = %+v", unpaid)
	}
	wantDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
	if !unpaid.Date.Equal(wantDate) {
		t.Errorf("unpaid.Date = %v, want %v", unpaid.Date, wantDate)
	}

	paid := assignments[1]
	if !paid.Paid || paid.Reason != "Pünktlichkeit" || paid.AmountCents != 250 {
		t.Errorf("paid = %+v", paid)
	}
}

func TestParsePenaltyAssignments_EmptyIsNotAnError(t *testing.T) {
	// Confirmed from a live account with no assigned punishments yet: the
	// page renders an explicit empty-state placeholder, not a broken
	// layout - this must not be treated as a parse failure.
	doc := mustDoc(t, `<html><body><div class="empty">Bisher wurden keine Strafen verteilt.</div></body></html>`)
	assignments, err := parsePaymentsPage(doc)
	if err != nil {
		t.Fatalf("parsePaymentsPage() error = %v, want no error for an empty punishments list", err)
	}
	if len(assignments) != 0 {
		t.Errorf("assignments = %+v, want none", assignments)
	}
}
