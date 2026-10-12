package app_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
)

func assets() fs.FS {
	return fstest.MapFS{
		"views/layouts/base.html": {Data: []byte(`{{define "base"}}<title>{{block "title" .}}app{{end}}</title>{{block "content" .}}{{end}}{{end}}`)},
		"views/home.html":         {Data: []byte(`{{define "title"}}Home{{end}}{{define "content"}}<h1>Hello</h1>{{end}}`)},
		"static/app.css":          {Data: []byte("body{}")},
	}
}

func TestWiresRouteAndStatic(t *testing.T) {
	a, err := app.New(app.Config{
		Assets: assets(),
		Routes: func(r *router.Router) {
			r.Get("/", func(c *core.Context) core.Response {
				return c.Render(http.StatusOK, "home", nil)
			}).As("home")
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Hello") {
		t.Fatalf("home: code=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("static: code=%d", rec.Code)
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing: code=%d, want 404", rec.Code)
	}
}

func TestUIMountsStylesheet(t *testing.T) {
	a, err := app.New(app.Config{Assets: assets(), UI: true})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/ui/base.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("ui stylesheet: code=%d, want 200", rec.Code)
	}
}

func TestUIFuncsReachTemplates(t *testing.T) {
	fsys := fstest.MapFS{
		"views/layouts/base.html": {Data: []byte(`{{define "base"}}{{block "content" .}}{{end}}{{end}}`)},
		"views/probe.html":        {Data: []byte(`{{define "content"}}{{ $p := props "k" "v" }}{{index $p "k"}}{{end}}`)},
	}
	a, err := app.New(app.Config{
		Assets: fsys,
		UI:     true,
		Routes: func(r *router.Router) {
			r.Get("/", func(c *core.Context) core.Response {
				return c.Render(http.StatusOK, "probe", nil)
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "v" {
		t.Fatalf("probe: code=%d body=%q, want %q", rec.Code, rec.Body.String(), "v")
	}
}

func TestMissingViewsErrors(t *testing.T) {
	if _, err := app.New(app.Config{Assets: fstest.MapFS{}}); err == nil {
		t.Fatal("expected error when views/ is absent")
	}
}
