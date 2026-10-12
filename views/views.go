// Package views composes server-rendered HTML over stdlib html/template.
//
// The engine is fixed: html/template already auto-escapes (XSS), so there is no
// third-party template language and no hand-rolled escaping (frugality, 0001).
// Composition uses the stdlib {{block}} model: a base layout defines named
// blocks; each view overrides them by re-defining the same names.
//
// Layout convention: files under "layouts/" and "partials/" are shared, every
// other *.html file is a view named by its path without the extension
// (e.g. "users/show"). A view defines "content" (and optionally "title", etc.)
// to fill the base layout's {{block}}s.
package views

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"

	"github.com/lalithaar/cozy-mug/core"
)

// DefaultLayout is the template name executed for every view.
const DefaultLayout = "base"

// Set is a parsed, immutable collection of views. It is safe for concurrent use.
type Set struct {
	views  map[string]*template.Template
	layout string
}

// New parses shared layouts/partials once, then clones that base for each view
// so a view's {{define}}s override the base {{block}}s without colliding with
// other views. Parse errors are returned with file+line (fail-fast at startup).
func New(fsys fs.FS, funcs template.FuncMap) (*Set, error) {
	base := template.New("views")
	if len(funcs) > 0 {
		base = base.Funcs(funcs)
	}
	var err error
	if base, err = parse(base, fsys, "layouts/*.html", "partials/*.html"); err != nil {
		return nil, fmt.Errorf("views: parse layouts/partials: %w", err)
	}
	if base.Lookup(DefaultLayout) == nil {
		return nil, fmt.Errorf("views: layout %q not found under layouts/", DefaultLayout)
	}

	s := &Set{views: map[string]*template.Template{}, layout: DefaultLayout}
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		if dir := path.Dir(p); dir == "layouts" || dir == "partials" {
			return nil
		}
		view, err := base.Clone()
		if err != nil {
			return err
		}
		if view, err = parse(view, fsys, p); err != nil {
			return fmt.Errorf("views: parse %s: %w", p, err)
		}
		s.views[strings.TrimSuffix(p, ".html")] = view
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// parse adds the files matched by patterns to t, tolerating patterns that match
// nothing (so an absent partials/ directory is fine).
func parse(t *template.Template, fsys fs.FS, patterns ...string) (*template.Template, error) {
	var files []string
	for _, pat := range patterns {
		m, err := fs.Glob(fsys, pat)
		if err != nil {
			return nil, err
		}
		files = append(files, m...)
	}
	if len(files) == 0 {
		return t, nil
	}
	return t.ParseFS(fsys, files...)
}

// Layout returns the template name each view renders into.
func (s *Set) Layout() string { return s.layout }

// Has reports whether a view with the given name was parsed. The core Errors
// middleware uses it to find an app error view (errors/<status>).
func (s *Set) Has(name string) bool {
	_, ok := s.views[name]
	return ok
}

// Render executes view name into a buffer and returns the bytes. Nothing is
// written on error; the caller turns the error into a complete response.
func (s *Set) Render(name string, data any) ([]byte, error) {
	t, ok := s.views[name]
	if !ok {
		return nil, fmt.Errorf("views: no such view %q", name)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, s.layout, data); err != nil {
		return nil, fmt.Errorf("views: render %q: %w", name, err)
	}
	return buf.Bytes(), nil
}

var _ core.Renderer = (*Set)(nil)

// Middleware attaches s to every request's Context so handlers can call
// c.Render.
func Middleware(s *Set) core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(c *core.Context) core.Response {
			c.Views = s
			return next(c)
		}
	}
}
