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
}

// Service implements stats business logic.
type Service struct {
	repo statsRepo
}

// NewService creates a new Service.
func NewService(repo statsRepo) *Service {
	return &Service{repo: repo}
}

// defaultDateRange returns from = 90 days ago, to = today if not specified.
func defaultDateRange(from, to *openapi_types.Date) (string, string) {
	now := time.Now()
	toStr := now.Format("2006-01-02")
	fromStr := now.AddDate(0, -3, 0).Format("2006-01-02")
	if from != nil {
		fromStr = from.Time.Format("2006-01-02")
	}
	if to != nil {
		toStr = to.Time.Format("2006-01-02")
	}
	return fromStr, toStr
}

// GetOverview builds the full StatsOverview for the given team and date range.
func (s *Service) GetOverview(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error) {
	fromStr, toStr := defaultDateRange(from, to)

	members, err := s.repo.MemberStats(ctx, teamID, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("stats.Service.GetOverview members: %w", err)
	}

	events, err := s.repo.EventStats(ctx, teamID, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("stats.Service.GetOverview events: %w", err)
	}

	genMembers := make([]gen.MemberStat, 0, len(members))
	var totalQuote float32
	for _, m := range members {
		q := quote(m.Yes, m.Counted)
		totalQuote += q
		hp := m.HasPhoto
		genMembers = append(genMembers, gen.MemberStat{
			UserId:      openapi_types.UUID(m.UserID),
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
			Id:        openapi_types.UUID(e.EventID),
			Title:     e.Title,
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
