package spielerplus

import "testing"

func TestParseEuroCents(t *testing.T) {
	cases := map[string]int64{
		"20,00 €":    2000,
		"-6,00 €":    -600,
		"1,00 €":     100,
		"0,50 €":     50,
		"1.234,56 €": 123456,
		"  3,00 €  ": 300,
	}
	for in, want := range cases {
		got, err := parseEuroCents(in)
		if err != nil {
			t.Errorf("parseEuroCents(%q) error = %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseEuroCents(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseEuroCents_Invalid(t *testing.T) {
	for _, in := range []string{"", "abc", "20.00 €", "20,0 €", "20 €"} {
		if _, err := parseEuroCents(in); err == nil {
			t.Errorf("parseEuroCents(%q) expected an error, got none", in)
		}
	}
}
