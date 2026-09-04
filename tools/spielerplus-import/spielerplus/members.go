package spielerplus

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

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
	memberRowSelector   = "#pjax-members .team-list-item, .team-list-item"
	memberLinkSelector  = ".list-item-link"
	memberNameSelector  = ".list-label-section .list-label"
	memberRoleSelector  = ".user-role .user-role-item"
	memberMailSelector  = `a[href^="mailto:"]`
	memberPhotoSelector = ".user-icon img"
	// defaultPhotoMarker is a substring of the placeholder silhouette
	// SpielerPlus serves in place of a real photo (confirmed from a HAR
	// capture: ".../images/user/200x200/default.svg") - matched by
	// substring rather than exact path, since the asset hash prefix in the
	// URL changes across deploys.
	defaultPhotoMarker = "default.svg"
)

// The profile page's labeled fields (email, birthday, ...) are confirmed
// from a HAR capture to render as repeated
// `<div class="col-md-4 col-sm-6"><small class="light"><b>Label</b></small>
// <p class="dark">Value</p></div>` blocks - matched by the German label
// text (SpielerPlus has no language-independent marker here, unlike the
// mailto: href used for email), so this only works for a German-language
// account. birthdayLayout ("DD.MM.YYYY") is unverified - no populated
// birthday was present in the captured profile; a field found but not in
// this format is logged and skipped rather than failing the whole member.
const (
	profileFieldBlockSelector = ".col-md-4.col-sm-6"
	profileFieldLabelSelector = "small.light b"
	profileFieldValueSelector = "p.dark"
	birthdayFieldLabel        = "Geburtstag"
	birthdayLayout            = "02.01.2006"
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
		photoURL, _ := row.Find(memberPhotoSelector).First().Attr("src")
		if strings.Contains(photoURL, defaultPhotoMarker) {
			photoURL = ""
		}

		if id == "" || name == "" {
			skipped++
			log.Printf("spielerplus: skipping member row %d: missing user id or name", i)
			return
		}
		members = append(members, Member{ID: id, Name: name, Role: role, PhotoURL: photoURL})
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
	return parseEmailFromDoc(doc.Selection), nil
}

func parseEmailFromDoc(doc *goquery.Selection) string {
	href, ok := doc.Find(memberMailSelector).First().Attr("href")
	if !ok {
		return ""
	}
	return strings.TrimPrefix(href, "mailto:")
}

// ParseMemberBirthday parses a /user/view profile page for its "Geburtstag"
// field (see the profileFieldBlockSelector doc comment). Returns a zero
// time if the field isn't present, isn't in the expected date format, or
// the account's display language isn't German.
func ParseMemberBirthday(body io.Reader) (time.Time, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return time.Time{}, fmt.Errorf("spielerplus: parse user profile page: %w", err)
	}
	return parseBirthdayFromDoc(doc.Selection)
}

func parseBirthdayFromDoc(doc *goquery.Selection) (time.Time, error) {
	var raw string
	doc.Find(profileFieldBlockSelector).EachWithBreak(func(_ int, block *goquery.Selection) bool {
		label := strings.TrimSpace(block.Find(profileFieldLabelSelector).First().Text())
		if !strings.EqualFold(label, birthdayFieldLabel) {
			return true // keep looking
		}
		raw = strings.TrimSpace(block.Find(profileFieldValueSelector).First().Text())
		return false
	})
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(birthdayLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("spielerplus: parse birthday %q: %w", raw, err)
	}
	return t, nil
}

// FetchMembers fetches the team roster and then, for each member, their
// profile page's email address and birthday (see the package doc comment
// on membersPath for why that needs a separate request; both fields are
// read from the same page fetch). A member whose profile page has no
// visible email is skipped and logged, rather than imported with an
// empty/unusable email; a missing/unparseable birthday is not fatal, only
// logged.
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
		email, birthday, err := c.fetchMemberProfile(m.ID)
		if err != nil {
			return nil, fmt.Errorf("spielerplus: fetch profile for member %s (%s): %w", m.Name, m.ID, err)
		}
		if email == "" {
			skipped++
			log.Printf("spielerplus: skipping member %s (%s): no email visible on their profile page", m.Name, m.ID)
			continue
		}
		m.Email = email
		m.Birthday = birthday
		members = append(members, m)
	}
	if skipped > 0 {
		log.Printf("spielerplus: imported %d member(s), skipped %d with no visible email", len(members), skipped)
	}
	return members, nil
}

func (c *Client) fetchMemberProfile(userID string) (email string, birthday time.Time, err error) {
	body, err := c.get(fmt.Sprintf(userViewPathFmt, userID))
	if err != nil {
		return "", time.Time{}, err
	}
	defer body.Close()
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("spielerplus: parse user profile page: %w", err)
	}
	birthday, err = parseBirthdayFromDoc(doc.Selection)
	if err != nil {
		log.Printf("spielerplus: member %s: %v (skipping birthday, not fatal)", userID, err)
		birthday = time.Time{}
	}
	return parseEmailFromDoc(doc.Selection), birthday, nil
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
