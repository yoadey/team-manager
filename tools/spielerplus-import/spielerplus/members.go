package spielerplus

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// membersPath is grounded in a HAR capture of a live session: GET /team
// returned 200 text/html when the roster was viewed (a separate GET
// /site/team was also seen and returned 200 - possibly a team-switcher or
// settings page rather than the roster; /team is used here as the more
// likely candidate for the member list, but this wasn't confirmed by
// response bodies). Selectors below remain unverified guesses - see
// tasks.md 2.4/2.6.
const membersPath = "/team"

// Selectors below are unverified against a live account - see tasks.md
// 2.4/2.6.
const (
	memberRowSelector   = "[data-user-id]"
	memberNameSelector  = ".member-name, .name"
	memberEmailSelector = ".member-email, .email, [data-email]"
	memberRoleSelector  = ".member-role, .role"
)

// ParseMembers parses the team roster page into Member records.
func ParseMembers(body io.Reader) ([]Member, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse members page: %w", err)
	}

	rows := doc.Find(memberRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no member rows matched selector %q - this page/selector is an unverified guess (see tasks.md 2.4), inspect the real roster page and update members.go", memberRowSelector)
	}

	// See events.go's ParseEvents for why individually malformed rows are
	// logged and skipped rather than aborting the whole roster - a single
	// member missing an email shouldn't block importing everyone else.
	var members []Member
	var skipped int
	rows.Each(func(i int, row *goquery.Selection) {
		id, ok := row.Attr("data-user-id")
		if !ok || id == "" {
			skipped++
			log.Printf("spielerplus: skipping member row %d: missing data-user-id", i)
			return
		}
		name := strings.TrimSpace(row.Find(memberNameSelector).First().Text())
		email := strings.TrimSpace(row.Find(memberEmailSelector).First().Text())
		if email == "" {
			email, _ = row.Find(memberEmailSelector).First().Attr("data-email")
			email = strings.TrimSpace(email)
		}
		role := strings.TrimSpace(row.Find(memberRoleSelector).First().Text())

		if name == "" || email == "" {
			skipped++
			log.Printf("spielerplus: skipping member row %d (user %s): missing name or email", i, id)
			return
		}
		members = append(members, Member{ID: id, Name: name, Email: email, Role: role})
	})

	if len(members) == 0 {
		return nil, fmt.Errorf("spielerplus: %d member row(s) matched selector %q but none parsed successfully - selectors likely need adjusting, see logged per-row errors above", rows.Length(), memberRowSelector)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d member(s), skipped %d that failed to parse", len(members), skipped)
	}
	return members, nil
}

// FetchMembers fetches and parses the team roster.
func (c *Client) FetchMembers() ([]Member, error) {
	body, err := c.get(membersPath)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return ParseMembers(body)
}
