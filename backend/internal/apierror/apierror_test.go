package apierror_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
)

func TestAPIError_Error(t *testing.T) {
	err := apierror.New(http.StatusBadRequest, "Bad Request", "the id field is required")
	assert.Equal(t, "the id field is required", err.Error())
}

func TestAPIError_Render(t *testing.T) {
	err := apierror.NotFound("resource with id 42 not found")

	rec := httptest.NewRecorder()
	err.Render(rec)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var body apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, http.StatusNotFound, body.Status)
	assert.Equal(t, "Not Found", body.Title)
	assert.Equal(t, "resource with id 42 not found", body.Detail)
	assert.Contains(t, body.Type, "not-found")
}

func TestAPIError_Render_StatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        *apierror.APIError
		wantStatus int
	}{
		{"BadRequest", apierror.BadRequest("bad"), http.StatusBadRequest},
		{"Unauthorized", apierror.Unauthorized("unauth"), http.StatusUnauthorized},
		{"Forbidden", apierror.Forbidden("forbid"), http.StatusForbidden},
		{"NotFound", apierror.NotFound("nf"), http.StatusNotFound},
		{"Conflict", apierror.Conflict("conflict"), http.StatusConflict},
		{"UnprocessableEntity", apierror.UnprocessableEntity("upe"), http.StatusUnprocessableEntity},
		{"Internal", apierror.Internal("ise"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.err.Render(rec)
			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, tc.wantStatus, tc.err.Status)
			assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
		})
	}
}

func TestConstructors_SetTypeURI(t *testing.T) {
	const base = "https://teammanager.example/errors/"

	tests := []struct {
		name     string
		err      *apierror.APIError
		wantSlug string
	}{
		{"BadRequest", apierror.BadRequest("d"), "bad-request"},
		{"Unauthorized", apierror.Unauthorized("d"), "unauthorized"},
		{"Forbidden", apierror.Forbidden("d"), "forbidden"},
		{"NotFound", apierror.NotFound("d"), "not-found"},
		{"Conflict", apierror.Conflict("d"), "conflict"},
		{"UnprocessableEntity", apierror.UnprocessableEntity("d"), "unprocessable-entity"},
		{"Internal", apierror.Internal("d"), "internal-server-error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, base+tc.wantSlug, tc.err.Type)
		})
	}
}

func TestNew_CustomStatus(t *testing.T) {
	err := apierror.New(http.StatusTeapot, "I'm a teapot", "short and stout")
	assert.Equal(t, http.StatusTeapot, err.Status)
	assert.Equal(t, "I'm a teapot", err.Title)
	assert.Equal(t, "short and stout", err.Detail)
	assert.Equal(t, "short and stout", err.Error())

	rec := httptest.NewRecorder()
	err.Render(rec)
	assert.Equal(t, http.StatusTeapot, rec.Code)
}
