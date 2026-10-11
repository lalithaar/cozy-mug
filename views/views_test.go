package views_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"cozymug/core"
	"cozymug/router"
	"cozymug/views"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"layouts/base.html": {Data: []byte(`{{define "base"}}<title>{{block "title" .}}t{{end}}</title><main>{{block "content" .}}{{end}}</main>{{end}}`)},
		"partials/nav.html": {Data: []byte(`{{define "nav"}}<nav>n</nav>{{end}}`)},
		"home.html":         {Data: []byte(`{{define "title"}}Home{{end}}{{define "content"}}hi {{.Name}} {{template "nav" .}}{{end}}`)},
	}
}

func TestRenderLayoutAndPartial(t *testing.T) {
	set, err := views.New(testFS(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := set.Render("home", struct{ Name string }{"Al"})
	if err != nil {
		t.Fatal(err)
	}
	want := `<title>Home</title><main>hi Al <nav>n</nav></main>`
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRenderMissingViewErrors(t *testing.T) {
	set, err := views.New(testFS(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Render("nope", nil); err == nil {
		t.Fatal("expected error for missing view")
	}
}

func TestNewWithoutLayoutFailsFast(t *testing.T) {
	fsys := fstest.MapFS{"home.html": {Data: []byte(`{{define "content"}}x{{end}}`)}}
	if _, err := views.New(fsys, nil); err == nil {
		t.Fatal("expected error when layouts/ has no base")
	}
}

func TestContextRenderInjectsGlobalsURL(t *testing.T) {
	fsys := fstest.MapFS{
		"layouts/base.html": {Data: []byte(`{{define "base"}}<a href="{{.URL "user" "id" "7"}}">u</a>{{end}}`)},
		"home.html":         {Data: []byte(`{{define "content"}}x{{end}}`)},
	}
	set, err := views.New(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}

	r := router.New()
	r.Use(views.Middleware(set))
	r.Get("/users/{id}", func(*core.Context) core.Response { return core.String(200, "u") }).As("user")
	r.Get("/", func(c *core.Context) core.Response {
		return c.Render(http.StatusOK, "home", &struct{ core.Globals }{})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if body := rec.Body.String(); !strings.Contains(body, `href="/users/7"`) {
		t.Fatalf("body = %q", body)
	}
}

func TestRenderErrorIsComplete500(t *testing.T) {
	fsys := fstest.MapFS{
		"layouts/base.html": {Data: []byte(`{{define "base"}}{{.URL "missing"}}{{end}}`)},
		"home.html":         {Data: []byte(`{{define "content"}}x{{end}}`)},
	}
	set, err := views.New(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := router.New()
	r.Use(views.Middleware(set))
	r.Get("/", func(c *core.Context) core.Response {
		return c.Render(http.StatusOK, "home", &struct{ core.Globals }{})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", rec.Code)
	}
}
