package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
)

func TestDevReloadPlumbing(t *testing.T) {
	fsys := fstest.MapFS{
		"views/layouts/base.html": {Data: []byte(`{{define "base"}}<!doctype html><html><body>{{block "content" .}}{{end}}</body></html>{{end}}`)},
		"views/home.html":         {Data: []byte(`{{define "content"}}<h1>Home</h1>{{end}}`)},
	}

	a, err := app.New(app.Config{
		Assets: fsys,
		Dev:    true,
		Routes: func(r *router.Router) {
			r.Get("/", func(c *core.Context) core.Response {
				return c.Render(http.StatusOK, "home", nil)
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. HTML injection
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<script src="/__cozy/dev-reload"></script></body>`) {
		t.Fatalf("body missing dev reload script injection before </body>: %s", body)
	}

	// 2. /__cozy/dev-reload script content
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/__cozy/dev-reload", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dev-reload status = %d, want 200", rec.Code)
	}
	script := rec.Body.String()
	if !strings.Contains(script, `sessionStorage.setItem(key, data.build)`) {
		t.Fatalf("dev-reload script missing sessionStorage update: %s", script)
	}
	// Verify that setItem happens before location.reload to prevent infinite loops.
	setIdx := strings.Index(script, `sessionStorage.setItem(key, data.build)`)
	reloadIdx := strings.Index(script, `location.reload()`)
	if setIdx == -1 || reloadIdx == -1 || setIdx > reloadIdx {
		t.Fatalf("sessionStorage update must occur before location.reload: setIdx=%d, reloadIdx=%d", setIdx, reloadIdx)
	}

	// 3. /__cozy/dev-build response
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/__cozy/dev-build", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dev-build status = %d, want 200", rec.Code)
	}
	var res struct {
		Build string `json:"build"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode build json: %v", err)
	}
	if res.Build == "" {
		t.Fatal("expected non-empty build id")
	}
}
