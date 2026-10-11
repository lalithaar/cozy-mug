package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cozymug/core"
	"cozymug/router"
)

func TestContextReverseURL(t *testing.T) {
	rt := router.New()
	rt.Get("/users/{id}", ok).As("user")
	rt.Get("/", func(c *core.Context) core.Response {
		link, err := c.URL("user", map[string]string{"id": "7"})
		if err != nil {
			t.Fatal(err)
		}
		return core.String(http.StatusOK, `<a href="`+link+`">u</a>`)
	})

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !strings.Contains(rec.Body.String(), `href="/users/7"`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}
