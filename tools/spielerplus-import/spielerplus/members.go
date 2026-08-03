package spielerplus

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// membersPath and userViewPathFmt are confirmed from a HAR capture of a
// live session (response bodies included): GET /team lists the roster
// (name, SpielerPlus role, and a link to each member's own profile), but
// does NOT show email addresses. Each member's email is only available on
// their own profile page (GET /user/view?id=<id>), under a labeled
// "E-Mail Adresse" field rendered as a `mailto:` link - matched by the
// mailto: href itself rather than the German label text, so this doesn't
// depend on the account's display language. Fetching one profile page per
// member is an N+1 cost this importer accepts (a few dozen extra requests
// for a typical club roster, run once).
//
// This assumes the authenticated account (trainer/admin) has permission to
// see every member's contact details on their profile page - only the
// account's own profile was captured in the HAR, so it's unverified
// whether another member's email is shown the same way (privacy settings
// or role-based visibility could hide it) - see tasks.md 2.4.
const (
	membersPath     = "/team"
	userViewPathFmt = "/user/view?id=%s"
)

// Selectors below are confirmed from a HAR capture of a live /team page
// and a live /user/view page (response bodies included).
const (
	memberRowSelector  = "#pjax-members .team-list-item, .team-list-item"
	memberLinkSelector = ".list-item-link"
	memberNameSelector = ".list-label-section .list-label"
	memberRoleSelector = ".user-role .user-role-item"
	memberMailSelector = `a[href^="mailto:"]`
)

// ParseMembers parses the /team roster page into Member records (without
// Email - see FetchMembers, which fills that in via a per-member profile
// fetch).
func ParseMembers(body io.Reader) ([]Member, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse members page: %w", err)
	}

	rows := doc.Find(memberRowSelector)
	if rows.Length() == 0 {
		return nil, fmt.Errorf("spielerplus: no member rows matched selector %q - SpielerPlus markup likely changed, inspect the /team page and update memberRowSelector in members.go", memberRowSelector)
	}

	// See events.go's ParseEvents for why individually malformed rows are
	// logged and skipped rather than aborting the whole roster.
	var members []Member
	var skipped int
	rows.Each(func(i int, row *goquery.Selection) {
		href, _ := row.Find(memberLinkSelector).First().Attr("href")
		id := userIDFromHref(href)
		name := strings.TrimSpace(row.Find(memberNameSelector).First().Text())
		role := strings.TrimSpace(row.Find(memberRoleSelector).First().Text())

		if id == "" || name == "" {
			skipped++
			log.Printf("spielerplus: skipping member row %d: missing user id or name", i)
			return
		}
		members = append(members, Member{ID: id, Name: name, Role: role})
	})

	if len(members) == 0 {
		return nil, fmt.Errorf("spielerplus: %d member row(s) matched selector %q but none parsed successfully - selectors likely need adjusting, see logged per-row errors above", rows.Length(), memberRowSelector)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d member(s), skipped %d that failed to parse", len(members), skipped)
	}
	return members, nil
}

// ParseMemberEmail parses a /user/view profile page for its "E-Mail
// Adresse" mailto: link. Returns "" if the page has no such link (e.g. the
// viewer lacks permission to see it).
func ParseMemberEmail(body io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", fmt.Errorf("spielerplus: parse user profile page: %w", err)
	}
	href, ok := doc.Find(memberMailSelector).First().Attr("href")
	if !ok {
		return "", nil
	}
	return strings.TrimPrefix(href, "mailto:"), nil
}

// FetchMembers fetches the team roster and then, for each member, their
// profile page's email address (see the package doc comment on
// membersPath for why that needs a separate request). A member whose
// profile page has no visible email is skipped and logged, rather than
// imported with an empty/unusable email.
func (c *Client) FetchMembers() ([]Member, error) {
	body, err := c.get(membersPath)
	if err != nil {
		return nil, err
	}
	roster, err := ParseMembers(body)
	body.Close()
	if err != nil {
		return nil, err
	}

	var members []Member
	var skipped int
	for _, m := range roster {
		email, err := c.fetchMemberEmail(m.ID)
		if err != nil {
			return nil, fmt.Errorf("spielerplus: fetch profile for member %s (%s): %w", m.Name, m.ID, err)
		}
		if email == "" {
			skipped++
			log.Printf("spielerplus: skipping member %s (%s): no email visible on their profile page", m.Name, m.ID)
			continue
		}
		m.Email = email
		members = append(members, m)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d member(s), skipped %d with no visible email", len(members), skipped)
	}
	return members, nil
}

func (c *Client) fetchMemberEmail(userID string) (string, error) {
	body, err := c.get(fmt.Sprintf(userViewPathFmt, userID))
	if err != nil {
		return "", err
	}
	defer body.Close()
	return ParseMemberEmail(body)
}

// userIDFromHref extracts the "id" query parameter from a
// "/user/view?id=123" style href. Shared with attendance.go's participant
// list parsing.
func userIDFromHref(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return u.Query().Get("id")
}
