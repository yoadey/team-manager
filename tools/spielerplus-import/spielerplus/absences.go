package spielerplus

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// absencesPath is confirmed from a HAR capture of a live session (response
// body included): GET /absence renders every absence type's list in a
// single page load, organized into tab-panes
// (`<div id="absence-tab0..5" class="tab-pane">`). tab0 ("Aktuell") is a
// filtered view of only currently-relevant absences and is skipped here -
// tabs 1-5 (Urlaub/Krank-Verletzt/Inaktiv/Sonstige/"1 Tag pro Woche")
// together contain every absence, including past ones, with no separate
// pagination call observed. absenceListSelector below excludes tab0.
const absencesPath = "/absence"

// recurringTabID is the "1 Tag pro Woche" (weekly-recurring) tab observed
// in the "create absence" type menu (type=5) - see design.md. It was empty
// in the HAR capture this parser is grounded in, so how a populated
// recurring entry actually renders (in particular, which weekday it
// applies to) is unverified; expandRecurringAbsence below is a best-effort
// guess (looking for a German weekday name in the reason text) rather than
// a confirmed selector - see tasks.md 2.5.
const recurringTabID = "absence-tab5"

// Selectors below are confirmed from a HAR capture of a live /absence page
// (response body included):
//
//	<div class="list-item wrapmode">
//	    <a class="list-item-link" href="/absence/update?id={absenceID}"></a>
//	    <div class="member-icon-container"><a href="/user/view?id={userID}">...</a></div>
//	    <div class="list-content-label-sublabel">
//	        <div class="list-label-section"><div class="list-label">{Name}</div></div>
//	        <div class="list-sublabel-section hidden-xs"><div class="list-sublabel">{Reason}</div></div>
//	        <div class="list-sublabel-section visible-xs"><div class="list-sublabel">{DateRange}<br>{Reason}</div></div>
//	    </div>
//	    <div class="list-value hidden-xs">{DateRange}</div>
//	</div>
//
// (the hidden-xs/visible-xs sublabel wrapping was seen in two slightly
// different nestings across tabs in the capture; selecting the *first*
// `.list-sublabel` match was consistently the reason-only text in both).
const (
	absenceTabPaneSelector  = `div[id^="absence-tab"]`
	absenceRowSelector      = ".list-item.wrapmode"
	absenceLinkSelector     = ".list-item-link"
	absenceUserLinkSelector = `a[href^="/user/view?id="]`
	absenceDateSelector     = ".list-value"
	absenceReasonSelector   = ".list-sublabel"
)

// absenceDateLayout pair: SpielerPlus shows a range like "02.08 - 09.08.26"
// - the end date carries a 2-digit year, the start date doesn't (assumed to
// share the end date's year, adjusted back one year if that would put start
// after end - e.g. a Dec-to-Jan range).
const (
	absenceFromLayout = "02.01"
	absenceToLayout   = "02.01.06"
)

// rawAbsence is a SpielerPlus absence entry before a recurring one (see
// recurringTabID) is expanded into concrete occurrences.
type rawAbsence struct {
	id        string
	memberID  string
	from      time.Time
	to        time.Time
	reason    string
	recurring bool
}

// parseAbsences parses the full /absence page (all tabs) into raw absence
// entries, skipping tab0's redundant "currently relevant" subset (every
// absence it contains also appears in its own type tab - tab1-5).
func parseAbsences(body io.Reader) ([]rawAbsence, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse absences page: %w", err)
	}

	panes := doc.Find(absenceTabPaneSelector)
	if panes.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no absence tab panes matched selector %q - SpielerPlus markup likely changed, inspect the /absence page and update absenceTabPaneSelector in absences.go", absenceTabPaneSelector)
	}

	var raws []rawAbsence
	var totalRows, skipped int
	panes.Each(func(_ int, pane *goquery.Selection) {
		paneID, _ := pane.Attr("id")
		if paneID == "absence-tab0" {
			return // redundant filtered subset of tabs 1-5, see doc comment
		}
		recurring := paneID == recurringTabID

		pane.Find(absenceRowSelector).Each(func(i int, row *goquery.Selection) {
			totalRows++
			raw, err := parseAbsenceRow(row, recurring)
			if err != nil {
				skipped++
				log.Printf("spielerplus: skipping absence row %d in %s: %v", i, paneID, err)
				return
			}
			raws = append(raws, raw)
		})
	})

	if len(raws) == 0 && totalRows > 0 {
		return nil, fmt.Errorf("spielerplus: %d absence row(s) found but none parsed successfully - selectors likely need adjusting, see logged per-row errors above", totalRows)
	}
	if skipped > 0 {
		log.Printf("spielerplus: parsed %d absence(s), skipped %d that failed to parse", len(raws), skipped)
	}
	return raws, nil
}

func parseAbsenceRow(row *goquery.Selection, recurring bool) (rawAbsence, error) {
	linkHref, _ := row.Find(absenceLinkSelector).First().Attr("href")
	id := idFromQueryParam(linkHref, "id")
	if id == "" {
		return rawAbsence{}, fmt.Errorf("missing absence id")
	}

	userHref, _ := row.Find(absenceUserLinkSelector).First().Attr("href")
	memberID := userIDFromHref(userHref)
	if memberID == "" {
		return rawAbsence{}, fmt.Errorf("absence %s: missing member id", id)
	}

	dateText := strings.TrimSpace(row.Find(absenceDateSelector).First().Text())
	from, to, err := parseAbsenceDateRange(dateText)
	if err != nil {
		return rawAbsence{}, fmt.Errorf("absence %s: %w", id, err)
	}

	reason := strings.TrimSpace(row.Find(absenceReasonSelector).First().Text())

	return rawAbsence{id: id, memberID: memberID, from: from, to: to, reason: reason, recurring: recurring}, nil
}

// parseAbsenceDateRange parses SpielerPlus's "DD.MM - DD.MM.YY" range
// format (see absenceFromLayout/absenceToLayout doc comment).
func parseAbsenceDateRange(s string) (from, to time.Time, err error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("expected \"DD.MM - DD.MM.YY\", got %q", s)
	}
	fromText, toText := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	to, err = time.Parse(absenceToLayout, toText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse end date %q: %w", toText, err)
	}
	fromNoYear, err := time.Parse(absenceFromLayout, fromText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse start date %q: %w", fromText, err)
	}
	from = time.Date(to.Year(), fromNoYear.Month(), fromNoYear.Day(), 0, 0, 0, 0, time.UTC)
	if from.After(to) {
		from = from.AddDate(-1, 0, 0)
	}
	return from, to, nil
}

// germanWeekdays maps German weekday names/abbreviations (as might appear
// in a recurring absence's reason text) to time.Weekday. Best-effort and
// unverified - see recurringTabID's doc comment.
var germanWeekdays = map[string]time.Weekday{
	"montag": time.Monday, "montags": time.Monday, "mo": time.Monday, "mo.": time.Monday,
	"dienstag": time.Tuesday, "dienstags": time.Tuesday, "di": time.Tuesday, "di.": time.Tuesday,
	"mittwoch": time.Wednesday, "mittwochs": time.Wednesday, "mi": time.Wednesday, "mi.": time.Wednesday,
	"donnerstag": time.Thursday, "donnerstags": time.Thursday, "do": time.Thursday, "do.": time.Thursday,
	"freitag": time.Friday, "freitags": time.Friday, "fr": time.Friday, "fr.": time.Friday,
	"samstag": time.Saturday, "samstags": time.Saturday, "sa": time.Saturday, "sa.": time.Saturday,
	"sonntag": time.Sunday, "sonntags": time.Sunday, "so": time.Sunday, "so.": time.Sunday,
}

// detectWeekday looks for a German weekday name/abbreviation as a
// whole word in reason. Best-effort - see recurringTabID's doc comment.
func detectWeekday(reason string) (time.Weekday, bool) {
	for _, word := range strings.Fields(strings.ToLower(reason)) {
		word = strings.Trim(word, `.,;:"'()`)
		if wd, ok := germanWeekdays[word]; ok {
			return wd, true
		}
	}
	return 0, false
}

// expandAbsences turns raw absences into concrete Absence occurrences.
// Non-recurring absences pass through unchanged. A recurring entry
// (recurringTabID) whose reason names a weekday is expanded into one
// single-day Absence per matching weekday within [from, until] -
// Teamverwaltung has no recurrence concept for absences, so history is
// materialized as concrete dates instead (see design.md). A recurring
// entry with no detectable weekday is imported as a single literal range
// (its from/to as-is) and logged, since there's no confirmed way to
// determine its actual recurrence pattern (see recurringTabID's doc
// comment).
func expandAbsences(raws []rawAbsence, until time.Time) []Absence {
	var out []Absence
	for _, r := range raws {
		if !r.recurring {
			out = append(out, Absence{ID: r.id, MemberID: r.memberID, From: r.from, To: r.to, Reason: r.reason})
			continue
		}

		weekday, ok := detectWeekday(r.reason)
		if !ok {
			log.Printf("spielerplus: recurring absence %s: no weekday detected in reason %q, importing as a single literal date range instead of expanding", r.id, r.reason)
			out = append(out, Absence{ID: r.id, MemberID: r.memberID, From: r.from, To: r.to, Reason: r.reason})
			continue
		}

		end := r.to
		if until.Before(end) {
			end = until
		}
		for d := firstOccurrence(r.from, weekday); !d.After(end); d = d.AddDate(0, 0, 7) {
			out = append(out, Absence{
				ID:       fmt.Sprintf("%s:%s", r.id, d.Format("2006-01-02")),
				MemberID: r.memberID,
				From:     d,
				To:       d,
				Reason:   r.reason,
			})
		}
	}
	return out
}

func firstOccurrence(from time.Time, weekday time.Weekday) time.Time {
	daysUntil := (int(weekday) - int(from.Weekday()) + 7) % 7
	return from.AddDate(0, 0, daysUntil)
}

// idFromQueryParam extracts a query parameter from a relative href, e.g.
// idFromQueryParam("/absence/update?id=123", "id") == "123".
func idFromQueryParam(href, param string) string {
	i := strings.Index(href, "?")
	if i == -1 {
		return ""
	}
	values := href[i+1:]
	for _, kv := range strings.Split(values, "&") {
		k, v, found := strings.Cut(kv, "=")
		if found && k == param {
			return v
		}
	}
	return ""
}

// FetchAbsences fetches and parses the absences page, expanding any
// recurring entries into concrete occurrences up to now.
func (c *Client) FetchAbsences(now time.Time) ([]Absence, error) {
	body, err := c.get(absencesPath)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	raws, err := parseAbsences(body)
	if err != nil {
		return nil, err
	}
	return expandAbsences(raws, now), nil
}
