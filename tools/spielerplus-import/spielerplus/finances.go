package spielerplus

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// cashboxPath is the team's general Kasse (cashbox) ledger. Confirmed from
// a HAR capture of a live session: GET /cashbox lists transactions as
// `.list-item[data-key]` rows - the same list-item convention already
// confirmed for /team and /absence - paginated via a standard
// `<ul class="pagination">` with "/cashbox/index?page=N&per-page=25" links
// (see paginationLastPage), unlike events' offset-based ajax pagination.
const (
	cashboxPath       = "/cashbox"
	cashboxPagePath   = "/cashbox/index"
	cashboxPerPage    = 25
	cashboxDateLayout = "02.01.2006" // full year, unlike events' year-less "DD.MM"
)

const (
	transactionRowSelector   = ".list-item[data-key]"
	transactionTitleSelector = ".list-label-section .list-label"
	transactionDateSelector  = ".list-sublabel-section .list-sublabel"
	transactionValueSelector = ".list-value span"
)

// Transaction is a single Kasse (cashbox) ledger entry.
type Transaction struct {
	// ID is SpielerPlus's own transaction id ("data-key" on the row), used
	// as the external key for import idempotency (transactions has no
	// natural unique key of its own).
	ID    string
	Title string
	Date  time.Time
	// AmountCents is always positive; Type says whether it was income or
	// expense - SpielerPlus shows a leading "-" for an expense (confirmed
	// from the capture), matched onto Teamverwaltung's transactions.type
	// CHECK ('income', 'expense').
	AmountCents int64
	Type        string
}

func parseTransactionRows(rows *goquery.Selection) (transactions []Transaction, skipped int) {
	rows.Each(func(i int, row *goquery.Selection) {
		id, ok := row.Attr("data-key")
		if !ok || id == "" {
			skipped++
			log.Printf("spielerplus: skipping transaction row %d: missing data-key", i)
			return
		}
		title := strings.TrimSpace(row.Find(transactionTitleSelector).First().Text())
		dateText := strings.TrimSpace(row.Find(transactionDateSelector).First().Text())
		valueText := strings.TrimSpace(row.Find(transactionValueSelector).First().Text())
		if title == "" || dateText == "" || valueText == "" {
			skipped++
			log.Printf("spielerplus: skipping transaction row %d (%s): missing title, date, or amount", i, id)
			return
		}
		date, err := time.Parse(cashboxDateLayout, dateText)
		if err != nil {
			skipped++
			log.Printf("spielerplus: skipping transaction row %d (%s): parse date %q: %v", i, id, dateText, err)
			return
		}
		cents, err := parseEuroCents(valueText)
		if err != nil {
			skipped++
			log.Printf("spielerplus: skipping transaction row %d (%s): parse amount %q: %v", i, id, valueText, err)
			return
		}
		txType := "income"
		if cents < 0 {
			txType = "expense"
			cents = -cents
		}
		transactions = append(transactions, Transaction{ID: id, Title: title, Date: date, AmountCents: cents, Type: txType})
	})
	return transactions, skipped
}

func parseTransactionsPage(doc *goquery.Document) ([]Transaction, error) {
	rows := doc.Find(transactionRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no transaction rows matched selector %q - SpielerPlus markup likely changed, inspect the /cashbox page and update transactionRowSelector in finances.go", transactionRowSelector)
	}
	transactions, skipped := parseTransactionRows(rows)
	if len(transactions) == 0 {
		return nil, fmt.Errorf("spielerplus: %d transaction row(s) matched selector %q but none parsed successfully - selectors likely need adjusting, see logged per-row errors above", rows.Length(), transactionRowSelector)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d transaction(s), skipped %d that failed to parse", len(transactions), skipped)
	}
	return transactions, nil
}

// FetchTransactions fetches every page of the team's cashbox ledger.
func (c *Client) FetchTransactions() ([]Transaction, error) {
	body, err := c.get(cashboxPath)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(body)
	body.Close()
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse cashbox page: %w", err)
	}
	transactions, err := parseTransactionsPage(doc)
	if err != nil {
		return nil, err
	}

	lastPage := paginationLastPage(doc.Selection)
	for page := 2; page <= lastPage; page++ {
		pageBody, err := c.get(fmt.Sprintf("%s?page=%d&per-page=%d", cashboxPagePath, page, cashboxPerPage))
		if err != nil {
			return nil, fmt.Errorf("spielerplus: fetch cashbox page %d: %w", page, err)
		}
		pageDoc, err := goquery.NewDocumentFromReader(pageBody)
		pageBody.Close()
		if err != nil {
			return nil, fmt.Errorf("spielerplus: parse cashbox page %d: %w", page, err)
		}
		pageTransactions, err := parseTransactionsPage(pageDoc)
		if err != nil {
			return nil, fmt.Errorf("spielerplus: cashbox page %d: %w", page, err)
		}
		transactions = append(transactions, pageTransactions...)
	}
	return transactions, nil
}
