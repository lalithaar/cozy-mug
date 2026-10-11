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
