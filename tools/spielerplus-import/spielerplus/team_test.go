package spielerplus

import (
	"strings"
	"testing"
)

func TestParseActiveTeamName(t *testing.T) {
	html := `<div class="navigation__card">
		<div class="navigation__card__content">
			<div class="navigation__card__title">TSC B-Team 25/26</div>
		</div>
		<a href="/site/select-team" class="navigation__card__button">Team wechseln</a>
	</div>`

	name, err := ParseActiveTeamName(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseActiveTeamName() error = %v", err)
	}
	if name != "TSC B-Team 25/26" {
		t.Errorf("name = %q, want %q", name, "TSC B-Team 25/26")
	}
}

func TestParseActiveTeamName_NotFound(t *testing.T) {
	_, err := ParseActiveTeamName(strings.NewReader(`<html><body>no nav card here</body></html>`))
	if err == nil {
		t.Fatal("expected an error when no active-team name is found")
	}
}
