package auth

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/lalithaar/cozy-mug/core"
)

// GuardOptions configures a guard. The zero value is usable: session key
// DefaultSessionKey, login URL DefaultLoginURL, no next-path parameter.
type GuardOptions struct {
	SessionKey string
	LoginURL   string

	// IncludeNext adds ?next=<current path> to the login URL. The value is
	// URL-escaped request state, never a redirect target, so it cannot become
	// an open redirect.
	IncludeNext bool
}

// loginURL returns the configured (or default) login URL, or an error if it
// is empty/unsafe relative.
func (o GuardOptions) loginURL() (string, error) {
	u := o.LoginURL
	if u == "" {
		u = DefaultLoginURL
	}
	if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") {
		return "", fmt.Errorf("auth: unsafe login URL %q", u)
	}
	return u, nil
}

func (o GuardOptions) key() string { return sessionKey(o.Options()) }

// Options converts guard options into the shared option shape.
func (o GuardOptions) Options() Options {
	return Options{SessionKey: o.SessionKey}
}

// RequireLoginMiddleware returns an http.Handler wrapper that redirects
// unauthenticated requests to the login page.
func RequireLoginMiddleware(opts ...GuardOptions) func(core.Handler) core.Handler {
	o := GuardOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	return func(next core.Handler) core.Handler {
		return func(c *core.Context) core.Response {
			if AuthenticatedWithOptions(c, o.Options()) {
				return next(c)
			}
			login, err := o.loginURL()
			if err != nil {
				return c.Error(err)
			}
			target := login
			if o.IncludeNext && c != nil && c.Path() != "" {
				values := url.Values{}
				values.Set("next", c.Path())
				target = login + "?" + values.Encode()
			}
			// Redirect target is a fixed configured URL plus an escaped query
			// value; it is never taken from request input (redirects are
			// security surfaces, see heuristics).
			return core.Redirect(http.StatusSeeOther, target)
		}
	}
}

// AuthenticatedWithOptions is Authenticated with a configured session key.
func AuthenticatedWithOptions(c *core.Context, opts Options) bool {
	_, ok := UserIDWithOptions(c, opts)
	return ok
}

// RequireRoleMiddleware rejects requests where check returns false. A denied
// request gets a 403 by default; onDeny may override the response.
func RequireRoleMiddleware(check func(user any) bool, onDeny func(*core.Context) core.Response) func(core.Handler) core.Handler {
	return func(next core.Handler) core.Handler {
		return func(c *core.Context) core.Response {
			id, ok := UserID(c)
			if !ok {
				// Keep this focused on authorization; authentication is a
				// separate concern handled by RequireLoginMiddleware.
				if onDeny != nil {
					return onDeny(c)
				}
				return core.Error(http.StatusForbidden, "forbidden")
			}
			// Without a current user object, role checks at middleware level use
			// only the id. Apps that need richer behavior should guard inside
			// handlers with the loaded user from a processor.
			user := c.Global("user")
			if user == nil {
				user = id
			}
			if check != nil && !check(user) {
				if onDeny != nil {
					return onDeny(c)
				}
				return core.Error(http.StatusForbidden, "forbidden")
			}
			return next(c)
		}
	}
}

// CheckLogin is useful in handlers when a redirect is not desired.
func CheckLogin(c *core.Context, opts ...GuardOptions) error {
	o := GuardOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	if AuthenticatedWithOptions(c, o.Options()) {
		return nil
	}
	return errors.New("auth: not authenticated")
}

// CanonicalLoginPath cleans a configured login path defensively.
func CanonicalLoginPath(raw string) string {
	return path.Clean(raw)
}
