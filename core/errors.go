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
	method := html.EscapeString(c.Method())
	path := html.EscapeString(c.Path())
	id := c.RequestID()
	if id == "" {
		id = "none"
	}
	id = html.EscapeString(id)
	errText := ""
	if f.Err != nil {
		errText = html.EscapeString(f.Err.Error())
	}
	stack := ""
	if len(f.stack) > 0 {
		stack = html.EscapeString(string(f.stack))
	}
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	b.WriteString("<title>")
	b.WriteString(code)
	b.WriteString(" ")
	b.WriteString(title)
	b.WriteString(" — cozy-mug (dev)</title><style>")
	b.WriteString(":root{color-scheme:light dark}body{margin:0;background:Canvas;color:CanvasText;font:15px/1.55 ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif}")
	b.WriteString("main{max-width:960px;margin:0 auto;padding:28px 20px 48px}")
	b.WriteString("header{margin-bottom:24px}h1{font-size:2rem;line-height:1.2;margin:0 0 8px}h2{font-size:1.05rem;margin:20px 0 8px}")
	b.WriteString(".eyebrow{margin:0 0 6px;font-size:.8rem;letter-spacing:.08em;text-transform:uppercase;opacity:.65}")
	b.WriteString(".status{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:2.25rem;font-weight:700}")
	b.WriteString(".summary{margin:0;opacity:.85}.summary code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}")
	b.WriteString(".card{margin:18px 0;padding:18px;border-radius:14px;background:rgba(127,127,127,.07);border:1px solid rgba(127,127,127,.22)}")
	b.WriteString("dl{display:grid;grid-template-columns:max-content 1fr;gap:6px 16px;margin:0}dt{font-weight:600;opacity:.75}dd{margin:0;word-break:break-word}")
	b.WriteString("pre{margin:0;white-space:pre-wrap;overflow-wrap:anywhere;font:13px/1.5 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}")
	b.WriteString(".muted{opacity:.7}")
	b.WriteString("</style></head><body><main>")
	b.WriteString("<header><p class=\"eyebrow\">cozy-mug · dev error</p><h1><span class=\"status\">")
	b.WriteString(code)
	b.WriteString("</span> ")
	b.WriteString(title)
	b.WriteString("</h1><p class=\"summary\"><strong>")
	b.WriteString(method)
	b.WriteString("</strong> <code>")
	b.WriteString(path)
	b.WriteString("</code> <span class=\"muted\">request id: ")
	b.WriteString(id)
	b.WriteString("</span></p></header>")
	b.WriteString("<section class=\"card\"><h2>Exception</h2>")
	if errText != "" {
		b.WriteString("<pre>")
		b.WriteString(errText)
		b.WriteString("</pre>")
	} else {
		b.WriteString("<p class=\"muted\">No error detail attached.</p>")
	}
	b.WriteString("</section><section class=\"card\"><h2>Request</h2><dl>")
	b.WriteString("<dt>Method</dt><dd>")
	b.WriteString(method)
	b.WriteString("</dd><dt>Path</dt><dd>")
	b.WriteString(path)
	b.WriteString("</dd><dt>Status</dt><dd>")
	b.WriteString(code)
	b.WriteString(" ")
	b.WriteString(title)
	b.WriteString("</dd><dt>Request ID</dt><dd>")
	b.WriteString(id)
	b.WriteString("</dd></dl></section>")
	if stack != "" {
		b.WriteString("<section class=\"card\"><h2>Stack trace</h2><pre>")
		b.WriteString(stack)
		b.WriteString("</pre></section>")
	}
	b.WriteString("<p class=\"muted\">Dev only: this page and its detail are not shown in production. Switch <code>Dev</code> to false (or <code>APP_ENV=production</code>) to disable it.</p>")
	b.WriteString("</main></body></html>")
	return b.String()
}
