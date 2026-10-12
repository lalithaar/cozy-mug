package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/examples/app/handlers"
	"github.com/lalithaar/cozy-mug/funcs"
)

func TestExampleRenders(t *testing.T) {
	a := app.Must(app.Config{
		Assets: assets,
		Routes: handlers.Register,
		UI:     true,
		Funcs:  funcs.Funcs(),
	})

	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("home: status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="odd"`) || !strings.Contains(body, "1.") {
		t.Errorf("home missing loop-func output:\n%s", body)
	}
	if !strings.Contains(body, "2 notes") {
		t.Errorf("home missing pluralize output:\n%s", body)
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/notes/1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("note: status %d", rec.Code)
	}
	body = rec.Body.String()
	if !strings.Contains(body, `name="note-id"`) || !strings.Contains(body, "/static/style.css") {
		t.Errorf("note head did not extend the base (super):\n%s", body)
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/notes/2", nil))
	if body := rec.Body.String(); !strings.Contains(body, "<br>") {
		t.Errorf("note body missing linebreaksbr output:\n%s", body)
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/notes/999", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing note: status %d, want 404", rec.Code)
	}
}
