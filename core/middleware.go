package core

// Middleware wraps a handler. The first middleware passed to Chain is the
// outermost (runs first on the way in, last on the way out).
type Middleware func(Handler) Handler

// Chain applies middleware around h, first argument = outermost.
func Chain(h Handler, mw ...Middleware) Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// Processor computes request-scoped values exposed to every view. Each returns a
// map merged into Globals.Values, so a template reads it as {{.Values.key}}.
// A processor may read the session and other Context state; it runs once per
// request. Return nil to contribute nothing.
type Processor func(*Context) map[string]any

// Processors installs middleware that runs fns once per request and stores their
// values for templates ({{.Values.key}}). Run it after sessions (so a processor
// can read the session) and before views. Later processors win on key conflicts.
func Processors(fns ...Processor) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) Response {
			for _, fn := range fns {
				if fn == nil {
					continue
				}
				for name, value := range fn(c) {
					c.Set(name, value)
				}
			}
			return next(c)
		}
	}
}
