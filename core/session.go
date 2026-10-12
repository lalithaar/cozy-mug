package core

import (
	"context"
	"net/http"
)

// Session is the per-request session seam. The session package implements it and
// installs it on the request; handlers read it with Context.Session. Only the
// operations the framework and its templates need live on this interface —
// richer helpers stay in the session package (ADR 0013).
type Session interface {
	Get(key string) any
	Set(key string, value any)
	Delete(key string)
	Clear()
	Destroy()
	Dirty() bool
	Regenerate()
	CSRFToken() string
}

type sessionKey struct{}

// WithSession returns r with s attached as the request's session. The session
// middleware calls this; handlers normally never need it.
func WithSession(r *http.Request, s Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), sessionKey{}, s))
}

// Session returns this request's session, or nil when the app did not configure
// sessions.
func (c *Context) Session() Session {
	s, _ := c.R.Context().Value(sessionKey{}).(Session)
	return s
}

// CSRF returns the request's CSRF token, or "" when there is no session. Most
// templates use Globals.CSRF instead; this is the escape hatch for hand-written
// markup or a JSON/form header.
func (c *Context) CSRF() string {
	if s := c.Session(); s != nil {
		return s.CSRFToken()
	}
	return ""
}
