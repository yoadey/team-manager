package spielerplus

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// punishmentCatalogPath and punishmentPaymentsPath are confirmed from a HAR
// capture of a live session: GET /punishment-catalog/index lists the
// club's penalty catalog (label + amount per entry, no per-member data);
// GET /punishments/index lists every punishment actually assigned to a
// member, using the same `.list-item[data-key]` convention as /cashbox and
// /team, paginated the same way (see paginationLastPage) - unconfirmed for
// this specific page (the captured club had few enough assigned
// punishments that no pagination widget appeared), but following the exact
// same query-param convention already confirmed at /cashbox is a
// reasonable extension by analogy rather than a blind guess.
const (
	punishmentCatalogPath  = "/punishment-catalog/index"
	punishmentPaymentsPath = "/punishments/index"
	// pageQueryParam is the pagination query parameter shared by every
	// paginated SpielerPlus list page, confirmed at /cashbox (see
	// paginationLastPage).
	pageQueryParam = "page"
)

const (
	catalogRowSelector   = ".list-item[data-key]"
	catalogLabelSelector = ".list-label"
	catalogValueSelector = ".list-value"

	paymentRowSelector      = ".list-item[data-key]"
	paymentNameSelector     = ".list-label-section .list-label"
	paymentSublabelSelector = ".list-sublabel-section .list-sublabel"
	paymentDateSelector     = ".list-sublabel-section .list-sublabel b"
	paymentValueSelector    = ".list-value span"
	paymentPaidIconSelector = ".list-icon i"
)

// PenaltyCatalogEntry is one entry in SpielerPlus's penalty catalog
// (Strafenkatalog).
type PenaltyCatalogEntry struct {
	// ID is SpielerPlus's own catalog entry id, used as the external key
	// for import idempotency.
	ID          string
	Label       string
	AmountCents int64
}

// PenaltyAssignment is one punishment actually assigned to a member.
//
// Unlike every other entity this tool imports, SpielerPlus identifies the
// member here by display name only - the assigned-punishment list and its
// own detail view both show just a "Name" field, with no id or profile
// link anywhere (confirmed from a HAR capture). Matching this back to the
// imported roster is therefore a best-effort exact match against
// Member.Name, not a reliable id join - see importrun, which skips and logs
// (rather than fails) an assignment whose name doesn't match any imported
// member.
type PenaltyAssignment struct {
	// ID is SpielerPlus's own assignment id, used as the external key for
	// import idempotency.
	ID          string
	MemberName  string
	Reason      string
	Date        time.Time
	AmountCents int64
	Paid        bool
}

func parseCatalogRows(rows *goquery.Selection) (entries []PenaltyCatalogEntry, skipped int) {
	rows.Each(func(i int, row *goquery.Selection) {
		id, ok := row.Attr("data-key")
		if !ok || id == "" {
			skipped++
			log.Printf("spielerplus: skipping penalty catalog row %d: missing data-key", i)
			return
		}
		label := strings.TrimSpace(row.Find(catalogLabelSelector).First().Text())
		valueText := strings.TrimSpace(row.Find(catalogValueSelector).First().Text())
		if label == "" || valueText == "" {
			skipped++
			log.Printf("spielerplus: skipping penalty catalog row %d (%s): missing label or amount", i, id)
			return
		}
		cents, err := parseEuroCents(valueText)
		if err != nil {
			skipped++
			log.Printf("spielerplus: skipping penalty catalog row %d (%s): parse amount %q: %v", i, id, valueText, err)
			return
		}
		entries = append(entries, PenaltyCatalogEntry{ID: id, Label: label, AmountCents: cents})
	})
	return entries, skipped
}

// ParsePenaltyCatalog parses a full /punishment-catalog/index page.
func ParsePenaltyCatalog(body io.Reader) ([]PenaltyCatalogEntry, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse punishment catalog page: %w", err)
	}
	rows := doc.Find(catalogRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no penalty catalog rows matched selector %q - SpielerPlus markup likely changed, inspect the /punishment-catalog/index page and update catalogRowSelector in punishments.go", catalogRowSelector)
	}
	entries, skipped := parseCatalogRows(rows)
	if len(entries) == 0 {
		return nil, fmt.Errorf("spielerplus: %d penalty catalog row(s) matched selector %q but none parsed successfully - selectors likely need adjusting, see logged per-row errors above", rows.Length(), catalogRowSelector)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d penalty catalog entr(y/ies), skipped %d that failed to parse", len(entries), skipped)
	}
	return entries, nil
}

func parseCatalogPage(doc *goquery.Document) ([]PenaltyCatalogEntry, error) {
	rows := doc.Find(catalogRowSelector)
	if rows.Length() == 0 {
		return nil, nil
	}
	entries, skipped := parseCatalogRows(rows)
	if skipped > 0 {
		log.Printf("spielerplus: imported %d penalty catalog entr(y/ies), skipped %d that failed to parse", len(entries), skipped)
	}
	return entries, nil
}

// FetchPenaltyCatalog fetches every page of the club's penalty catalog.
func (c *Client) FetchPenaltyCatalog() ([]PenaltyCatalogEntry, error) {
	body, err := c.get(punishmentCatalogPath)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(body)
	body.Close()
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse punishment catalog page: %w", err)
	}
	entries, err := parseCatalogPage(doc)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("spielerplus: no penalty catalog entries found on %s - if the club genuinely has an empty catalog this is unexpected, otherwise SpielerPlus markup likely changed", punishmentCatalogPath)
	}

	lastPage := paginationLastPage(doc.Selection)
	for page := 2; page <= lastPage; page++ {
		pageBody, err := c.get(fmt.Sprintf("%s?%s=%d", punishmentCatalogPath, pageQueryParam, page))
		if err != nil {
			return nil, fmt.Errorf("spielerplus: fetch punishment catalog page %d: %w", page, err)
		}
		pageDoc, err := goquery.NewDocumentFromReader(pageBody)
		pageBody.Close()
		if err != nil {
			return nil, fmt.Errorf("spielerplus: parse punishment catalog page %d: %w", page, err)
		}
		pageEntries, err := parseCatalogPage(pageDoc)
		if err != nil {
			return nil, fmt.Errorf("spielerplus: punishment catalog page %d: %w", page, err)
		}
		entries = append(entries, pageEntries...)
	}
	return entries, nil
}

func parsePaymentRows(rows *goquery.Selection) (assignments []PenaltyAssignment, skipped int) {
	rows.Each(func(i int, row *goquery.Selection) {
		id, ok := row.Attr("data-key")
		if !ok || id == "" {
			skipped++
			log.Printf("spielerplus: skipping penalty assignment row %d: missing data-key", i)
			return
		}
		name := strings.TrimSpace(row.Find(paymentNameSelector).First().Text())
		dateText := strings.TrimSpace(row.Find(paymentDateSelector).First().Text())
		valueText := strings.TrimSpace(row.Find(paymentValueSelector).First().Text())
		if name == "" || dateText == "" || valueText == "" {
			skipped++
			log.Printf("spielerplus: skipping penalty assignment row %d (%s): missing name, date, or amount", i, id)
			return
		}
		// The sublabel is "<b>DD.MM.YYYY</b><span class="dot"></span>Reason"
		// with no text-node separator between them (the dot is drawn by
		// CSS, not a character) - the reason is recovered by trimming the
		// already-extracted date text off the front of the full sublabel.
		fullSublabel := strings.TrimSpace(row.Find(paymentSublabelSelector).First().Text())
		reason := strings.TrimSpace(strings.TrimPrefix(fullSublabel, dateText))

		date, err := time.Parse(cashboxDateLayout, dateText)
		if err != nil {
			skipped++
			log.Printf("spielerplus: skipping penalty assignment row %d (%s): parse date %q: %v", i, id, dateText, err)
			return
		}
		cents, err := parseEuroCents(valueText)
		if err != nil {
			skipped++
			log.Printf("spielerplus: skipping penalty assignment row %d (%s): parse amount %q: %v", i, id, valueText, err)
			return
		}
		paid := strings.Contains(row.Find(paymentPaidIconSelector).First().AttrOr("class", ""), "fa-circle-check")

		assignments = append(assignments, PenaltyAssignment{
			ID:          id,
			MemberName:  name,
			Reason:      reason,
			Date:        date,
			AmountCents: cents,
			Paid:        paid,
		})
	})
	return assignments, skipped
}

func parsePaymentsPage(doc *goquery.Document) ([]PenaltyAssignment, error) {
	rows := doc.Find(paymentRowSelector)
	if rows.Length() == 0 {
		return nil, nil // no punishments assigned yet is a valid, non-error state (confirmed: the page shows an explicit empty-state placeholder rather than a broken layout)
	}
	assignments, skipped := parsePaymentRows(rows)
	if skipped > 0 {
		log.Printf("spielerplus: imported %d penalty assignment(s), skipped %d that failed to parse", len(assignments), skipped)
	}
	return assignments, nil
}

// FetchPenaltyAssignments fetches every page of assigned punishments.
func (c *Client) FetchPenaltyAssignments() ([]PenaltyAssignment, error) {
	body, err := c.get(punishmentPaymentsPath)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(body)
	body.Close()
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse punishments page: %w", err)
	}
	assignments, err := parsePaymentsPage(doc)
	if err != nil {
		return nil, err
	}

	lastPage := paginationLastPage(doc.Selection)
	for page := 2; page <= lastPage; page++ {
		pageBody, err := c.get(fmt.Sprintf("%s?%s=%d", punishmentPaymentsPath, pageQueryParam, page))
		if err != nil {
			return nil, fmt.Errorf("spielerplus: fetch punishments page %d: %w", page, err)
		}
		pageDoc, err := goquery.NewDocumentFromReader(pageBody)
		pageBody.Close()
		if err != nil {
			return nil, fmt.Errorf("spielerplus: parse punishments page %d: %w", page, err)
		}
		pageAssignments, err := parsePaymentsPage(pageDoc)
		if err != nil {
			return nil, fmt.Errorf("spielerplus: punishments page %d: %w", page, err)
		}
		assignments = append(assignments, pageAssignments...)
	}
	return assignments, nil
}
