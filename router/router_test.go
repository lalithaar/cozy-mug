package router_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"cozymug/core"
	"cozymug/router"
)

func ok(*core.Context) core.Response { return core.Status(http.StatusOK) }

// End-to-end: dispatch through the router, capture params, render.
func TestRouteParamAndRender(t *testing.T) {
	rt := router.New()
	rt.Get("/hello/{name}", func(c *core.Context) core.Response {
		return core.String(http.StatusOK, "<h1>Hello "+c.Param("name")+"</h1>")
	})

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello/Ada", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "<h1>Hello Ada</h1>" {
		t.Fatalf("body = %q", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" {
		t.Fatal("missing Content-Type")
	}
}

// The seam: a handler is a plain function, callable with no server, no writer.
func TestHandlerCalledDirectly(t *testing.T) {
	var h core.Handler = func(*core.Context) core.Response {
		return core.String(http.StatusOK, "hi")
	}

	resp := h(&core.Context{}) // no dispatch, no ResponseWriter
	if resp == nil {
		t.Fatal("nil response")
	}
	rec := httptest.NewRecorder()
	resp.WriteTo(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Body.String() != "hi" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

// Finding: params live on the request and are installed by ServeMux dispatch,
// so a directly-invoked handler sees none. This is the gap to decide on.
func TestParamNeedsRouterDispatch(t *testing.T) {
	c := &core.Context{R: httptest.NewRequest(http.MethodGet, "/hello/Ada", nil)}
	if got := c.Param("name"); got != "" {
		t.Fatalf("expected empty param outside dispatch, got %q", got)
	}
}

func TestRedirect(t *testing.T) {
	rt := router.New()
	rt.Post("/old", func(*core.Context) core.Response {
		return core.Redirect(http.StatusSeeOther, "/new")
	})

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/old", nil))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/new" {
		t.Fatalf("Location = %q", loc)
	}
}

func TestNotFoundFallback(t *testing.T) {
	rt := router.New()
	rt.NotFound(func(*core.Context) core.Response {
		return core.Error(http.StatusNotFound, "no such page")
	})

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAnyMatchesAllMethods(t *testing.T) {
	rt := router.New()
	rt.Any("/ping", func(*core.Context) core.Response {
		return core.Text(http.StatusOK, "pong")
	})

	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		rec := httptest.NewRecorder()
		rt.ServeHTTP(rec, httptest.NewRequest(m, "/ping", nil))
		if rec.Body.String() != "pong" {
			t.Fatalf("%s body = %q", m, rec.Body.String())
		}
	}
}

func TestStaticServesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	rt := router.New()
	rt.Static("/assets/", dir)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.css", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "body{}" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestNamedRouteReverse(t *testing.T) {
	rt := router.New()
	rt.Get("/users/{id}", ok).As("user")

	got, err := rt.Path("user", map[string]string{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/users/42" {
		t.Fatalf("path = %q, want /users/42", got)
	}
}

func TestReverseEscapesAndWildcard(t *testing.T) {
	rt := router.New()
	rt.Get("/users/{id}", ok).As("user")
	rt.Get("/files/{path...}", ok).As("file")

	if got, _ := rt.Path("user", map[string]string{"id": "a b"}); got != "/users/a%20b" {
		t.Fatalf("segment escape = %q", got)
	}
	if got, _ := rt.Path("file", map[string]string{"path": "a b/c"}); got != "/files/a%20b/c" {
		t.Fatalf("wildcard escape = %q", got)
	}
}

func TestReverseErrors(t *testing.T) {
	rt := router.New()
	rt.Get("/users/{id}", ok).As("user")

	if _, err := rt.Path("missing", nil); err == nil {
		t.Fatal("unknown route: want error")
	}
	if _, err := rt.Path("user", nil); err == nil {
		t.Fatal("missing param: want error")
	}
}

func TestRoutesIntrospection(t *testing.T) {
	rt := router.New()
	rt.Get("/a", ok).As("a")
	rt.Post("/b", ok)

	routes := rt.Routes()
	if len(routes) != 2 {
		t.Fatalf("routes = %d, want 2", len(routes))
	}
	if routes[0].Method() != http.MethodGet || routes[0].Name() != "a" || routes[0].Pattern() != "/a" {
		t.Fatalf("unexpected route metadata: %s %s %s", routes[0].Method(), routes[0].Pattern(), routes[0].Name())
	}
}
