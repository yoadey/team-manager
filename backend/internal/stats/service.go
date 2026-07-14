package stats

import (
	"context"
	"fmt"
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
// and attendance row the team has ever had, unlike GetMemberStats which
// always uses the fixed 3-month default (its request has no from/to params
// at all, so this function's from==nil, to==nil branch is its only path).
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
