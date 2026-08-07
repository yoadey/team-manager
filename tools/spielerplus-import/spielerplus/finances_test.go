package spielerplus

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// transactionRowHTML mirrors the confirmed markup from a HAR capture of a
// live /cashbox page.
func transactionRowHTML(id, title, dateDDMMYYYY, value string) string {
	return fmt.Sprintf(`<div class="list-item" data-key="%s">
		<a class="btnOpenModal list-item-link" href="/cashbox" data-modal-action="/cashbox-transaction/view?id=%s"></a>
		<div class="list-content-label-sublabel">
			<div class="list-label-section"><div class="list-label">%s</div></div>
			<div class="list-sublabel-section"><div class="list-sublabel">%s</div></div>
		</div>
		<div class="list-value no-ellipsis"><span>%s</span></div>
	</div>`, id, id, title, dateDDMMYYYY, value)
}

func TestParseTransactionsPage(t *testing.T) {
	html := `<div id="cashbox">` +
		transactionRowHTML("1", "Geschenk Rene", "05.05.2026", "-6,00 €") +
		transactionRowHTML("2", "Beitrag Anna", "06.01.2026", "20,00 €") +
		`</div>`

	doc := mustDoc(t, html)
	transactions, err := parseTransactionsPage(doc)
	if err != nil {
		t.Fatalf("parseTransactionsPage() error = %v", err)
	}
	if len(transactions) != 2 {
		t.Fatalf("got %d transactions, want 2", len(transactions))
	}

	expense := transactions[0]
	if expense.ID != "1" || expense.Title != "Geschenk Rene" || expense.Type != "expense" || expense.AmountCents != 600 {
		t.Errorf("expense = %+v", expense)
	}
	wantDate := time.Date(2026, time.May, 5, 0, 0, 0, 0, time.UTC)
	if !expense.Date.Equal(wantDate) {
		t.Errorf("expense.Date = %v, want %v", expense.Date, wantDate)
	}

	income := transactions[1]
	if income.Type != "income" || income.AmountCents != 2000 {
		t.Errorf("income = %+v", income)
	}
}

func TestParseTransactionsPage_BadRowSkippedNotFatal(t *testing.T) {
	html := transactionRowHTML("1", "Ok", "05.05.2026", "-6,00 €") +
		`<div class="list-item" data-key="2"></div>` // missing title/date/amount: should be skipped, not fatal

	doc := mustDoc(t, html)
	transactions, err := parseTransactionsPage(doc)
	if err != nil {
		t.Fatalf("parseTransactionsPage() error = %v, want a good row alongside a bad one to succeed", err)
	}
	if len(transactions) != 1 || transactions[0].ID != "1" {
		t.Fatalf("transactions = %+v, want only transaction 1", transactions)
	}
}

func TestParseTransactionsPage_NoRowsMatched(t *testing.T) {
	doc := mustDoc(t, `<html><body>empty</body></html>`)
	if _, err := parseTransactionsPage(doc); err == nil {
		t.Fatal("expected an error when no transaction rows match the selector")
	}
}

func mustDoc(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to build test document: %v", err)
	}
	return doc
}
