package core

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
)

// Reverser builds a URL for a named route. The router implements it and
// attaches itself to each Context so handlers can generate links.
type Reverser interface {
	Path(name string, params map[string]string) (string, error)
}

// Renderer executes a named view against data and returns the rendered bytes.
// The views package implements it. Rendering happens *before* any write, so a
// template error yields a complete error response, never a half-written page.
type Renderer interface {
	Render(name string, data any) ([]byte, error)
}

// Globals are request-scoped helpers available inside templates. Embed it in a
// view struct and pass a *pointer* to Context.Render; the framework fills it.
// URL returns an error, which html/template surfaces as a render error
// (fail-loud) rather than a silently empty link.
//
// Values carries the request-scoped data set by processors and handlers, so a
// view reads it without the app threading it into every view model:
//
//	{{with .Values.flash}}<p class="flash">{{.}}</p>{{end}}
type Globals struct {
	c *Context
}

// URL builds a reverse URL for use inside templates:
//
//	{{$.URL "note" "id" .ID}}
//
// Param values may be any type; they are stringified, so integer IDs work
// without a conversion in the template. Param names must be strings.
func (g Globals) URL(name string, pairs ...any) (string, error) {
	if g.c == nil {
		return "", errors.New("core: Globals used outside a request")
	}
	if len(pairs)%2 != 0 {
		return "", errors.New("core: URL expects name/value pairs")
	}
	params := make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return "", fmt.Errorf("core: URL param name must be a string, got %T", pairs[i])
		}
		params[key] = fmt.Sprint(pairs[i+1])
	}
	return g.c.URL(name, params)
}

// Context is the per-request state handed to a handler.
type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	Router Reverser
	Views  Renderer
	values map[string]any
}

// Set stores a request-scoped value for templates (see Globals.Values). A
// processor typically calls it; handlers may too. A later write wins.
func (c *Context) Set(name string, value any) {
	if c.values == nil {
		c.values = map[string]any{}
	}
	c.values[name] = value
}

// Global returns a value set with Set, or nil. It is the handler-side read of
// what templates see as .Values; use it when a handler needs the same value.
func (c *Context) Global(name string) any { return c.values[name] }

// Values returns the request-scoped values set by processors and handlers, for
// use inside templates: {{.Values.user}}. It is nil when nothing was set.
func (g Globals) Values() map[string]any {
	if g.c == nil {
		return nil
	}
	return g.c.values
}

// URL builds the URL for a named route, e.g.
// c.URL("user", map[string]string{"id": "1"}).
func (c *Context) URL(name string, params map[string]string) (string, error) {
	if c.Router == nil {
		return "", errors.New("core: no router attached to context")
	}
	return c.Router.Path(name, params)
}

// Globals returns this request's template helpers.
func (c *Context) Globals() Globals { return Globals{c: c} }

// CSRF is a ready-made hidden input carrying this request's CSRF token, for use
// inside a form. It is safe HTML; the token itself is escaped.
//
//	<form method="post"><{{.CSRF}} …</form>
//
// The field name is "csrf". With no session configured it renders nothing, so a
// form still submits (and the CSRF middleware, if any, decides).
func (g Globals) CSRF() template.HTML {
	if g.c == nil {
		return ""
	}
	tok := g.c.CSRF()
	if tok == "" {
		return ""
	}
	return template.HTML(`<input type="hidden" name="csrf" value="` + template.HTMLEscapeString(tok) + `">`)
}

var globalsType = reflect.TypeOf(Globals{})

// Render renders a named view and returns it as an HTML response. It fills an
// embedded Globals field on data (if present) and buffers the whole page: a
// render error becomes a 500 before any byte reaches the client.
func (c *Context) Render(status int, name string, data any) Response {
	if c.Views == nil {
		return Error(http.StatusInternalServerError, "core: no view renderer configured")
	}
	injectGlobals(data, c.Globals())
	b, err := c.Views.Render(name, data)
	if err != nil {
		return Error(http.StatusInternalServerError, err.Error())
	}
	return String(status, string(b))
}

// injectGlobals fills an embedded Globals field on a *struct so templates get
// working helpers without per-handler boilerplate. Non-pointer or Globals-less
// data is left untouched.
func injectGlobals(data any, g Globals) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	e := v.Elem()
	if e.Kind() != reflect.Struct {
		return
	}
	if f := e.FieldByName("Globals"); f.IsValid() && f.CanSet() && f.Type() == globalsType {
		f.Set(reflect.ValueOf(g))
	}
}

// Param returns a path parameter (e.g. {name}). It reads whatever the router
// installed on the request, so it is only populated on the dispatch path.
func (c *Context) Param(name string) string {
	return c.R.PathValue(name)
}

func (c *Context) Method() string { return c.R.Method }
func (c *Context) Path() string   { return c.R.URL.Path }

// Query returns a URL query value.
func (c *Context) Query(key string) string { return c.R.URL.Query().Get(key) }

// Form returns a submitted form value (URL query or body).
func (c *Context) Form(key string) string { return c.R.FormValue(key) }

// Cookie returns a request cookie by name.
func (c *Context) Cookie(name string) (*http.Cookie, error) { return c.R.Cookie(name) }
