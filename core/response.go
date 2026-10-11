package core

import (
	"html"
	"io"
	"net/http"
)

// Response is the value a handler returns. Deliberately tiny: this framework
// renders HTML only (scope 0002), so there is no content negotiation here.
type Response interface {
	WriteTo(w http.ResponseWriter, r *http.Request)
}

// StatusCoder is implemented by responses that know their HTTP status, so the
// Logger can record it without capturing the writer. A Response that does not
// implement it is logged as 200.
type StatusCoder interface {
	StatusCode() int
}

// HTML is an HTML response. A zero Status means 200.
type HTML struct {
	Status  int
	Headers http.Header
	Body    string
}

// String builds an HTML response. The body is written verbatim, so it must be
// rendered through html/template (auto-escaping) before reaching here.
func String(status int, body string) HTML {
	return HTML{Status: status, Body: body}
}

// With returns a copy of h with a response header set.
func (h HTML) With(key, value string) HTML {
	out := h
	out.Headers = h.Headers.Clone()
	if out.Headers == nil {
		out.Headers = http.Header{}
	}
	out.Headers.Set(key, value)
	return out
}

func (h HTML) WriteTo(w http.ResponseWriter, r *http.Request) {
	copyHeaders(w, h.Headers)
	setContentType(w, "text/html; charset=utf-8")
	writeStatus(w, h.Status)
	io.WriteString(w, h.Body)
}

func (h HTML) StatusCode() int {
	if h.Status == 0 {
		return http.StatusOK
	}
	return h.Status
}

// Text is a text/plain response.
func Text(status int, body string) Response { return text{status, body} }

type text struct {
	status int
	body   string
}

func (t text) WriteTo(w http.ResponseWriter, _ *http.Request) {
	setContentType(w, "text/plain; charset=utf-8")
	writeStatus(w, t.status)
	io.WriteString(w, t.body)
}

func (t text) StatusCode() int { return t.status }

// Redirect sends the client to url. Use http.StatusSeeOther after a POST and
// http.StatusMovedPermanently for permanent moves.
func Redirect(status int, url string) Response { return redirect{status, url} }

type redirect struct {
	status int
	url    string
}

func (rd redirect) WriteTo(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, rd.url, rd.status)
}

func (rd redirect) StatusCode() int { return rd.status }

// Error is a minimal HTML error response. The message is escaped; styling is
// the UI layer's job (0003), so the default is intentionally plain.
func Error(code int, msg string) Response {
	return String(code, "<!doctype html><title>"+statusText(code)+"</title><h1>"+html.EscapeString(msg)+"</h1>")
}

// Status is an empty response with the given code (e.g. 204 No Content).
func Status(code int) Response { return empty{code} }

type empty struct{ code int }

func (e empty) WriteTo(w http.ResponseWriter, _ *http.Request) { writeStatus(w, e.code) }

func (e empty) StatusCode() int { return e.code }

func copyHeaders(w http.ResponseWriter, h http.Header) {
	for k, vs := range h {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
}

func setContentType(w http.ResponseWriter, ct string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", ct)
	}
}

func writeStatus(w http.ResponseWriter, code int) {
	if code != 0 {
		w.WriteHeader(code)
	}
}

func statusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return "Error"
}
