package auth

import (
	"context"

	"github.com/lalithaar/cozy-mug/core"
)

// UserLoader resolves a user object from a stored session ID.
type UserLoader func(context.Context, any) (any, error)

// UserProcessor returns a core.Processor that exposes the logged-in user to
// every view as {{.Values.user}}. Apps with a custom User type supply a
// UserLoader; apps that only need the ID may pass nil.
func UserProcessor(loader UserLoader) core.Processor {
	return UserProcessorWithOptions(loader, Options{})
}

// UserProcessorWithOptions is UserProcessor with a configured session key.
func UserProcessorWithOptions(loader UserLoader, opts Options) core.Processor {
	return func(c *core.Context) map[string]any {
		id, ok := UserIDWithOptions(c, opts)
		if !ok {
			return map[string]any{"user": nil}
		}
		if loader == nil {
			return map[string]any{"user": id}
		}
		user, err := loader(c.R.Context(), id)
		if err != nil || user == nil {
			return map[string]any{"user": nil}
		}
		return map[string]any{"user": user}
	}
}

// From returns the user set by UserProcessor, or nil.
func From(c *core.Context) any {
	if c == nil {
		return nil
	}
	return c.Global("user")
}
