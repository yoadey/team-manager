package spielerplus

import (
	"strings"
	"testing"
)

// duesPageHTML mirrors the confirmed markup from a HAR capture of a live
// /cashbox/dues page: a header row with labeled+amount columns, then one
// <tr> per member with a toggle button per column.
func duesPageHTML() string {
	return `<div class="cashbox-table"><table class="table">
		<thead><tr>
			<th class="first-column"></th>
			<th class="text-center"><span title="Teamkasse1">Teamkasse120,00 €</span></th>
			<th class="text-center"><span title="Fahrtgeld1">Fahrtgeld15,00 €</span></th>
		</tr></thead>
		<tr><td class="first-column"><a href="/user/view?id=1"></a></td>
			<td class="text-center"><button onClick="toggleCashbox(this, 1, 100)"><i class="fa-circle-check sp fa-fw fa-lg"></i></button></td>
			<td class="text-center"><button onClick="toggleCashbox(this, 1, 101)"><i class="fa-circle sp fa-fw fa-lg"></i></button></td>
		</tr>
		<tr><td class="first-column"><a href="/user/view?id=2"></a></td>
			<td class="text-center"><button onClick="toggleCashbox(this, 2, 100)"><i class="fa-circle sp fa-fw fa-lg"></i></button></td>
			<td class="text-center"><button onClick="toggleCashbox(this, 2, 101)"><i class="fa-circle-check sp fa-fw fa-lg"></i></button></td>
		</tr>
	</table></div>`
}

func TestParseDues(t *testing.T) {
	dues, err := ParseDues(strings.NewReader(duesPageHTML()))
	if err != nil {
		t.Fatalf("ParseDues() error = %v", err)
	}
	if len(dues) != 4 {
		t.Fatalf("got %d dues, want 4", len(dues))
	}

	byID := map[string]Due{}
	for _, d := range dues {
		byID[d.ID] = d
	}

	teamkasse1Member1, ok := byID["100:1"]
	if !ok {
		t.Fatalf("dues = %+v, want an entry for column 100, member 1", dues)
	}
	if teamkasse1Member1.Label != "Teamkasse1" || teamkasse1Member1.AmountCents != 2000 || !teamkasse1Member1.Paid {
		t.Errorf("teamkasse1Member1 = %+v", teamkasse1Member1)
	}

	fahrtgeld1Member1, ok := byID["101:1"]
	if !ok || fahrtgeld1Member1.Paid {
		t.Errorf("fahrtgeld1Member1 = %+v, want unpaid", fahrtgeld1Member1)
	}
	if fahrtgeld1Member1.AmountCents != 500 {
		t.Errorf("fahrtgeld1Member1 = %+v", fahrtgeld1Member1)
	}
}

func TestParseDues_NoColumns(t *testing.T) {
	html := `<table class="table"><thead><tr><th class="first-column"></th></tr></thead></table>`
	if _, err := ParseDues(strings.NewReader(html)); err == nil {
		t.Fatal("expected an error when no due columns are present")
	}
}

func TestParseDues_NoRows(t *testing.T) {
	html := `<table class="table"><thead><tr>
		<th class="first-column"></th>
		<th class="text-center"><span title="Teamkasse1">Teamkasse120,00 €</span></th>
	</tr></thead></table>`
	if _, err := ParseDues(strings.NewReader(html)); err == nil {
		t.Fatal("expected an error when no member rows are present")
	}
}
