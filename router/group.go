package router

import (
	"net/http"
	"strings"

	"cozymug/core"
)

// Group mounts routes under a shared path prefix and middleware.
type Group struct {
	router     *Router
	prefix     string
	middleware []core.Middleware
}

// Group creates a group under prefix. Middleware passed here runs after any
// global router middleware and before the handler.
func (rt *Router) Group(prefix string, mw ...core.Middleware) *Group {
	return &Group{router: rt, prefix: strings.TrimRight(prefix, "/"), middleware: mw}
}

// Use appends middleware to the group (and its subgroups).
func (g *Group) Use(mw ...core.Middleware) {
	g.middleware = append(g.middleware, mw...)
}

// Group creates a nested group; the child inherits the parent's middleware.
func (g *Group) Group(prefix string, mw ...core.Middleware) *Group {
	inherited := append(append([]core.Middleware{}, g.middleware...), mw...)
	return &Group{
		router:     g.router,
		prefix:     g.prefix + strings.TrimRight(prefix, "/"),
		middleware: inherited,
	}
}

func (g *Group) Handle(method, pattern string, h core.Handler) *Route {
	full := g.prefix + pattern
	mw := append(append([]core.Middleware{}, g.router.middleware...), g.middleware...)
	return g.router.register(method+" "+full, method, full, core.Chain(h, mw...))
}

// Any registers h for every method under the group.
func (g *Group) Any(pattern string, h core.Handler) *Route {
	full := g.prefix + pattern
	mw := append(append([]core.Middleware{}, g.router.middleware...), g.middleware...)
	return g.router.register(full, "ANY", full, core.Chain(h, mw...))
}

func (g *Group) Get(pattern string, h core.Handler) *Route {
	return g.Handle(http.MethodGet, pattern, h)
}
func (g *Group) Post(pattern string, h core.Handler) *Route {
	return g.Handle(http.MethodPost, pattern, h)
}
func (g *Group) Put(pattern string, h core.Handler) *Route {
	return g.Handle(http.MethodPut, pattern, h)
}
func (g *Group) Patch(pattern string, h core.Handler) *Route {
	return g.Handle(http.MethodPatch, pattern, h)
}
func (g *Group) Delete(pattern string, h core.Handler) *Route {
	return g.Handle(http.MethodDelete, pattern, h)
}
