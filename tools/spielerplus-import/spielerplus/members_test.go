package spielerplus

import (
	"fmt"
	"strings"
	"testing"
	"time"
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

// memberRowHTMLWithPhoto mirrors the confirmed markup from a HAR capture of
// a live /team page for a member with a photo set: a `.user-icon img` whose
// src is their photo on SpielerPlus's asset CDN.
func memberRowHTMLWithPhoto(id, name, role, photoSrc string) string {
	return fmt.Sprintf(`<div class="list-item team-list-item">
		<a class="list-item-link" href="/user/view?id=%s"></a>
		<div class="list-icon size-56 hidden-xs"><div class="member-icon-container">
			<a href="/user/view?id=%s"><div class="user-icon"><img src="%s" alt=""></div></a>
		</div></div>
		<div class="list-content-label-sublabel">
			<div class="list-label-section">
				<div class="list-label">%s</div>
				<div class="user-role"><div class="user-role-item">%s</div></div>
			</div>
		</div>
	</div>`, id, id, photoSrc, name, role)
}

func TestParseMembers_PhotoURL(t *testing.T) {
	html := memberRowHTMLWithPhoto("1", "Anna Trainer", "Trainer", "https://assets.spielerplus.de/images/user/200x200/abc123.jpg")
	members, err := ParseMembers(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMembers() error = %v", err)
	}
	if members[0].PhotoURL != "https://assets.spielerplus.de/images/user/200x200/abc123.jpg" {
		t.Errorf("PhotoURL = %q", members[0].PhotoURL)
	}
}

func TestParseMembers_DefaultPhotoIsNotAPhoto(t *testing.T) {
	// Confirmed from a HAR capture: SpielerPlus falls back to a generic
	// "default.svg" silhouette for a member with no custom photo, rather
	// than omitting the <img> - this must not be imported as if it were a
	// real photo.
	html := memberRowHTMLWithPhoto("1", "Anna Trainer", "Trainer", "//assets.spielerplus.de/assets/2216hash/images/user/200x200/default.svg")
	members, err := ParseMembers(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMembers() error = %v", err)
	}
	if members[0].PhotoURL != "" {
		t.Errorf("PhotoURL = %q, want empty for the default placeholder", members[0].PhotoURL)
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

func profileFieldHTML(label, value string) string {
	return fmt.Sprintf(`<div class="col-md-4 col-sm-6">
		<small class="light"><b>%s</b></small>
		<p class="dark">%s</p>
	</div>`, label, value)
}

func TestParseMemberBirthday(t *testing.T) {
	html := profileFieldHTML("Handynummer", "0151 12345678") +
		profileFieldHTML("Geburtstag", "03.07.1995")
	birthday, err := ParseMemberBirthday(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseMemberBirthday() error = %v", err)
	}
	want := time.Date(1995, time.July, 3, 0, 0, 0, 0, time.UTC)
	if !birthday.Equal(want) {
		t.Errorf("birthday = %v, want %v", birthday, want)
	}
}

func TestParseMemberBirthday_NotPresent(t *testing.T) {
	birthday, err := ParseMemberBirthday(strings.NewReader(`<html><body>no fields here</body></html>`))
	if err != nil {
		t.Fatalf("ParseMemberBirthday() error = %v", err)
	}
	if !birthday.IsZero() {
		t.Errorf("birthday = %v, want zero", birthday)
	}
}

func TestParseMemberBirthday_UnparseableNotFatal(t *testing.T) {
	html := profileFieldHTML("Geburtstag", "not a date")
	_, err := ParseMemberBirthday(strings.NewReader(html))
	if err == nil {
		t.Fatal("expected an error for an unparseable birthday value")
	}
}
