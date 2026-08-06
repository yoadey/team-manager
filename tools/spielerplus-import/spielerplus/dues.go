package spielerplus

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// cashboxDuesPath is SpielerPlus's membership-dues matrix: one row per
// member, one column per club-defined, freely-named due/installment (e.g.
// "Teamkasse1", "Fahrtgeld1" - confirmed from a HAR capture of a live
// /cashbox/dues page). Unlike Teamverwaltung's contributions (a fixed
// monthly cycle per member), SpielerPlus's due columns carry no date/month
// of their own anywhere in the page - see FetchDues and importrun, which
// spreads a member's columns across synthetic consecutive months at import
// time (an explicit, documented approximation - see design.md).
const cashboxDuesPath = "/cashbox/dues"

const (
	duesHeaderSelector     = "table.table thead th"
	duesRowSelector        = `table.table tr:has(td.first-column)`
	duesMemberLinkSelector = "td.first-column a"
	duesCellSelector       = "td.text-center button"
)

// toggleCashboxRegexp extracts the (memberID, dueColumnID) pair from a due
// cell's `onClick="toggleCashbox(this, <memberID>, <dueColumnID>)"`
// attribute, confirmed from a HAR capture: dueColumnID is stable per
// column across every member row, and is the only place a due column's own
// id appears (not in the header).
var toggleCashboxRegexp = regexp.MustCompile(`toggleCashbox\(this,\s*(\d+),\s*(\d+)\)`)

// Due is one member's status for one SpielerPlus due column.
type Due struct {
	// ID uniquely identifies this (member, due-column) pair - synthesized
	// as "<dueColumnID>:<memberID>" (SpielerPlus's own dueColumnID is
	// shared across every member's row for the same column).
	ID          string
	MemberID    string
	Label       string
	AmountCents int64
	Paid        bool
	// ColumnIndex is the column's 0-based position on the page (stable
	// across all members) - used by importrun to derive a consistent
	// synthetic month per column, since SpielerPlus gives none.
	ColumnIndex int
}

type duesColumn struct {
	label       string
	amountCents int64
}

// parseDuesColumn reads a header `<th>`'s label+amount, confirmed from a
// HAR capture: `<span title="Teamkasse1">Teamkasse1<br/>20,00 €</span>` -
// the title attribute gives a clean label; goquery's .Text() joins the
// label and amount text nodes with no separator (a <br/> adds none), so
// the amount is recovered by trimming the known label off the front.
func parseDuesColumn(th *goquery.Selection) (duesColumn, error) {
	span := th.Find("span").First()
	label, ok := span.Attr("title")
	if !ok || label == "" {
		return duesColumn{}, fmt.Errorf("missing column label")
	}
	amountText := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(span.Text()), label))
	cents, err := parseEuroCents(amountText)
	if err != nil {
		return duesColumn{}, fmt.Errorf("column %q: parse amount %q: %w", label, amountText, err)
	}
	return duesColumn{label: label, amountCents: cents}, nil
}

// ParseDues parses a full /cashbox/dues page into per-(member,column) Due
// records.
func ParseDues(body io.Reader) ([]Due, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse cashbox dues page: %w", err)
	}

	headerCells := doc.Find(duesHeaderSelector)
	if headerCells.Length() < 2 { // first <th> is the empty "first-column" header above member names
		return nil, fmt.Errorf("spielerplus: no due columns matched selector %q - SpielerPlus markup likely changed, inspect the /cashbox/dues page and update duesHeaderSelector in dues.go", duesHeaderSelector)
	}
	var columns []duesColumn
	headerCells.Each(func(i int, th *goquery.Selection) {
		if i == 0 {
			return
		}
		col, err := parseDuesColumn(th)
		if err != nil {
			log.Printf("spielerplus: skipping dues column %d: %v", i, err)
			columns = append(columns, duesColumn{})
			return
		}
		columns = append(columns, col)
	})

	rows := doc.Find(duesRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no member rows matched selector %q on the dues page - SpielerPlus markup likely changed, inspect the /cashbox/dues page and update duesRowSelector in dues.go", duesRowSelector)
	}

	var dues []Due
	var skipped int
	rows.Each(func(rowIdx int, row *goquery.Selection) {
		href, _ := row.Find(duesMemberLinkSelector).First().Attr("href")
		memberID := userIDFromHref(href)
		if memberID == "" {
			skipped++
			log.Printf("spielerplus: skipping dues row %d: missing member id", rowIdx)
			return
		}
		row.Find(duesCellSelector).Each(func(colIdx int, btn *goquery.Selection) {
			onclick, _ := btn.Attr("onclick")
			m := toggleCashboxRegexp.FindStringSubmatch(onclick)
			if m == nil {
				skipped++
				log.Printf("spielerplus: skipping dues cell (row %d, col %d): unrecognized onClick %q", rowIdx, colIdx, onclick)
				return
			}
			if colIdx >= len(columns) || columns[colIdx].label == "" {
				skipped++
				log.Printf("spielerplus: skipping dues cell (row %d, col %d): no usable column header at this position", rowIdx, colIdx)
				return
			}
			dueColumnID := m[2]
			col := columns[colIdx]
			paid := strings.Contains(btn.Find("i").First().AttrOr("class", ""), "fa-circle-check")
			dues = append(dues, Due{
				ID:          dueColumnID + ":" + memberID,
				MemberID:    memberID,
				Label:       col.label,
				AmountCents: col.amountCents,
				Paid:        paid,
				ColumnIndex: colIdx,
			})
		})
	})

	if len(dues) == 0 {
		return nil, fmt.Errorf("spielerplus: %d row(s) matched selector %q but no dues parsed successfully - selectors likely need adjusting, see logged per-row errors above", rows.Length(), duesRowSelector)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d due(s), skipped %d that failed to parse", len(dues), skipped)
	}
	return dues, nil
}

// FetchDues fetches the team's membership-dues matrix. Confirmed from the
// HAR capture to render in full on a single page fetch, with no pagination
// widget observed (unlike /cashbox).
func (c *Client) FetchDues() ([]Due, error) {
	body, err := c.get(cashboxDuesPath)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return ParseDues(body)
}
