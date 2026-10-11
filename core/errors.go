package core

import (
	"fmt"
	"html"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
)

// Errors is the request error boundary. It recovers panics, turns Failure
// responses into rendered error pages, and logs both.
//
// Dev: renders a detail page (status, message, stack, request id) straight to
// the browser. Prod: renders the app's views/errors/<status>.html if present,
// else a plain page — and never leaks the cause.
func Errors(o Options) Middleware {
	log := o.logger()
	return func(next Handler) Handler {
		return func(c *Context) (resp Response) {
			defer func() {
				if v := recover(); v != nil {
					err := panicError(v)
					log.Error("panic", "err", err, "path", c.Path(), "id", c.RequestID())
					resp = o.errorResponse(c, Failure{Code: http.StatusInternalServerError, Err: err, stack: debug.Stack()})
				}
			}()
			resp = next(c)
			if f, ok := resp.(Failure); ok {
				if f.StatusCode() >= 500 {
					log.Error("request failed", "status", f.StatusCode(), "err", f.Err, "path", c.Path(), "id", c.RequestID())
				} else {
					log.Warn("request failed", "status", f.StatusCode(), "err", f.Err, "path", c.Path(), "id", c.RequestID())
				}
				resp = o.errorResponse(c, f)
			}
			return resp
		}
	}
}

func (o Options) errorResponse(c *Context, f Failure) Response {
	if o.Dev {
		return String(f.StatusCode(), devErrorPage(c, f))
	}
	name := "errors/" + strconv.Itoa(f.StatusCode())
	if vf, ok := c.Views.(interface{ Has(string) bool }); ok && vf.Has(name) {
		data := &struct{ Globals }{}
		injectGlobals(data, c.Globals())
		if b, err := c.Views.Render(name, data); err == nil {
			return String(f.StatusCode(), string(b))
		}
	}
	msg := f.Public
	if msg == "" {
		msg = statusText(f.StatusCode())
	}
	return Error(f.StatusCode(), msg)
}

func panicError(v any) error {
	if err, ok := v.(error); ok {
		return err
	}
	return fmt.Errorf("%v", v)
}

func devErrorPage(c *Context, f Failure) string {
	code := strconv.Itoa(f.StatusCode())
	title := html.EscapeString(statusText(f.StatusCode()))
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=\"utf-8\"><title>")
	b.WriteString(code)
	b.WriteString(" ")
	b.WriteString(title)
	b.WriteString("</title><h1>")
	b.WriteString(code)
	b.WriteString(" ")
	b.WriteString(title)
	b.WriteString("</h1><p><b>")
	b.WriteString(html.EscapeString(c.Method()))
	b.WriteString("</b> ")
	b.WriteString(html.EscapeString(c.Path()))
	if id := c.RequestID(); id != "" {
		b.WriteString(" <code>")
		b.WriteString(html.EscapeString(id))
		b.WriteString("</code>")
	}
	b.WriteString("</p>")
	if f.Err != nil {
		b.WriteString("<h2>error</h2><pre>")
		b.WriteString(html.EscapeString(f.Err.Error()))
		b.WriteString("</pre>")
	}
	if len(f.stack) > 0 {
		b.WriteString("<h2>stack</h2><pre>")
		b.WriteString(html.EscapeString(string(f.stack)))
		b.WriteString("</pre>")
	}
	return b.String()
}
