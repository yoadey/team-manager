package spielerplus

import (
	"strings"
	"testing"
)

func TestParseMembers(t *testing.T) {
	html := `
	<html><body>
	<div data-user-id="1">
		<span class="name">Anna Trainer</span>
		<span class="email">anna@example.com</span>
		<span class="role">Trainer</span>
	</div>
	<div data-user-id="2">
		<span class="name">Ben Spieler</span>
		<span class="email">ben@example.com</span>
		<span class="role">Spieler</span>
	</div>
	</body></html>`

	members, err := ParseMembers(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMembers() error = %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if members[0] != (Member{ID: "1", Name: "Anna Trainer", Email: "anna@example.com", Role: "Trainer"}) {
		t.Errorf("members[0] = %+v", members[0])
	}
	if members[1].Role != "Spieler" {
		t.Errorf("members[1].Role = %q, want Spieler", members[1].Role)
	}
}

func TestParseMembers_MissingEmail(t *testing.T) {
	html := `<div data-user-id="1"><span class="name">Anna</span></div>`
	_, err := ParseMembers(strings.NewReader(html))
	if err == nil {
		t.Fatal("expected an error for a member row missing an email")
	}
}

func TestParseMembers_BadRowSkippedNotFatal(t *testing.T) {
	html := `
	<div data-user-id="1"><span class="name">Anna Trainer</span><span class="email">anna@example.com</span></div>
	<div data-user-id="2"><span class="name">No Email</span></div>`

	members, err := ParseMembers(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMembers() error = %v, want a good row alongside a bad one to succeed", err)
	}
	if len(members) != 1 || members[0].ID != "1" {
		t.Fatalf("members = %+v, want only member 1 (member 2's bad row skipped)", members)
	}
}

func TestParseMembers_NoRowsMatched(t *testing.T) {
	_, err := ParseMembers(strings.NewReader(`<html><body>empty</body></html>`))
	if err == nil {
		t.Fatal("expected an error when no member rows match the selector")
	}
}
