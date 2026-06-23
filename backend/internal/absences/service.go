package absences

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

// absenceRepo is the interface the Service relies on.
type absenceRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*AbsenceRow, error)
	ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit, offset int) ([]*AbsenceRow, error)
	Create(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*AbsenceRow, error)
	Update(ctx context.Context, id uuid.UUID, fromDate, toDate *string, reason *string) (*AbsenceRow, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service implements absence business logic.
type Service struct {
	repo absenceRepo
}

// NewService creates a new Service.
func NewService(repo absenceRepo) *Service {
	return &Service{repo: repo}
}

// ListByTeam returns paginated absences for the given team.
func (s *Service) ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]gen.Absence, error) {
	rows, err := s.repo.ListByTeam(ctx, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]gen.Absence, 0, len(rows))
	for _, row := range rows {
		result = append(result, toGenAbsence(row))
	}
	return result, nil
}

// ListByUser returns paginated absences for the authenticated user in the given team.
func (s *Service) ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit, offset int) ([]gen.Absence, error) {
	rows, err := s.repo.ListByUser(ctx, teamID, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]gen.Absence, 0, len(rows))
	for _, row := range rows {
		result = append(result, toGenAbsence(row))
	}
	return result, nil
}

// Create adds a new absence.
func (s *Service) Create(ctx context.Context, teamID uuid.UUID, body *gen.CreateAbsenceRequest) (gen.Absence, error) {
	row, err := s.repo.Create(ctx, teamID, uuid.UUID(body.UserId), body.From.Time.Format("2006-01-02"), body.To.Time.Format("2006-01-02"), body.Reason)
	if err != nil {
		return gen.Absence{}, err
	}
	return toGenAbsence(row), nil
}

// Update modifies an existing absence.
func (s *Service) Update(ctx context.Context, id uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error) {
	var from, to *string
	if body.From != nil {
		s := body.From.Time.Format("2006-01-02")
		from = &s
	}
	if body.To != nil {
		s := body.To.Time.Format("2006-01-02")
		to = &s
	}
	row, err := s.repo.Update(ctx, id, from, to, body.Reason)
	if err != nil {
		return gen.Absence{}, err
	}
	return toGenAbsence(row), nil
}

// Delete removes an absence by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// toGenAbsence maps an AbsenceRow to the generated gen.Absence type.
func toGenAbsence(row *AbsenceRow) gen.Absence {
	hasPhoto := len(row.PhotoData) > 0
	a := gen.Absence{
		Id:        openapi_types.UUID(row.Id),
		UserId:    openapi_types.UUID(row.UserId),
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
