package auth

import (
	"errors"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/session"
)

const (
	DefaultSessionKey = "cozy_auth_user_id"
	DefaultLoginURL   = "/login"
)

// Options tunes the session key used by auth. The zero value is usable.
type Options struct {
	// SessionKey overrides the reserved session key. Empty uses DefaultSessionKey.
	SessionKey string
}

// sessionID is the effective session key for opts.
func sessionKey(opts Options) string {
	if opts.SessionKey != "" {
		return opts.SessionKey
	}
	return DefaultSessionKey
}

// Login marks userID as logged in after regenerating the session first.
// It requires a session store (session.From(c) != nil).
func Login(c *core.Context, userID any) error {
	return LoginWithOptions(c, userID, Options{})
}

// LoginWithOptions is Login with an overridden reserved session key.
func LoginWithOptions(c *core.Context, userID any, opts Options) error {
	if c == nil {
		return errors.New("auth: nil context")
	}
	if userID == nil {
		return errors.New("auth: nil user id")
	}
	sess := session.From(c)
	if sess == nil {
		return errors.New("auth: no session configured for this request")
	}
	// Regenerate before storing identity to avoid session fixation.
	sess.Regenerate()
	sess.Set(sessionKey(opts), userID)
	return nil
}

// Logout removes the stored identity. It keeps the session and other values
// intact so the browser retains its CSRF token for the logged-out page.
func Logout(c *core.Context) error {
	return LogoutWithOptions(c, Options{})
}

// LogoutWithOptions is Logout with an overridden reserved session key.
func LogoutWithOptions(c *core.Context, opts Options) error {
	if c == nil {
		return errors.New("auth: nil context")
	}
	sess := session.From(c)
	if sess == nil {
		return errors.New("auth: no session configured for this request")
	}
	sess.Delete(sessionKey(opts))
	return nil
}

// UserID returns the logged-in user ID (and true) when present.
func UserID(c *core.Context) (any, bool) {
	return UserIDWithOptions(c, Options{})
}

// UserIDWithOptions is UserID with an overridden reserved session key.
func UserIDWithOptions(c *core.Context, opts Options) (any, bool) {
	if c == nil {
		return nil, false
	}
	sess := session.From(c)
	if sess == nil {
		return nil, false
	}
	value := sess.Get(sessionKey(opts))
	if value == nil {
		return nil, false
	}
	return value, true
}

// Authenticated reports whether a user ID is present in the session.
func Authenticated(c *core.Context) bool {
	_, ok := UserID(c)
	return ok
}

// DestroySession removes all session state (identity included). Use this when a
// logout should also discard flash/messages/CSRF state.
func DestroySession(c *core.Context) error {
	sess := session.From(c)
	if sess == nil {
		return errors.New("auth: no session configured for this request")
	}
	sess.Destroy()
	return nil
}
