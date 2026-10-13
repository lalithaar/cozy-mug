package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/examples/app/handlers"
	"github.com/lalithaar/cozy-mug/funcs"
	"github.com/lalithaar/cozy-mug/session"
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
	stmts, err := db.Table[handlers.Note]().SchemaSQL()
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}
	if err := store.Seed(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	sessions, err := session.NewStore(db, session.Options{})
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if err := sessions.EnsureSchema(ctx); err != nil {
		t.Fatalf("session schema: %v", err)
	}

	return app.Must(app.Config{
		Assets:     assets,
		Routes:     store.Register,
		UI:         true,
		Funcs:      funcs.Funcs(),
		Session:    sessions,
		Processors: handlers.Processors(),
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

// csrfToken pulls the token out of the hidden input the home form renders.
var csrfToken = regexp.MustCompile(`name="csrf" value="([^"]+)"`)

func TestCreateNoteWithCSRF(t *testing.T) {
	a := newApp(t)

	// A GET renders the form, which mints a token and sets the session cookie.
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("home: status %d", rec.Code)
	}
	m := csrfToken.FindStringSubmatch(rec.Body.String())
	if m == nil {
		t.Fatalf("home form missing CSRF input:\n%s", rec.Body.String())
	}
	token := m[1]
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("home did not set a session cookie")
	}

	// A POST with no token is rejected before the handler runs.
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader("title=nope")))
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST without token: status %d, want 403", rec.Code)
	}

	// A POST with the session cookie but a wrong token is also rejected.
	bad := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader("title=nope&csrf=wrong"))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	bad.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, bad)
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST with wrong token: status %d, want 403", rec.Code)
	}

	// The real token posts, inserts, flashes, and redirects. The response carries
	// the updated session cookie (holding the flash); use it for the next hop, as
	// a browser would.
	form := url.Values{"title": {"Third note"}, "body": {"added via form"}, "csrf": {token}}
	good := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(form.Encode()))
	good.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	good.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, good)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST with token: status %d, want 303", rec.Code)
	}
	cookies = rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("POST did not set the flash cookie")
	}

	// Following the redirect carries the flash once and shows the new note.
	follow := httptest.NewRequest(http.MethodGet, "/", nil)
	follow.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, follow)
	body := rec.Body.String()
	if !strings.Contains(body, "Note added.") {
		t.Errorf("flash missing after redirect:\n%s", body)
	}
	if !strings.Contains(body, "Third note") {
		t.Errorf("new note missing:\n%s", body)
	}
}
