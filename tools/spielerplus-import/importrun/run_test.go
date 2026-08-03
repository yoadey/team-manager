package importrun

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDateRange_Overlaps(t *testing.T) {
	base := dateRange{from: day("2026-06-01"), to: day("2026-06-10")}

	cases := []struct {
		name string
		r    dateRange
		want bool
	}{
		{"identical", dateRange{from: day("2026-06-01"), to: day("2026-06-10")}, true},
		{"overlapping start", dateRange{from: day("2026-05-25"), to: day("2026-06-02")}, true},
		{"overlapping end", dateRange{from: day("2026-06-09"), to: day("2026-06-20")}, true},
		{"contained", dateRange{from: day("2026-06-03"), to: day("2026-06-05")}, true},
		{"touching boundary", dateRange{from: day("2026-06-10"), to: day("2026-06-15")}, true},
		{"adjacent, no overlap", dateRange{from: day("2026-06-11"), to: day("2026-06-15")}, false},
		{"well before", dateRange{from: day("2026-01-01"), to: day("2026-01-31")}, false},
		{"well after", dateRange{from: day("2026-12-01"), to: day("2026-12-31")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := base.overlaps(tc.r); got != tc.want {
				t.Errorf("base.overlaps(%+v) = %v, want %v", tc.r, got, tc.want)
			}
			if got := tc.r.overlaps(base); got != tc.want {
				t.Errorf("overlaps should be symmetric: %+v.overlaps(base) = %v, want %v", tc.r, got, tc.want)
			}
		})
	}
}

func TestOverlapsAny(t *testing.T) {
	ranges := []dateRange{
		{from: day("2026-06-01"), to: day("2026-06-10")},
		{from: day("2026-08-01"), to: day("2026-08-10")},
	}
	if !overlapsAny(ranges, dateRange{from: day("2026-08-05"), to: day("2026-08-06")}) {
		t.Error("expected overlap with second range")
	}
	if overlapsAny(ranges, dateRange{from: day("2026-07-01"), to: day("2026-07-10")}) {
		t.Error("expected no overlap with either range")
	}
	if overlapsAny(nil, dateRange{from: day("2026-01-01"), to: day("2026-01-02")}) {
		t.Error("empty range list should never overlap")
	}
}
