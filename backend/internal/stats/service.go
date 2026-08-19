package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

// statsRepo is the interface the Service relies on.
type statsRepo interface {
	MemberStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]MemberStatRow, error)
	EventStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]EventStatRow, error)
	SingleMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to string) (*MemberStatRow, error)
	WithReadTx(ctx context.Context, fn func(OverviewReader) error) error
	AttendanceMatrix(ctx context.Context, teamID uuid.UUID, from, to string) ([]MatrixColumnRow, []MatrixCellRow, error)
}

// Service implements stats business logic.
type Service struct {
	repo statsRepo
}

// NewService creates a new Service.
func NewService(repo statsRepo) *Service {
	return &Service{repo: repo}
}

// maxStatsRangeDays caps how far apart from/to may be. Generous for any
// club's history view, while preventing a caller-supplied range (e.g.
// from=0001-01-01) from forcing a full-table aggregation across every event
// and attendance row the team has ever had. Applies uniformly to every
// stats endpoint, including GetMemberStats (which, like the others, falls
// back to the last-3-months default via this function's from==nil,
// to==nil branch when the caller omits from/to).
const maxStatsRangeDays = 730

// defaultDateRange returns from = 3 months ago, to = today if not specified,
// clamping the effective range to at most maxStatsRangeDays wide.
//
// "today"/"3 months ago" are derived from time.Now(), i.e. the server
// process's local clock -- there is no per-team (or even per-deployment)
// timezone concept anywhere in this codebase to derive them from instead.
// The closest precedent, events.ZonedTimeToUTC (internal/events/zonedtime.go),
// hardcodes Europe/Berlin for interpreting event wall-clock times, but that's
// an app-wide constant standing in for "the club's timezone", not a per-team
// setting stats could look up -- and adopting it here would just trade one
// hardcoded assumption for another. In the shipped image (backend/Dockerfile,
// distroless/static-debian12) no TZ is set and no tzdata/zoneinfo is
// installed, so time.Now() resolves to UTC in production; docker-compose.yml
// and helm/team-manager/values.yaml likewise never set TZ. Operators wanting
// a different reference timezone must set TZ explicitly (and ship tzdata,
// e.g. via a blank `time/tzdata` import) for the server process.
//
// Net effect: for a club whose local calendar day doesn't align with the
// server's clock (UTC by default), the computed from/to boundary can be off
// by up to a day around midnight. Low impact in practice: from/to are
// date-only (no time-of-day component), and the default 3-month window is
// generous enough that a one-day shift at either edge rarely changes which
// events/attendance rows are included. This is a known, accepted trade-off,
// not a bug to fix here -- doing so properly would mean introducing a
// per-team timezone setting, which doesn't exist anywhere in this codebase
// and is out of scope for this change.
func defaultDateRange(from, to *openapi_types.Date) (fromStr, toStr string) {
	now := time.Now()
	toTime := now
	if to != nil {
		toTime = to.Time
	}
	fromTime := toTime.AddDate(0, -3, 0)
	if from != nil {
		fromTime = from.Time
	}
	if fromTime.After(toTime) {
		fromTime = toTime
	}
	if toTime.Sub(fromTime) > maxStatsRangeDays*24*time.Hour {
		fromTime = toTime.AddDate(0, 0, -maxStatsRangeDays)
	}
	return fromTime.Format("2006-01-02"), toTime.Format("2006-01-02")
}

// GetOverview builds the full StatsOverview for the given team and date range.
func (s *Service) GetOverview(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error) {
	fromStr, toStr := defaultDateRange(from, to)

	var (
		members []MemberStatRow
		events  []EventStatRow
	)
	// Run both reads inside one read-only transaction so Members[].Quote/Avg
	// (from MemberStats) and Events/PastCount (from EventStats) reflect the
	// same underlying event/attendance snapshot, instead of possibly
	// drifting if an event is created/cancelled or attendance is recorded
	// between the two queries -- mirrors finances.GetOverview's identical
	// WithReadTx guard.
	err := s.repo.WithReadTx(ctx, func(repo OverviewReader) error {
		var err error
		members, err = repo.MemberStats(ctx, teamID, fromStr, toStr)
		if err != nil {
			return fmt.Errorf("members: %w", err)
		}
		events, err = repo.EventStats(ctx, teamID, fromStr, toStr)
		if err != nil {
			return fmt.Errorf("events: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stats.Service.GetOverview: %w", err)
	}

	genMembers := make([]gen.MemberStat, 0, len(members))
	var totalQuote float32
	for _, m := range members {
		q := quote(m.Yes, m.Counted)
		totalQuote += q
		hp := m.HasPhoto
		genMembers = append(genMembers, gen.MemberStat{
			UserId:      m.UserID,
			Name:        m.Name,
			AvatarColor: m.AvatarColor,
			HasPhoto:    &hp,
			Yes:         m.Yes,
			Counted:     m.Counted,
			Quote:       q,
		})
	}

	var avg float32
	if len(members) > 0 {
		avg = totalQuote / float32(len(members))
	}

	genEvents := make([]gen.EventStat, 0, len(events))
	for _, e := range events {
		pct := quote(e.Yes, e.Counted)
		genEvents = append(genEvents, gen.EventStat{
			Id:        e.EventID,
			Title:     e.Title,
			Type:      gen.EventType(e.Type),
			Date:      parseDateOrZero(e.Date),
			Yes:       e.Yes,
			Nominated: e.Counted,
			Pct:       pct,
			Enough:    pct >= 0.5,
		})
	}

	fromDate := parseDateOrZero(fromStr)
	toDate := parseDateOrZero(toStr)

	return &gen.StatsOverview{
		From:      fromDate,
		To:        toDate,
		Members:   genMembers,
		Events:    genEvents,
		Avg:       avg,
		PastCount: len(events),
	}, nil
}

// GetMemberStats returns attendance statistics for a single team member.
func (s *Service) GetMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error) {
	fromStr, toStr := defaultDateRange(from, to)

	m, err := s.repo.SingleMemberStats(ctx, teamID, userID, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("stats.Service.GetMemberStats: %w", err)
	}

	return &gen.MemberAttendanceStats{
		Yes:     m.Yes,
		Counted: m.Counted,
		Quote:   quote(m.Yes, m.Counted),
	}, nil
}

// GetAttendanceMatrix builds the per-member-per-event attendance matrix for the
// team and date range. Cells use the same effective status as the quotes;
// rows are ordered by attendance (most yes first) then name, columns by date.
func (s *Service) GetAttendanceMatrix(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.AttendanceMatrix, error) {
	fromStr, toStr := defaultDateRange(from, to)

	cols, cells, err := s.repo.AttendanceMatrix(ctx, teamID, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("stats.Service.GetAttendanceMatrix: %w", err)
	}

	genCols := make([]gen.AttendanceMatrixColumn, 0, len(cols))
	for _, c := range cols {
		genCols = append(genCols, gen.AttendanceMatrixColumn{
			Id:    c.EventID,
			Title: c.Title,
			Type:  gen.EventType(c.Type),
			Date:  parseDateOrZero(c.Date),
		})
	}

	// Assemble rows keyed by member, preserving first-seen order (the cells
	// query is ORDER BY name) so equal-yes rows fall back to name order after
	// the stable sort below.
	type rowAcc struct {
		row   *gen.AttendanceMatrixRow
		cells map[string]gen.AttendanceStatus
	}
	byUser := make(map[uuid.UUID]*rowAcc)
	order := make([]uuid.UUID, 0, len(cells))
	for _, c := range cells {
		acc, ok := byUser[c.UserID]
		if !ok {
			hp := c.HasPhoto
			acc = &rowAcc{
				row: &gen.AttendanceMatrixRow{
					UserId:      c.UserID,
					Name:        c.Name,
					AvatarColor: c.AvatarColor,
					HasPhoto:    &hp,
				},
				cells: make(map[string]gen.AttendanceStatus),
			}
			byUser[c.UserID] = acc
			order = append(order, c.UserID)
		}
		if c.EventID == nil {
			continue // placeholder row for a member with no events in range
		}
		acc.cells[c.EventID.String()] = gen.AttendanceStatus(c.Eff)
		switch c.Eff {
		case "yes":
			acc.row.Yes++
			acc.row.Counted++
		case "no", "maybe":
			acc.row.Counted++
		}
	}

	genRows := make([]gen.AttendanceMatrixRow, 0, len(order))
	for _, id := range order {
		acc := byUser[id]
		acc.row.Cells = acc.cells
		genRows = append(genRows, *acc.row)
	}
	// Stable sort keeps the name order from the query as the tiebreaker.
	sort.SliceStable(genRows, func(i, j int) bool {
		return genRows[i].Yes > genRows[j].Yes
	})

	return &gen.AttendanceMatrix{
		From:    parseDateOrZero(fromStr),
		To:      parseDateOrZero(toStr),
		Events:  genCols,
		Members: genRows,
	}, nil
}

func quote(yes, counted int) float32 {
	if counted == 0 {
		return 0
	}
	return float32(yes) / float32(counted)
}

func parseDateOrZero(s string) openapi_types.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return openapi_types.Date{}
	}
	return openapi_types.Date{Time: t}
}
