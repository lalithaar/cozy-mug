package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lalithaar/cozy-mug/router"
)

func TestCanonicalSlashNoOpenRedirect(t *testing.T) {
	rt := router.New()
	rt.Get("/users", ok)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.Path = "//evil.example/"

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)

	loc := rec.Header().Get("Location")
	if strings.HasPrefix(loc, "//") {
		t.Fatalf("protocol-relative open redirect: Location = %q", loc)
	}
}
