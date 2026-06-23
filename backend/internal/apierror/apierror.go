// Package apierror implements RFC 9457 Problem Details for HTTP APIs.
package apierror

import (
	"encoding/json"
	"net/http"
)

const baseURI = "https://teammanager.example/errors/"

// APIError represents an RFC 9457 Problem Details object.
type APIError struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// Error implements the error interface, returning the Detail field.
func (e *APIError) Error() string {
	return e.Detail
}

// Render writes the APIError as an RFC 9457 JSON response.
// It sets Content-Type to application/problem+json and the HTTP status code.
func (e *APIError) Render(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(e)
}

// New constructs a generic APIError with the given status, title, and detail.
// The Type URI is derived from the title slug.
func New(status int, title, detail string) *APIError {
	return &APIError{
		Type:   typeURI(status),
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

// typeURI maps an HTTP status code to a canonical Type URI.
func typeURI(status int) string {
	switch status {
	case http.StatusBadRequest:
		return baseURI + "bad-request"
	case http.StatusUnauthorized:
		return baseURI + "unauthorized"
	case http.StatusForbidden:
		return baseURI + "forbidden"
	case http.StatusNotFound:
		return baseURI + "not-found"
	case http.StatusConflict:
		return baseURI + "conflict"
	case http.StatusUnprocessableEntity:
		return baseURI + "unprocessable-entity"
	case http.StatusTooManyRequests:
		return baseURI + "too-many-requests"
	case http.StatusInternalServerError:
		return baseURI + "internal-server-error"
	default:
		return baseURI + "error"
	}
}

// BadRequest returns a 400 Bad Request APIError.
func BadRequest(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "bad-request",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: detail,
	}
}

// Unauthorized returns a 401 Unauthorized APIError.
func Unauthorized(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "unauthorized",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: detail,
	}
}

// Forbidden returns a 403 Forbidden APIError.
func Forbidden(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "forbidden",
		Title:  "Forbidden",
		Status: http.StatusForbidden,
		Detail: detail,
	}
}

// NotFound returns a 404 Not Found APIError.
func NotFound(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "not-found",
		Title:  "Not Found",
		Status: http.StatusNotFound,
		Detail: detail,
	}
}

// Conflict returns a 409 Conflict APIError.
func Conflict(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "conflict",
		Title:  "Conflict",
		Status: http.StatusConflict,
		Detail: detail,
	}
}

// UnprocessableEntity returns a 422 Unprocessable Entity APIError.
func UnprocessableEntity(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "unprocessable-entity",
		Title:  "Unprocessable Entity",
		Status: http.StatusUnprocessableEntity,
		Detail: detail,
	}
}

// Internal returns a 500 Internal Server Error APIError.
func Internal(detail string) *APIError {
	return &APIError{
		Type:   baseURI + "internal-server-error",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: detail,
	}
}
