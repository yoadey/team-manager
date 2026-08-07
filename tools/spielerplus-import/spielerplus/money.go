package spielerplus

import (
	"fmt"
	"strconv"
	"strings"
)

// parseEuroCents parses a SpielerPlus amount string into integer cents,
// matching how Teamverwaltung stores money (BIGINT cents, not floating
// point - see backend/internal/db/migrations/00001_init.sql). Confirmed
// from a HAR capture of live cashbox/dues/punishment pages: always
// German-locale "X,XX €" (comma decimal separator), with a leading "-" for
// a negative/expense value. A "." thousands separator (e.g. "1.234,00 €")
// was not observed in the capture but is standard German number
// formatting, so it's stripped defensively before parsing the whole-euro
// part.
func parseEuroCents(s string) (cents int64, err error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSpace(strings.TrimSuffix(s, "€"))
	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 || len(parts[1]) != 2 {
		return 0, fmt.Errorf("expected an amount like \"X,XX\", got %q", s)
	}
	whole, err := strconv.ParseInt(strings.ReplaceAll(parts[0], ".", ""), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse euros %q: %w", parts[0], err)
	}
	fraction, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse cents %q: %w", parts[1], err)
	}

	cents = whole*100 + fraction
	if negative {
		cents = -cents
	}
	return cents, nil
}
