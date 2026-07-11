package absences_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

type mockAbsenceService struct {
	create func(ctx context.Context, teamID uuid.UUID, body *gen.CreateAbsenceRequest) (gen.Absence, error)
	update func(ctx context.Context, id, teamID, userID uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error)
}

func (m *mockAbsenceService) ListByTeam(context.Context, uuid.UUID, int, string) ([]gen.Absence, *string, error) {
	panic("not implemented")
}

func (m *mockAbsenceService) ListByUser(context.Context, uuid.UUID, uuid.UUID, int, string) ([]gen.Absence, *string, error) {
	panic("not implemented")
}

func (m *mockAbsenceService) Create(ctx context.Context, teamID uuid.UUID, body *gen.CreateAbsenceRequest) (gen.Absence, error) {
	return m.create(ctx, teamID, body)
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

// CreateAbsenceRequest.From/.To are non-pointer Date fields (required per
// openapi.yaml), but nothing in this stack enforces "required" at decode
// time — an omitted field just leaves Go's zero time.Time{}. These
// regression tests cover the two ways that used to slip through: an omitted
// "to" skipped the ordering check entirely, and an omitted "from" passed it
// vacuously (zero time is never After anything), silently persisting an
// absence spanning from year 1.
func TestAbsenceHandler_CreateAbsence_MissingTo_Returns400(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		create: func(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
			t.Fatal("service must not be called when 'to' is missing")
			return gen.Absence{}, nil
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		From:   openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
	}
	_, err := h.CreateAbsence(ctx, gen.CreateAbsenceRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestAbsenceHandler_CreateAbsence_MissingFrom_Returns400(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		create: func(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
			t.Fatal("service must not be called when 'from' is missing")
			return gen.Absence{}, nil
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		To:     openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
	}
	_, err := h.CreateAbsence(ctx, gen.CreateAbsenceRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestAbsenceHandler_CreateAbsence_InvalidDateRange_Returns400(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		create: func(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
			return gen.Absence{}, absences.ErrInvalidDateRange
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		From:   openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		To:     openapi_types.Date{Time: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)},
	}
	_, err := h.CreateAbsence(ctx, gen.CreateAbsenceRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
	require.NotContains(t, err.Error(), "failed to create absence", "must map to the specific 400, not fall through to the generic 500")
}

// Regression test: from/to being correctly ordered wasn't enough -- a
// multi-decade span (e.g. a typo'd year) was accepted with no bound at all.
func TestAbsenceHandler_CreateAbsence_RejectsExcessiveSpan(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		create: func(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
			t.Fatal("service must not be called when the span exceeds the cap")
			return gen.Absence{}, nil
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		From:   openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		To:     openapi_types.Date{Time: time.Date(2226, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	_, err := h.CreateAbsence(ctx, gen.CreateAbsenceRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

// Regression test: maxAbsenceSpanDays was only enforced on Create -- when
// UpdateAbsence received both from and to in the same PATCH, the span could
// be computed with no extra query (exactly like Create does), but the check
// was simply missing, so a self-service PATCH could stretch any existing
// absence out to a multi-century span with no rejection.
func TestAbsenceHandler_UpdateAbsence_RejectsExcessiveSpan(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		update: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *gen.UpdateAbsenceRequest) (gen.Absence, error) {
			t.Fatal("service must not be called when the span exceeds the cap")
			return gen.Absence{}, nil
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	from := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	to := openapi_types.Date{Time: time.Date(2226, 1, 1, 0, 0, 0, 0, time.UTC)}
	body := &gen.UpdateAbsenceRequest{From: &from, To: &to}
	_, err := h.UpdateAbsence(ctx, gen.UpdateAbsenceRequestObject{TeamId: uuid.New(), AbsenceId: uuid.New(), Body: body})

	require.Error(t, err)
}

// Regression test: Create had no membership re-check at all -- a membership
// removal racing a concurrent CreateAbsence call could otherwise silently
// leave an orphaned absence row for a non-member. ErrNotMember must surface
// as 404, matching RequireMembership's own "not found" (not "forbidden")
// convention for a non-member, rather than falling through to a generic 500.
func TestAbsenceHandler_CreateAbsence_NotMember_Returns404(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		create: func(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
			return gen.Absence{}, absences.ErrNotMember
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		From:   openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		To:     openapi_types.Date{Time: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)},
	}
	_, err := h.CreateAbsence(ctx, gen.CreateAbsenceRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok, "must map to an APIError, not fall through to the generic 500")
	assert.Equal(t, 404, apiErr.Status)
}

// Regression test: nothing prevented two overlapping absence entries for the
// same user; ErrOverlappingAbsence must surface as 422, not the generic 500
// a bare repository error would produce.
func TestAbsenceHandler_CreateAbsence_OverlappingRange_Returns422(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		create: func(context.Context, uuid.UUID, *gen.CreateAbsenceRequest) (gen.Absence, error) {
			return gen.Absence{}, absences.ErrOverlappingAbsence
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		From:   openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		To:     openapi_types.Date{Time: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)},
	}
	_, err := h.CreateAbsence(ctx, gen.CreateAbsenceRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok, "must map to an APIError, not fall through to the generic 500")
	assert.Equal(t, 422, apiErr.Status)
}

func TestAbsenceHandler_UpdateAbsence_OverlappingRange_Returns422(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockAbsenceService{
		update: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *gen.UpdateAbsenceRequest) (gen.Absence, error) {
			return gen.Absence{}, absences.ErrOverlappingAbsence
		},
	}
	h := absences.NewHandler(svc, slog.Default())
	ctx := auth.ContextWithUser(context.Background(), &auth.UserRow{Id: userID, Name: "Alice", Email: "a@x.c"})
	to := openapi_types.Date{Time: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)}
	body := &gen.UpdateAbsenceRequest{To: &to}
	_, err := h.UpdateAbsence(ctx, gen.UpdateAbsenceRequestObject{TeamId: uuid.New(), AbsenceId: uuid.New(), Body: body})

	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok, "must map to an APIError, not fall through to the generic 500")
	assert.Equal(t, 422, apiErr.Status)
}
