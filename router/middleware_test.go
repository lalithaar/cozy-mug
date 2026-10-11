package router_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"cozymug/core"
	"cozymug/router"
)

func tagMW(tag string, log *[]string) core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(c *core.Context) core.Response {
			*log = append(*log, tag+":in")
			resp := next(c)
			*log = append(*log, tag+":out")
			return resp
		}
	}
}

func TestMiddlewareOrder(t *testing.T) {
	rt := router.New()
	var order []string
	rt.Use(tagMW("a", &order), tagMW("b", &order))
	rt.Get("/x", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	want := []string{"a:in", "b:in", "b:out", "a:out"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestGroupPrefixAndMiddleware(t *testing.T) {
	rt := router.New()
	var hit bool
	auth := func(next core.Handler) core.Handler {
		return func(c *core.Context) core.Response {
			hit = true
			return next(c)
		}
	}

	admin := rt.Group("/admin", auth)
	admin.Get("/users", ok).As("admin.users")

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !hit {
		t.Fatal("group middleware did not run")
	}
	if got, err := rt.Path("admin.users", nil); err != nil || got != "/admin/users" {
		t.Fatalf("path = %q, err = %v", got, err)
	}
}

func TestNestedGroupInheritsMiddleware(t *testing.T) {
	rt := router.New()
	var seen []string

	api := rt.Group("/api", tagMW("api", &seen))
	v1 := api.Group("/v1", tagMW("v1", &seen))
	v1.Get("/ping", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	want := []string{"api:in", "v1:in", "v1:out", "api:out"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("order = %v, want %v", seen, want)
	}
}
