package spielerplus

import (
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// dashboardPath is a lightweight page (confirmed to exist: "GET
// /dashboard/index" returned 200 text/html in a HAR capture) fetched purely
// to read the active-team nav card below - it's not otherwise used for
// data import.
const dashboardPath = "/dashboard/index"

// activeTeamNameSelector is confirmed from HAR captures with response
// bodies: every page's sidebar navigation carries a
// `<div class="navigation__card">` showing the SpielerPlus account's
// *currently active* team (name under `.navigation__card__title`, plus a
// "Team wechseln" link to /site/select-team). SpielerPlus scopes which
// team's data every other page/endpoint returns by this session-level
// active team - there is no team id in the URLs this importer calls - so
// an account that manages more than one team can silently have the wrong
// one active when its session cookie was captured. FetchActiveTeamName lets
// callers surface the detected name for a confirmation step before
// anything is imported (see main.go's confirmTeam).
const activeTeamNameSelector = ".navigation__card__title"

// ParseActiveTeamName parses the sidebar navigation's active-team name out
// of any full SpielerPlus page body.
func ParseActiveTeamName(body io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", fmt.Errorf("spielerplus: parse page for active team name: %w", err)
	}
	name := strings.TrimSpace(doc.Find(activeTeamNameSelector).First().Text())
	if name == "" {
		return "", fmt.Errorf("spielerplus: no active-team name matched selector %q - SpielerPlus markup likely changed, inspect the page and update activeTeamNameSelector in team.go", activeTeamNameSelector)
	}
	return name, nil
}

// FetchActiveTeamName reports the name of the team currently active in this
// Client's SpielerPlus session - i.e. the team every other fetch in this
// package will read from, for an account that manages more than one.
func (c *Client) FetchActiveTeamName() (string, error) {
	body, err := c.get(dashboardPath)
	if err != nil {
		return "", err
	}
	defer body.Close()
	return ParseActiveTeamName(body)
}
