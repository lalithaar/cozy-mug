package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lalithaar/cozy-mug/router"
)

func TestCanonicalSlashRedirectsGET(t *testing.T) {
	rt := router.New()
	rt.Get("/users", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users/", nil))

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/users" {
		t.Fatalf("Location = %q, want /users", loc)
	}
}

func TestCanonicalSlashPreservesMethod(t *testing.T) {
	rt := router.New()
	rt.Post("/users", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users/", nil))

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want 308", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/users" {
		t.Fatalf("Location = %q, want /users", loc)
	}
}

func TestCanonicalSlashKeepsQuery(t *testing.T) {
	rt := router.New()
	rt.Get("/users", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users/?page=2", nil))

	if loc := rec.Header().Get("Location"); loc != "/users?page=2" {
		t.Fatalf("Location = %q, want /users?page=2", loc)
	}
}

func TestCanonicalSlashExemptsSubtree(t *testing.T) {
	rt := router.New()
	rt.Get("/docs/", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no redirect)", rec.Code)
	}
}

func TestCanonicalSlashRootUntouched(t *testing.T) {
	rt := router.New()
	rt.Get("/", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCanonicalSlashDisable(t *testing.T) {
	rt := router.New()
	rt.DisableCanonicalSlash()
	rt.Get("/users", ok)

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users/", nil))

	if rec.Code == http.StatusMovedPermanently {
		t.Fatal("still redirecting after disable")
	}
}
