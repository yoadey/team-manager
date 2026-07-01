package apierror

import (
	"errors"
	"log/slog"
	"net/http"
)

// RequestErrorHandler returns a gen.StrictHTTPServerOptions.RequestErrorHandlerFunc
// that renders malformed request bodies (e.g. invalid JSON) as an RFC 9457
// Bad Request response instead of the library default (plain-text 400).
func RequestErrorHandler(logger *slog.Logger) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger.WarnContext(r.Context(), "request decode error", "err", err)
		BadRequest("the request body is malformed or missing required fields").Render(w)
	}
}

// ResponseErrorHandler returns a gen.StrictHTTPServerOptions.ResponseErrorHandlerFunc
// that renders errors returned by handler methods as RFC 9457 Problem Details.
//
// Handlers signal application errors by returning an *APIError as the error
// value (e.g. apierror.BadRequest(...), apierror.NotFound(...)); this is the
// only place that error reaches an http.ResponseWriter, since the generated
// strict-server dispatch treats every non-nil error uniformly. Without this
// handler, the library default writes err.Error() as a plain-text 500 for
// every error, which both reports the wrong status code and can leak
// internal error details (e.g. wrapped DB errors) to the client.
//
// Errors that are not an *APIError are logged with full detail server-side
// and reported to the client as a generic Internal Server Error with no
// error-specific detail.
func ResponseErrorHandler(logger *slog.Logger) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			apiErr.Render(w)
			return
		}
		logger.ErrorContext(r.Context(), "unhandled request error", "err", err)
		Internal("an unexpected error occurred").Render(w)
	}
}
