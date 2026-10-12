package router

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/lalithaar/cozy-mug/core"
)

// Router wraps net/http's ServeMux (Go 1.22+ patterns) and adds middleware,
// named routes with reverse URL generation, route groups, and canonical
// trailing-slash handling. Matching still belongs to mux.
type Router struct {
	mux            *http.ServeMux
	routes         []*Route
	byName         map[string]*Route
	subtrees       map[string]bool
	middleware     []core.Middleware
	canonicalSlash bool
}

func New() *Router {
	return &Router{
		mux:            http.NewServeMux(),
		byName:         map[string]*Route{},
		subtrees:       map[string]bool{},
		canonicalSlash: true,
	}
}

// Use appends middleware applied to every route registered after the call.
// The first middleware added is the outermost.
func (rt *Router) Use(mw ...core.Middleware) {
	rt.middleware = append(rt.middleware, mw...)
}

// DisableCanonicalSlash turns off the default trailing-slash redirect.
func (rt *Router) DisableCanonicalSlash() { rt.canonicalSlash = false }

// chain wraps h with the reverse-URL injector, then global middleware, then
// any extra (group) middleware. The injector is always outermost so every
// handler sees a Context with a working URL helper.
func (rt *Router) chain(h core.Handler, extra ...core.Middleware) core.Handler {
	mw := make([]core.Middleware, 0, 1+len(rt.middleware)+len(extra))
	mw = append(mw, rt.inject)
	mw = append(mw, rt.middleware...)
	mw = append(mw, extra...)
	return core.Chain(h, mw...)
}

func (rt *Router) inject(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		c.Router = rt
		return next(c)
	}
}

// Route is a registered route. Chain As to name it for reverse lookups.
type Route struct {
	router  *Router
	method  string
	pattern string
	name    string
	params  []param
}

type param struct {
	name     string
	raw      string
	wildcard bool
}

// As names the route so Path can generate its URL. Duplicate names panic at
// registration (fail-fast, like ServeMux's duplicate patterns).
func (r *Route) As(name string) *Route {
	if existing, ok := r.router.byName[name]; ok && existing != r {
		panic("router: duplicate route name: " + name)
	}
	if r.name != "" && r.name != name {
		delete(r.router.byName, r.name)
	}
	r.name = name
	r.router.byName[name] = r
	return r
}

func (r *Route) Name() string    { return r.name }
func (r *Route) Method() string  { return r.method }
func (r *Route) Pattern() string { return r.pattern }

// Handle registers h for one method and pattern, e.g. ("GET", "/users/{id}").
func (rt *Router) Handle(method, pattern string, h core.Handler) *Route {
	return rt.register(method+" "+exactRoot(pattern), method, pattern, rt.chain(h))
}

// Any registers h for every HTTP method (a method-less mux pattern).
func (rt *Router) Any(pattern string, h core.Handler) *Route {
	return rt.register(exactRoot(pattern), "ANY", pattern, rt.chain(h))
}

// exactRoot makes a bare "/" match only the root. ServeMux treats "GET /" as a
// catch-all subtree, which would shadow Router.NotFound (a "/" registration) for
// every GET path. A catch-all is expressed with NotFound (or an explicit
// "subtree/" pattern), chi-style.
func exactRoot(pattern string) string {
	if pattern == "/" {
		return "/{$}"
	}
	return pattern
}

func (rt *Router) Get(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodGet, pattern, h)
}
func (rt *Router) Post(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodPost, pattern, h)
}
func (rt *Router) Put(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodPut, pattern, h)
}
func (rt *Router) Patch(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodPatch, pattern, h)
}
func (rt *Router) Delete(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodDelete, pattern, h)
}
func (rt *Router) Head(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodHead, pattern, h)
}
func (rt *Router) Options(pattern string, h core.Handler) *Route {
	return rt.Handle(http.MethodOptions, pattern, h)
}

// NotFound sets the fallback handler for unmatched routes.
func (rt *Router) NotFound(h core.Handler) {
	rt.mux.Handle("/", rt.chain(h))
}

func (rt *Router) register(muxPattern, method, pattern string, h core.Handler) *Route {
	rt.mux.Handle(muxPattern, h)
	if strings.HasSuffix(pattern, "/") {
		rt.subtrees[pattern] = true
	}
	r := &Route{router: rt, method: method, pattern: pattern, params: patternParams(pattern)}
	rt.routes = append(rt.routes, r)
	return r
}

// Routes returns every registered route, for introspection (the agent CLI).
func (rt *Router) Routes() []*Route { return rt.routes }

// Path builds the URL for a named route. A "{name...}" wildcard may contain
// slashes; every other param is escaped as a single path segment.
func (rt *Router) Path(name string, params map[string]string) (string, error) {
	r, ok := rt.byName[name]
	if !ok {
		return "", fmt.Errorf("router: unknown route %q", name)
	}
	out := strings.ReplaceAll(r.pattern, "{$}", "")
	for _, p := range r.params {
		v, ok := params[p.name]
		if !ok {
			return "", fmt.Errorf("router: route %q missing param %q", name, p.name)
		}
		out = strings.Replace(out, p.raw, escapeParam(v, p.wildcard), 1)
	}
	return out, nil
}

// Static serves files from dir at the URL prefix (e.g. "/static/").
func (rt *Router) Static(prefix, dir string) {
	rt.mount(prefix, http.FileServer(http.Dir(dir)))
}

// StaticFS serves files from fsys at the URL prefix, for embedded assets.
func (rt *Router) StaticFS(prefix string, fsys fs.FS) {
	rt.mount(prefix, http.FileServerFS(fsys))
}

func (rt *Router) mount(prefix string, h http.Handler) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	rt.subtrees[prefix] = true
	rt.mux.Handle("GET "+prefix, http.StripPrefix(prefix, h))
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rt.canonicalSlash && rt.redirectCanonicalSlash(w, r) {
		return
	}
	rt.mux.ServeHTTP(w, r)
}

// redirectCanonicalSlash sends "/foo/" to "/foo" (the bare form is canonical),
// except for registered subtree/mount patterns, which are directory-like.
func (rt *Router) redirectCanonicalSlash(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	if len(p) <= 1 || !strings.HasSuffix(p, "/") || rt.subtrees[p] {
		return false
	}
	// path.Clean collapses "//", so the target can never be a protocol-relative
	// ("//host") Location — the open-redirect guard.
	target := path.Clean(strings.TrimRight(p, "/"))
	if target == "." {
		target = "/"
	}
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	code := http.StatusMovedPermanently // 301 for GET/HEAD
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		code = http.StatusPermanentRedirect // 308 preserves the method + body
	}
	http.Redirect(w, r, target, code)
	return true
}

func patternParams(pattern string) []param {
	var out []param
	for {
		i := strings.IndexByte(pattern, '{')
		if i < 0 {
			return out
		}
		j := strings.IndexByte(pattern[i:], '}')
		if j < 0 {
			return out
		}
		inner := pattern[i+1 : i+j]
		if inner != "" && inner != "$" {
			out = append(out, param{
				name:     strings.TrimSuffix(inner, "..."),
				raw:      pattern[i : i+j+1],
				wildcard: strings.HasSuffix(inner, "..."),
			})
		}
		pattern = pattern[i+j+1:]
	}
}

func escapeParam(v string, wildcard bool) string {
	if !wildcard {
		return url.PathEscape(v)
	}
	parts := strings.Split(v, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
