package absences

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// absenceRepo is the interface the Service relies on.
type absenceRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*AbsenceRow, error)
	ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit int, cur *ListCursor) ([]*AbsenceRow, error)
	Create(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*AbsenceRow, error)
	Update(ctx context.Context, id, teamID, userID uuid.UUID, fromDate, toDate *string, reason *string) (*AbsenceRow, error)
	Delete(ctx context.Context, id, teamID, userID uuid.UUID) error
	GetOwner(ctx context.Context, id, teamID uuid.UUID) (uuid.UUID, error)
	SetStatsRelevance(ctx context.Context, id, teamID uuid.UUID, notRelevant bool, setBy uuid.UUID) (*AbsenceRow, error)
}

// permChecker returns the effective per-module permissions for a (team,
// user) pair -- satisfied by members.Repository. Mirrors
// notifications.Service's identical dependency.
type permChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// ErrForbiddenStatsRelevance is returned by SetStatsRelevance when the
// caller is neither the absence's owner nor an events:write holder.
var ErrForbiddenStatsRelevance = errors.New("only the absence's owner or an events:write holder may set its stats relevance")

// Service implements absence business logic.
type Service struct {
	repo  absenceRepo
	pager *pagination.Paginator
	perms permChecker
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo absenceRepo, pager *pagination.Paginator, perms permChecker) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, pager: pager, perms: perms}
}

// ListByTeam returns a keyset page of team absences plus the next-page cursor
// (nil on the last page). cursor is the opaque token from a prior page.
//
// Known, accepted asymmetry: absences maps to no RBAC module at all
// (middleware/authz.go's routeModule["absences"] == ""), so any team member
// -- even one with every module permission set to "none" -- can read every
// other member's free-text absence reason via this endpoint, with no
// redaction or per-entry visibility choice comparable to events'
// reasonVisibility ("team"/"trainers", events.Service.ListAttendance). That
// design was deliberate (a team-wide "who's out and why" view was judged
// more useful open than gated), but it means an absence reason -- unlike a
// declined-attendance reason -- gets zero privacy protection even from a
// member the team has otherwise fully locked out. Revisiting this would mean
// adding a reasonVisibility-equivalent system to absences; left as a known
// gap rather than implemented speculatively here.
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

// Update modifies an existing absence that belongs to teamID and userID
// (self-service: a member may only update their own absence entries).
func (s *Service) Update(ctx context.Context, id, teamID, userID uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error) {
	var from, to *string
	if body.From != nil {
		s := body.From.Format("2006-01-02")
		from = &s
	}
	if body.To != nil {
		s := body.To.Format("2006-01-02")
		to = &s
	}
	row, err := s.repo.Update(ctx, id, teamID, userID, from, to, body.Reason)
	if err != nil {
		return gen.Absence{}, fmt.Errorf("absences.Service.Update: %w", err)
	}
	return toGenAbsence(row), nil
}

// Delete removes an absence by ID that belongs to teamID and userID
// (self-service: a member may only delete their own absence entries).
func (s *Service) Delete(ctx context.Context, id, teamID, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, teamID, userID); err != nil {
		return fmt.Errorf("absences.Service.Delete: %w", err)
	}
	return nil
}

// SetStatsRelevance sets notRelevantForStats on the absence identified by
// (id, teamID). callerID may always set this on their own absence,
// unconditionally (parity with the rest of absences' self-service model).
// Setting it on another member's absence additionally requires callerID to
// hold events:write on teamID -- absences carries no module-level RBAC gate
// of its own (x-rbac-module: public), so this check is enforced here rather
// than by the generated route table (see design.md's "Route stays
// x-rbac-module: public" decision).
func (s *Service) SetStatsRelevance(ctx context.Context, id, teamID, callerID uuid.UUID, notRelevant bool) (gen.Absence, error) {
	ownerID, err := s.repo.GetOwner(ctx, id, teamID)
	if err != nil {
		return gen.Absence{}, fmt.Errorf("absences.Service.SetStatsRelevance: %w", err)
	}
	if ownerID != callerID {
		perms, err := s.perms.GetPermissions(ctx, teamID, callerID)
		if err != nil {
			return gen.Absence{}, fmt.Errorf("absences.Service.SetStatsRelevance: %w", err)
		}
		if !hasEventsWritePermission(perms) {
			return gen.Absence{}, ErrForbiddenStatsRelevance
		}
	}
	row, err := s.repo.SetStatsRelevance(ctx, id, teamID, notRelevant, callerID)
	if err != nil {
		return gen.Absence{}, fmt.Errorf("absences.Service.SetStatsRelevance: %w", err)
	}
	return toGenAbsence(row), nil
}

// hasEventsWritePermission reports whether p grants "write" on the events
// module. Deliberately narrow (only the one module this package ever
// checks) rather than a general multi-module helper -- mirrors
// notifications.HasReadAccess's reasoning for locally re-implementing this
// same fail-closed switch instead of importing internal/middleware's
// unexported hasWritePermission.
func hasEventsWritePermission(p teams.PermissionsJSON) bool {
	return p.Events == "write"
}

// toGenAbsence maps an AbsenceRow to the generated gen.Absence type.
func toGenAbsence(row *AbsenceRow) gen.Absence {
	hasPhoto := row.HasPhoto
	a := gen.Absence{
		Id:                  row.Id,
		UserId:              row.UserId,
		From:                openapi_types.Date{Time: row.FromDate},
		To:                  openapi_types.Date{Time: row.ToDate},
		Reason:              row.Reason,
		CreatedAt:           row.CreatedAt,
		HasPhoto:            &hasPhoto,
		NotRelevantForStats: row.NotRelevantForStats,
	}
	if row.MemberName != nil {
		a.MemberName = row.MemberName
	}
	if row.MemberAvatarColor != nil {
		a.MemberAvatarColor = row.MemberAvatarColor
	}
	if row.MembershipId != nil {
		a.MemberMembershipId = row.MembershipId
	}
	if row.RoleName != nil {
		a.RoleName = row.RoleName
	}
	if row.RoleColor != nil {
		a.RoleColor = row.RoleColor
	}
	return a
}
