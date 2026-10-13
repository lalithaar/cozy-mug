package core

import (
	"errors"
	"net/http"
)

// HTTPError is a structured, status-aware error — the tier-2 path. Loaders
// return one; the route maps it to a status code and the matching error layout.
type HTTPError struct {
	Status int
	Public string // user-safe message; empty falls back to the status text
	Cause  error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return http.StatusText(e.Status)
}

func (e *HTTPError) Unwrap() error { return e.Cause }

// Sugar for the common statuses.
func NotFound(cause error) *HTTPError { return &HTTPError{Status: http.StatusNotFound, Cause: cause} }
func BadRequest(cause error) *HTTPError {
	return &HTTPError{Status: http.StatusBadRequest, Cause: cause}
}
func Unauthorized(cause error) *HTTPError {
	return &HTTPError{Status: http.StatusUnauthorized, Cause: cause}
}
func Forbidden(cause error) *HTTPError { return &HTTPError{Status: http.StatusForbidden, Cause: cause} }
func Conflict(cause error) *HTTPError {
	return &HTTPError{Status: http.StatusConflict, Cause: cause}
}
func Internal(cause error) *HTTPError {
	return &HTTPError{Status: http.StatusInternalServerError, Cause: cause}
}

// Failure is a Response that carries an error to the Errors middleware, which
// logs it and renders the dev/prod error page. With no such middleware it still
// writes a plain page, so a handler can never leave the client with nothing.
type Failure struct {
	Code   int
	Public string
	Err    error
	stack  []byte
}

// Fail wraps err as an error response, taking the status/message from an
// HTTPError when present (via errors.As), otherwise 500.
func Fail(err error) Failure {
	var he *HTTPError
	if errors.As(err, &he) {
		return Failure{Code: he.Status, Public: he.Public, Err: he.Cause}
	}
	return Failure{Code: http.StatusInternalServerError, Err: err}
}

func (f Failure) StatusCode() int {
	if f.Code == 0 {
		return http.StatusInternalServerError
	}
	return f.Code
}

func (f Failure) WriteTo(w http.ResponseWriter, r *http.Request) {
	msg := f.Public
	if msg == "" {
		msg = statusText(f.StatusCode())
	}
	Error(f.StatusCode(), msg).WriteTo(w, r)
}

// Error turns err into an error response. The Errors middleware renders it;
// without the middleware it writes a plain page.
func (c *Context) Error(err error) Response { return Fail(err) }

// Must panics on err — for boot/bootstrap failures only (tier 1), e.g.
// core.Must(db.Open(dsn)) in main. Never use it for request-time failures.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
