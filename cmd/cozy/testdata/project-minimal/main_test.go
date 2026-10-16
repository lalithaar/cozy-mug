package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"__MODULE__/handlers"
	"github.com/lalithaar/cozy-mug/app"
)

func TestHomeRenders(t *testing.T) {
	a := app.Must(app.Config{
		Assets:  assets,
		Routes:  handlers.Register,
		Dev:     true,
	})
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("home: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Hello from cozy-mug") {
		t.Fatalf("home body missing greeting:\n%s", rec.Body.String())
	}
}