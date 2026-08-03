package spielerplus

import (
	"fmt"
	"strings"
	"testing"
)

// memberRowHTML mirrors the confirmed markup from a HAR capture of a live
// /team page: a profile link carrying the user id, name, and role.
func memberRowHTML(id, name, role string) string {
	return fmt.Sprintf(`<div class="list-item team-list-item">
		<a class="list-item-link" href="/user/view?id=%s"></a>
		<div class="list-content-label-sublabel">
			<div class="list-label-section">
				<div class="list-label">%s</div>
				<div class="user-role"><div class="user-role-item">%s</div></div>
			</div>
		</div>
	</div>`, id, name, role)
}

func TestParseMembers(t *testing.T) {
	html := `<div id="pjax-members">` +
		memberRowHTML("1", "Anna Trainer", "Trainer") +
		memberRowHTML("2", "Ben Spieler", "Spieler") +
		`</div>`

	members, err := ParseMembers(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMembers() error = %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if members[0] != (Member{ID: "1", Name: "Anna Trainer", Role: "Trainer"}) {
		t.Errorf("members[0] = %+v", members[0])
	}
	if members[1].Role != "Spieler" {
		t.Errorf("members[1].Role = %q, want Spieler", members[1].Role)
	}
}

func TestParseMembers_BadRowSkippedNotFatal(t *testing.T) {
	html := `<div id="pjax-members">` +
		memberRowHTML("1", "Anna Trainer", "Trainer") +
		`<div class="list-item team-list-item"></div>` + // no profile link: should be skipped, not fatal
		`</div>`

	members, err := ParseMembers(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMembers() error = %v, want a good row alongside a bad one to succeed", err)
	}
	if len(members) != 1 || members[0].ID != "1" {
		t.Fatalf("members = %+v, want only member 1 (the second row skipped)", members)
	}
}

func TestParseMembers_NoRowsMatched(t *testing.T) {
	_, err := ParseMembers(strings.NewReader(`<html><body>empty</body></html>`))
	if err == nil {
		t.Fatal("expected an error when no member rows match the selector")
	}
}

func TestParseMemberEmail(t *testing.T) {
	html := `<div class="col-md-4 col-sm-6">
		<small class="light"><b>E-Mail Adresse</b></small>
		<p class="dark"><a href="mailto:anna@example.com">anna@example.com</a></p>
	</div>`
	email, err := ParseMemberEmail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMemberEmail() error = %v", err)
	}
	if email != "anna@example.com" {
		t.Errorf("email = %q, want anna@example.com", email)
	}
}

func TestParseMemberEmail_NoneVisible(t *testing.T) {
	email, err := ParseMemberEmail(strings.NewReader(`<html><body>no email here</body></html>`))
	if err != nil {
		t.Fatalf("ParseMemberEmail() error = %v", err)
	}
	if email != "" {
		t.Errorf("email = %q, want empty", email)
	}
}
