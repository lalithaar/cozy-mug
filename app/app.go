// Package app assembles a cozy-mug application (ADR 0009).
//
// It wires the pieces in the correct order — Logger (outermost) -> Errors ->
// Session -> Processors -> Views -> CSRF -> Router — mounts static assets, and
// exposes the result as an http.Handler. Apps never hand-wire middleware order;
// the ordering knowledge lives here.
package app

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/session"
	"github.com/lalithaar/cozy-mug/ui"
	"github.com/lalithaar/cozy-mug/views"
)

// Config assembles an App.
type Config struct {
	// Assets is an fs.FS containing the app's "views/" directory (see 0006) and,
	// optionally, a "static/" directory. Usually //go:embed views static.
	Assets fs.FS

	// Routes registers the app's routes. Routing lives in the router package;
	// this is the route table (e.g. a handlers.Register function). May be nil.
	Routes func(*router.Router)

	// Dev enables verbose error pages (see 0007).
	Dev bool

	// Logger receives the request log. nil uses slog.Default().
	Logger *slog.Logger

	// UI includes the default ui component library: its template funcs are
	// merged and its stylesheet is mounted at /static/ui/.
	UI bool

	// Funcs are the app's own template funcs (optional).
	Funcs template.FuncMap

	// Session enables sessions and CSRF protection (0013). nil disables both.
	// The default is session.NewStore(db) (server-side, revocable); session.New
	// seals sessions into the cookie instead. The example wires it here.
	Session *session.Store

	// Processors compute request-scoped values (current user, flash) exposed to
	// every view as {{.Values.name}} (gap 7). They run after sessions and before
	// views, so a processor may read the session. Optional.
	Processors []core.Processor
}

// App is an assembled application. It is an http.Handler, so tests and the agent
// CLI drive it in-process via ServeHTTP.
type App struct {
	cfg    Config
	router *router.Router
	views  *views.Set
}

// New assembles an App from cfg.
func New(cfg Config) (*App, error) {
	if cfg.Assets == nil {
		return nil, errors.New("app: Config.Assets is required (embed the views/ directory)")
	}
	viewFS, err := fs.Sub(cfg.Assets, "views")
	if err != nil {
		return nil, fmt.Errorf("app: no views/ directory in Assets: %w", err)
	}

	funcs := template.FuncMap{}
	for name, fn := range cfg.Funcs {
		funcs[name] = fn
	}
	if cfg.UI {
		for name, fn := range ui.Funcs() {
			funcs[name] = fn
		}
	}
	set, err := views.New(viewFS, funcs)
	if err != nil {
		return nil, err
	}

	r := router.New()
	opts := core.Options{Dev: cfg.Dev, Logger: cfg.Logger}
	r.Use(core.Logger(opts), core.Errors(opts))
	if cfg.Session != nil {
		r.Use(cfg.Session.Middleware)
	}
	if len(cfg.Processors) > 0 {
		r.Use(core.Processors(cfg.Processors...))
	}
	// Views before CSRF: a rejected unsafe request still renders its error page
	// with the global values. Views installs the renderer, it does not dispatch.
	r.Use(views.Middleware(set))
	if cfg.Session != nil {
		r.Use(cfg.Session.CSRFMiddleware)
	}

	if cfg.UI {
		r.StaticFS("/static/ui/", ui.FS())
	}
	if staticFS, err := fs.Sub(cfg.Assets, "static"); err == nil {
		r.StaticFS("/static/", staticFS)
	}
	if cfg.Routes != nil {
		cfg.Routes(r)
	}

	return &App{cfg: cfg, router: r, views: set}, nil
}

// Must is New, panicking on error. Boot-only (see 0007).
func Must(cfg Config) *App {
	a, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return a
}

// Router exposes the underlying router as an escape hatch (e.g. to add routes
// after assembly).
func (a *App) Router() *router.Router { return a.router }

// ServeHTTP dispatches the request through the assembled router.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) { a.router.ServeHTTP(w, r) }

// ListenAndServe serves the app on addr.
func (a *App) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, a)
}
