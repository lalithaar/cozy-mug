package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/examples/app/handlers"
	"github.com/lalithaar/cozy-mug/funcs"
)

func newApp(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()

	db, err := data.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := handlers.NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("schema: %v", err)
	}

	return app.Must(app.Config{
		Assets: assets,
		Routes: store.Register,
		UI:     true,
		Funcs:  funcs.Funcs(),
	})
}

func TestExampleRenders(t *testing.T) {
	a := newApp(t)

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
