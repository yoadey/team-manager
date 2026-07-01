package apierror_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestResponseErrorHandler_RendersAPIErrorWithCorrectStatus(t *testing.T) {
	handler := apierror.ResponseErrorHandler(silentLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/teams", http.NoBody)
	handler(rec, req, apierror.NotFound("team not found"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var body apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "team not found", body.Detail)
}

func TestResponseErrorHandler_WrappedAPIErrorStillRenders(t *testing.T) {
	handler := apierror.ResponseErrorHandler(silentLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/teams", http.NoBody)
	wrapped := fmt.Errorf("teams.Handler.CreateTeam: %w", apierror.BadRequest("name is required"))
	handler(rec, req, wrapped)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

func TestResponseErrorHandler_GenericErrorDoesNotLeakDetail(t *testing.T) {
	handler := apierror.ResponseErrorHandler(silentLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/teams", http.NoBody)
	handler(rec, req, errors.New("pq: connection refused to internal-db-host:5432"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
	assert.NotContains(t, rec.Body.String(), "internal-db-host")

	var body apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "an unexpected error occurred", body.Detail)
}

func TestRequestErrorHandler_RendersBadRequest(t *testing.T) {
	handler := apierror.RequestErrorHandler(silentLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/teams", http.NoBody)
	handler(rec, req, errors.New("can't decode JSON body: unexpected EOF"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

// stubServer implements gen.StrictServerInterface by embedding it (nil) and
// overriding only CreateTeam, to exercise the real generated dispatch path
// (gen.NewStrictHandlerWithOptions -> strictHandler.CreateTeam) end-to-end
// rather than calling the handler function in isolation. This is what
// verifies the actual bug: without wiring RequestErrorHandler/
// ResponseErrorHandler, the generated dispatcher's own defaults would
// return a plain-text 500 instead of the apierror.APIError's real status.
type stubServer struct {
	gen.StrictServerInterface
	createTeamErr error
}

func (s *stubServer) CreateTeam(_ context.Context, _ gen.CreateTeamRequestObject) (gen.CreateTeamResponseObject, error) {
	return nil, s.createTeamErr
}

func TestStrictHandler_EndToEnd_RendersAPIErrorNotGeneric500(t *testing.T) {
	stub := &stubServer{createTeamErr: apierror.BadRequest("name is required")}
	strictSrv := gen.NewStrictHandlerWithOptions(stub, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  apierror.RequestErrorHandler(silentLogger()),
		ResponseErrorHandlerFunc: apierror.ResponseErrorHandler(silentLogger()),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/teams", strings.NewReader(`{"name":"Test"}`))

	strictSrv.CreateTeam(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "expected the apierror's own status, not the library default 500")
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var body apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "name is required", body.Detail)
}
