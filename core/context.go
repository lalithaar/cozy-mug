package core

import "net/http"

// Context is the per-request state handed to a handler.
type Context struct {
	W http.ResponseWriter
	R *http.Request
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
