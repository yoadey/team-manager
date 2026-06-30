package absences

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// absenceRepo is the interface the Service relies on.
type absenceRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*AbsenceRow, error)
	ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit int, cur *ListCursor) ([]*AbsenceRow, error)
	Create(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*AbsenceRow, error)
	Update(ctx context.Context, id uuid.UUID, fromDate, toDate *string, reason *string) (*AbsenceRow, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service implements absence business logic.
type Service struct {
	repo  absenceRepo
	pager *pagination.Paginator
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo absenceRepo, pager *pagination.Paginator) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, pager: pager}
}

// ListByTeam returns a keyset page of team absences plus the next-page cursor
// (nil on the last page). cursor is the opaque token from a prior page.
func (s *Service) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.Absence, *string, error) {
	cur, err := s.decodeAbsenceCursor(cursor)
	if err != nil {
		return nil, nil, fmt.Errorf("absences.Service.ListByTeam: %w", err)
	}
	rows, err := s.repo.ListByTeam(ctx, teamID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("absences.Service.ListByTeam: %w", err)
	}
	return s.absencePage(rows, limit)
}

// ListByUser returns a keyset page of the user's absences in the team plus the
// next-page cursor (nil on the last page).
func (s *Service) ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit int, cursor string) ([]gen.Absence, *string, error) {
	cur, err := s.decodeAbsenceCursor(cursor)
	if err != nil {
		return nil, nil, fmt.Errorf("absences.Service.ListByUser: %w", err)
	}
	rows, err := s.repo.ListByUser(ctx, teamID, userID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("absences.Service.ListByUser: %w", err)
	}
	return s.absencePage(rows, limit)
}

// decodeAbsenceCursor parses the opaque cursor token ("" = first page).
func (s *Service) decodeAbsenceCursor(cursor string) (*ListCursor, error) {
	var decoded ListCursor
	ok, err := s.pager.Decode(cursor, &decoded)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	if !ok {
		return nil, nil
	}
	return &decoded, nil
}

// absencePage trims the limit+1 fetch to a page and computes the next cursor.
func (s *Service) absencePage(rows []*AbsenceRow, limit int) ([]gen.Absence, *string, error) {
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		token, err := s.pager.Encode(ListCursor{FromDate: last.FromDate, ID: last.Id})
		if err != nil {
			return nil, nil, fmt.Errorf("encode cursor: %w", err)
		}
		next = &token
	}
	result := make([]gen.Absence, 0, len(rows))
	for _, row := range rows {
		result = append(result, toGenAbsence(row))
	}
	return result, next, nil
}

// Create adds a new absence.
func (s *Service) Create(ctx context.Context, teamID uuid.UUID, body *gen.CreateAbsenceRequest) (gen.Absence, error) {
	row, err := s.repo.Create(ctx, teamID, body.UserId, body.From.Format("2006-01-02"), body.To.Format("2006-01-02"), body.Reason)
	if err != nil {
		return gen.Absence{}, fmt.Errorf("absences.Service.Create: %w", err)
	}
	return toGenAbsence(row), nil
}

// Update modifies an existing absence.
func (s *Service) Update(ctx context.Context, id uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error) {
	var from, to *string
	if body.From != nil {
		s := body.From.Format("2006-01-02")
		from = &s
	}
	if body.To != nil {
		s := body.To.Format("2006-01-02")
		to = &s
	}
	row, err := s.repo.Update(ctx, id, from, to, body.Reason)
	if err != nil {
		return gen.Absence{}, fmt.Errorf("absences.Service.Update: %w", err)
	}
	return toGenAbsence(row), nil
}

// Delete removes an absence by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("absences.Service.Delete: %w", err)
	}
	return nil
}

// toGenAbsence maps an AbsenceRow to the generated gen.Absence type.
func toGenAbsence(row *AbsenceRow) gen.Absence {
	hasPhoto := len(row.PhotoData) > 0
	a := gen.Absence{
		Id:        row.Id,
		UserId:    row.UserId,
		From:      openapi_types.Date{Time: row.FromDate},
		To:        openapi_types.Date{Time: row.ToDate},
		Reason:    row.Reason,
		CreatedAt: row.CreatedAt,
		HasPhoto:  &hasPhoto,
	}
	if row.MemberName != nil {
		a.MemberName = row.MemberName
	}
	if row.MemberAvatarColor != nil {
		a.MemberAvatarColor = row.MemberAvatarColor
	}
	if row.RoleName != nil {
		a.RoleName = row.RoleName
	}
	if row.RoleColor != nil {
		a.RoleColor = row.RoleColor
	}
	return a
}
