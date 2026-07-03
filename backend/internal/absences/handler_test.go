package absences_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

type mockAbsenceService struct {
	update func(ctx context.Context, id, teamID, userID uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error)
}

func (m *mockAbsenceService) ListByTeam(context.Context, uuid.UUID, int, string) ([]gen.Absence, *string, error) {
	panic("not implemented")
}

func (m *mockAbsenceService) ListByUser(context.Context, uuid.UUID, uuid.UUID, int, string) ([]gen.Absence, *string, error) {
	panic("not implemented")
}

func (m *mockAbsenceService) Create(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
	panic("not implemented")
}

func (m *mockAbsenceService) Update(ctx context.Context, id, teamID, userID uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error) {
	return m.update(ctx, id, teamID, userID, body)
}

func (m *mockAbsenceService) Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	panic("not implemented")
}

// A partial update (only "from" supplied) that would push the range past the
// existing "to" date can only be caught by the DB's CHECK constraint, since
// the merge happens inside the UPDATE statement. This must surface as 400,
// not the generic 500 that a bare repository error would produce.
func TestAbsenceHandler_UpdateAbsence_InvalidDateRange_Returns400(t *testing.T) {
	t.Parallel()

	svc := &mockAbsenceService{
		update: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *gen.UpdateAbsenceRequest) (gen.Absence, error) {
			return gen.Absence{}, absences.ErrInvalidDateRange
		},
	}
	h := absences.NewHandler(svc, slog.Default())

	userID := uuid.New()
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	from := openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
	body := &gen.UpdateAbsenceRequest{From: &from}
	_, err := h.UpdateAbsence(ctx, gen.UpdateAbsenceRequestObject{
		TeamId: uuid.New(), AbsenceId: uuid.New(), Body: body,
	})

	require.Error(t, err)
	require.NotContains(t, err.Error(), "failed to update absence", "must map to the specific 400, not fall through to the generic 500")
}
