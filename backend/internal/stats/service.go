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
	ListGroupThresholds(ctx context.Context, teamID uuid.UUID) ([]GroupThresholdRow, error)
	SetGroupThresholds(ctx context.Context, teamID uuid.UUID, items []GroupThresholdRow) ([]GroupThresholdRow, error)
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
		groups  []EventGroupStatRow
	)
	// Run all reads inside one read-only transaction so Members[].Quote/Avg
	// (from MemberStats), Events/PastCount (from EventStats) and each
	// event's Groups (from EventGroupStats) reflect the same underlying
	// event/attendance/role snapshot, instead of possibly
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
		groups, err = repo.EventGroupStats(ctx, teamID, fromStr, toStr)
		if err != nil {
			return fmt.Errorf("event groups: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stats.Service.GetOverview: %w", err)
	}

	genMembers := make([]gen.MemberStat, 0, len(members))
	var totalQuote float32
	var membersWithData int
	for _, m := range members {
		q := quote(m.Yes, m.Counted)
		// Members with no counted events in the range (Counted == 0) have no
		// attendance data, not a 0% attendance record -- exclude them from
		// both the numerator and denominator of the average so they aren't
		// scored as a hard 0%. Mirrors the per-member breakdown, where the
		// frontend maps this same Counted == 0 state to "no data" (quote:
		// null, rendered "-") instead of 0% (see mapMemberStat in
		// frontend/src/api/map.ts).
		if m.Counted > 0 {
			totalQuote += q
			membersWithData++
		}
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

	// avg is the mean attendance quote across members who have at least one
	// counted event in the range. Members with Counted == 0 (no data --
	// e.g. a brand-new member, or someone whose only events all fell under
	// not_relevant_for_stats/exclude_from_stats) are excluded entirely
	// rather than averaged in as 0%; if no member in the range has any
	// counted events, avg is 0 (nothing to average, not a real 0%
	// attendance rate).
	var avg float32
	if membersWithData > 0 {
		avg = totalQuote / float32(membersWithData)
	}

	// Bucket the group rows by event, preserving the query's own order
	// (system roles first, then by name) within each event.
	groupsByEvent := make(map[uuid.UUID][]gen.EventGroupStat)
	for _, g := range groups {
		groupsByEvent[g.EventID] = append(groupsByEvent[g.EventID], gen.EventGroupStat{
			RoleId:         g.RoleID,
			Name:           g.Name,
			Color:          g.Color,
			Yes:            g.Yes,
			Counted:        g.Counted,
			Pending:        g.Pending,
			ThresholdMode:  gen.StatsThresholdMode(g.ThresholdMode),
			ThresholdValue: g.ThresholdValue,
		})
	}

	genEvents := make([]gen.EventStat, 0, len(events))
	for _, e := range events {
		pct := quote(e.Yes, e.Counted)
		// Always a non-nil slice: `groups` is a required array in the
		// schema, so an event whose team has no role-holding member must
		// serialize as [] rather than null.
		evGroups := groupsByEvent[e.EventID]
		if evGroups == nil {
			evGroups = []gen.EventGroupStat{}
		}
		genEvents = append(genEvents, gen.EventStat{
			Id:        e.EventID,
			Title:     e.Title,
			Type:      gen.EventType(e.Type),
			Date:      parseDateOrZero(e.Date),
			Yes:       e.Yes,
			Nominated: e.Counted,
			Pending:   e.Pending,
			Pct:       pct,
			Enough:    pct >= 0.5,
			Groups:    evGroups,
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

// ListGroupThresholds returns the attendance threshold of every role of the
// team, with the default applied for roles that were never configured.
func (s *Service) ListGroupThresholds(ctx context.Context, teamID uuid.UUID) ([]GroupThresholdRow, error) {
	rows, err := s.repo.ListGroupThresholds(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("stats.Service.ListGroupThresholds: %w", err)
	}
	return rows, nil
}

// SetGroupThresholds validates and stores the given roles' thresholds and
// returns the team's full threshold list afterwards. Every item is validated
// before anything is written, so one bad item rejects the whole request.
func (s *Service) SetGroupThresholds(ctx context.Context, teamID uuid.UUID, items []GroupThresholdRow) ([]GroupThresholdRow, error) {
	for _, it := range items {
		if err := validateThreshold(it.Mode, it.Value); err != nil {
			return nil, err
		}
	}
	rows, err := s.repo.SetGroupThresholds(ctx, teamID, items)
	if err != nil {
		return nil, fmt.Errorf("stats.Service.SetGroupThresholds: %w", err)
	}
	return rows, nil
}

// validateThreshold enforces the mode enum and the per-mode value range.
// There is no request-validation middleware in this codebase, so the
// OpenAPI-declared enum/minimum/maximum must be re-enforced here to take
// effect (role_stats_thresholds' CHECKs are only the backstop).
func validateThreshold(mode string, value int) error {
	var maxValue int
	switch mode {
	case ThresholdModePercent:
		maxValue = maxThresholdPercent
	case ThresholdModeCount:
		maxValue = maxThresholdCount
	default:
		return fmt.Errorf("%w: mode must be %q or %q", ErrInvalidThreshold, ThresholdModePercent, ThresholdModeCount)
	}
	if value < 0 || value > maxValue {
		return fmt.Errorf("%w: value must be between 0 and %d for mode %q", ErrInvalidThreshold, maxValue, mode)
	}
	return nil
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
