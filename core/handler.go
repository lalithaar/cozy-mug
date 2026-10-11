package core

import "net/http"

// Handler is a view: it returns a Response instead of writing one. Returning a
// value rather than taking a ResponseWriter keeps a view callable as a plain
// function — the seam the agent CLI and tests rely on.
type Handler func(*Context) Response

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if resp := h(&Context{W: w, R: r}); resp != nil {
		resp.WriteTo(w, r)
	}
}
