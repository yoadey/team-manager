package spielerplus

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// absencesPath has no reference implementation to confirm it against (see
// design.md) - best guess, MUST be confirmed against a live account before
// relying on it (tasks.md 2.5).
const absencesPath = "/absences"

// Selectors below are unverified against a live account - see tasks.md
// 2.5/2.6.
const (
	absenceRowSelector          = "[data-absence-id]"
	absenceMemberAttr           = "data-user-id"
	absenceFromSelector         = ".absence-from, .from"
	absenceToSelector           = ".absence-to, .to"
	absenceReasonSelector       = ".absence-reason, .reason"
	absenceRecurringWeekdayAttr = "data-recurring-weekday" // 0=Sunday..6=Saturday, empty if not recurring
)

// rawAbsence is a SpielerPlus absence entry before recurring ones are
// expanded into concrete occurrences.
type rawAbsence struct {
	id               string
	memberID         string
	from             time.Time
	to               time.Time
	reason           string
	recurringWeekday *time.Weekday
}

// dateLayout is a plain "DD.MM.YYYY" date, the assumed format for the
// absences page (unlike /events, an absence range is far more likely to
// carry a year since it can span years - unverified, see tasks.md 2.5).
const dateLayout = "02.01.2006"

// ParseAbsences parses the absences page into raw (not-yet-expanded)
// absence entries.
func parseAbsences(body io.Reader) ([]rawAbsence, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse absences page: %w", err)
	}

	rows := doc.Find(absenceRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no absence rows matched selector %q - this page/selector is an unverified guess (see tasks.md 2.5), inspect the real absences page and update absences.go", absenceRowSelector)
	}

	var raws []rawAbsence
	var errs []string
	rows.Each(func(i int, row *goquery.Selection) {
		id, _ := row.Attr("data-absence-id")
		memberID, _ := row.Attr(absenceMemberAttr)
		if id == "" || memberID == "" {
			errs = append(errs, fmt.Sprintf("row %d: missing absence/user id", i))
			return
		}

		fromText := strings.TrimSpace(row.Find(absenceFromSelector).First().Text())
		toText := strings.TrimSpace(row.Find(absenceToSelector).First().Text())
		from, err := time.Parse(dateLayout, fromText)
		if err != nil {
			errs = append(errs, fmt.Sprintf("row %d (absence %s): parse from date %q: %v", i, id, fromText, err))
			return
		}
		to, err := time.Parse(dateLayout, toText)
		if err != nil {
			errs = append(errs, fmt.Sprintf("row %d (absence %s): parse to date %q: %v", i, id, toText, err))
			return
		}

		reason := strings.TrimSpace(row.Find(absenceReasonSelector).First().Text())

		var weekday *time.Weekday
		if wd, ok := row.Attr(absenceRecurringWeekdayAttr); ok && wd != "" {
			parsed, perr := parseWeekday(wd)
			if perr != nil {
				errs = append(errs, fmt.Sprintf("row %d (absence %s): parse recurring weekday %q: %v", i, id, wd, perr))
				return
			}
			weekday = &parsed
		}

		raws = append(raws, rawAbsence{
			id:               id,
			memberID:         memberID,
			from:             from,
			to:               to,
			reason:           reason,
			recurringWeekday: weekday,
		})
	})

	if len(errs) > 0 {
		return raws, fmt.Errorf("spielerplus: failed to parse %d/%d absence row(s): %s", len(errs), rows.Length(), strings.Join(errs, "; "))
	}
	return raws, nil
}

func parseWeekday(s string) (time.Weekday, error) {
	switch strings.TrimSpace(s) {
	case "0":
		return time.Sunday, nil
	case "1":
		return time.Monday, nil
	case "2":
		return time.Tuesday, nil
	case "3":
		return time.Wednesday, nil
	case "4":
		return time.Thursday, nil
	case "5":
		return time.Friday, nil
	case "6":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("expected 0-6, got %q", s)
	}
}

// expandAbsences turns raw absences into concrete Absence occurrences.
// Non-recurring absences pass through unchanged. A recurring "fixed
// weekday" absence (from..to spanning a recurrence window, one specific
// weekday) is expanded into one single-day Absence per matching weekday
// within [from, until] - Teamverwaltung has no recurrence concept for
// absences, so history is materialized as concrete dates instead (see
// design.md).
func expandAbsences(raws []rawAbsence, until time.Time) []Absence {
	var out []Absence
	for _, r := range raws {
		if r.recurringWeekday == nil {
			out = append(out, Absence{
				ID:       r.id,
				MemberID: r.memberID,
				From:     r.from,
				To:       r.to,
				Reason:   r.reason,
			})
			continue
		}

		end := r.to
		if until.Before(end) {
			end = until
		}
		for d := firstOccurrence(r.from, *r.recurringWeekday); !d.After(end); d = d.AddDate(0, 0, 7) {
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
