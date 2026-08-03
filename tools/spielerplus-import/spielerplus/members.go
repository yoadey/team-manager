package spielerplus

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// membersPath has no reference implementation to confirm it against (see
// design.md) - this is a best guess based on SpielerPlus's common URL
// naming and MUST be confirmed against a live account before relying on it
// (tasks.md 2.4).
const membersPath = "/squad/members"

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

	var members []Member
	var errs []string
	rows.Each(func(i int, row *goquery.Selection) {
		id, ok := row.Attr("data-user-id")
		if !ok || id == "" {
			errs = append(errs, fmt.Sprintf("row %d: missing data-user-id", i))
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
			errs = append(errs, fmt.Sprintf("row %d (user %s): missing name or email", i, id))
			return
		}
		members = append(members, Member{ID: id, Name: name, Email: email, Role: role})
	})

	if len(errs) > 0 {
		return members, fmt.Errorf("spielerplus: failed to parse %d/%d member row(s): %s", len(errs), rows.Length(), strings.Join(errs, "; "))
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
