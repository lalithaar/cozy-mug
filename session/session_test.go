package session

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
)

func testStore(t *testing.T, opts Options) *Store {
	t.Helper()
	if opts.Key == nil {
		opts.Key = []byte("0123456789abcdef0123456789abcdef")
	}
	s, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

// do runs req through the session middleware and a no-op handler, returning the
// response status and any cookies set.
func do(s *Store, req *http.Request, fn func(c *core.Context)) (int, []*http.Cookie) {
	rec := httptest.NewRecorder()
	h := s.Middleware(func(c *core.Context) core.Response {
		if fn != nil {
			fn(c)
		}
		return nil
	})
	resp := h(&core.Context{W: rec, R: req})
	code := http.StatusOK
	if sc, ok := resp.(interface{ StatusCode() int }); ok {
		code = sc.StatusCode()
	}
	return code, rec.Result().Cookies()
}

func TestNewRejectsBadKey(t *testing.T) {
	if _, err := New(Options{Key: []byte("short")}); err == nil {
		t.Fatal("expected error for short key")
	}
	if _, err := New(Options{Key: make([]byte, 32)}); err != nil {
		t.Fatalf("32-byte key should be accepted: %v", err)
	}
}

func TestKeyFromEnv(t *testing.T) {
	if _, err := Key("SESSION_KEY"); err == nil {
		t.Fatal("unset key should error")
	}
	t.Setenv("SESSION_KEY", "!!!not base64!!!")
	if _, err := Key("SESSION_KEY"); err == nil {
		t.Fatal("malformed base64 should error")
	}
	t.Setenv("SESSION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 16)))
	if _, err := Key("SESSION_KEY"); err == nil {
		t.Fatal("wrong length should error")
	}
	want := []byte("0123456789abcdef0123456789abcdef")
	t.Setenv("SESSION_KEY", base64.StdEncoding.EncodeToString(want))
	got, err := Key("SESSION_KEY")
	if err != nil || string(got) != string(want) {
		t.Fatalf("Key = %q, %v", got, err)
	}
}

func TestRoundTrip(t *testing.T) {
	s := testStore(t, Options{})

	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("user", "ada")
	})
	if len(cookies) != 1 {
		t.Fatalf("want 1 cookie, got %d", len(cookies))
	}
	if _, err := base64.RawURLEncoding.DecodeString(cookies[0].Value); err != nil {
		t.Fatalf("cookie is not opaque base64: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])
	var got any
	do(s, req, func(c *core.Context) { got = c.Session().Get("user") })
	if got != "ada" {
		t.Fatalf("round-trip lost value: %v", got)
	}
}

func TestUntouchedRequestSetsNoCookie(t *testing.T) {
	s := testStore(t, Options{})
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {})
	if len(cookies) != 0 {
		t.Fatalf("untouched request set a cookie: %v", cookies)
	}
}

func TestTamperedCookieIsDiscarded(t *testing.T) {
	s := testStore(t, Options{})
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("user", "ada")
	})
	bad := cookies[0].Value
	b := []byte(bad)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: DefaultName, Value: string(b)})
	var got any
	do(s, req, func(c *core.Context) { got = c.Session().Get("user") })
	if got != nil {
		t.Fatalf("tampered cookie should yield an empty session, got %v", got)
	}
}

func TestCookieAttributes(t *testing.T) {
	s := testStore(t, Options{Secure: true})
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("k", "v")
	})
	ck := cookies[0]
	if !ck.HttpOnly || !ck.Secure || ck.Path != "/" || ck.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie attributes wrong: %+v", ck)
	}
}

func TestCSRF(t *testing.T) {
	s := testStore(t, Options{})

	var tok string
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		tok = c.Session().CSRFToken()
	})
	if tok == "" || len(cookies) != 1 {
		t.Fatalf("GET did not mint a token/cookie: %q %v", tok, cookies)
	}

	// No token at all.
	if code, _ := do(s, httptest.NewRequest("POST", "/", nil), func(c *core.Context) {}); code != 403 {
		t.Fatalf("POST without token: code %d, want 403", code)
	}

	// Cookie present, wrong token.
	wrong := httptest.NewRequest("POST", "/", strings.NewReader("csrf=wrong"))
	wrong.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wrong.AddCookie(cookies[0])
	if code, _ := do(s, wrong, func(c *core.Context) {}); code != 403 {
		t.Fatalf("POST with wrong token: code %d, want 403", code)
	}

	// Correct token in the form field.
	form := httptest.NewRequest("POST", "/", strings.NewReader("csrf="+tok))
	form.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form.AddCookie(cookies[0])
	if code, _ := do(s, form, func(c *core.Context) {}); code != 200 {
		t.Fatalf("POST with token: code %d, want 200", code)
	}

	// Correct token in the header.
	head := httptest.NewRequest("POST", "/", nil)
	head.Header.Set("X-CSRF-Token", tok)
	head.AddCookie(cookies[0])
	if code, _ := do(s, head, func(c *core.Context) {}); code != 200 {
		t.Fatalf("POST with header token: code %d, want 200", code)
	}

	// Safe methods never need a token.
	if code, _ := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {}); code != http.StatusOK {
		t.Fatalf("GET should not be blocked, got %d", code)
	}
}

func TestDisableCSRF(t *testing.T) {
	s := testStore(t, Options{DisableCSRF: true})
	if code, _ := do(s, httptest.NewRequest("POST", "/", nil), func(c *core.Context) {}); code != http.StatusOK {
		t.Fatalf("CSRF disabled, POST should reach the handler, code %d", code)
	}
}

func TestDestroyClearsCookie(t *testing.T) {
	s := testStore(t, Options{})
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("k", "v")
	})
	if len(cookies) != 1 {
		t.Fatalf("setup: want 1 cookie, got %d", len(cookies))
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])
	_, after := do(s, req, func(c *core.Context) { c.Session().Destroy() })
	if len(after) != 1 || after[0].MaxAge >= 0 || after[0].Value != "" {
		t.Fatalf("destroy did not clear the cookie: %+v", after)
	}
}

func TestRegenerate(t *testing.T) {
	sess := &Session{id: "old", data: map[string]any{"user": "ada"}}
	sess.Regenerate()
	if sess.ID() == "old" {
		t.Fatal("Regenerate kept the old id")
	}
	if len(sess.data) != 0 {
		t.Fatal("Regenerate kept data")
	}
	if !sess.Dirty() {
		t.Fatal("Regenerate should mark the session dirty")
	}
}

func TestOversizeSessionNotPersisted(t *testing.T) {
	s := testStore(t, Options{})
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("k", strings.Repeat("x", 5000))
	})
	if len(cookies) != 0 {
		t.Fatalf("oversize session set a cookie: %v", cookies)
	}
}

// --- database store -----------------------------------------------------------

func testDBStore(t *testing.T, opts Options) (*Store, *data.DB) {
	t.Helper()
	ctx := context.Background()
	db, err := data.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := NewStore(db, opts)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return s, db
}

func countSessions(t *testing.T, db *data.DB) int {
	t.Helper()
	rows, err := data.All[sessionRow](context.Background(), db,
		`select id, data, created_at, expires_at from sessions`)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return len(rows)
}

func TestNewStoreRejectsNilDB(t *testing.T) {
	if _, err := NewStore(nil, Options{}); err == nil {
		t.Fatal("NewStore(nil) should error")
	}
}

func TestDBStoreRoundTrip(t *testing.T) {
	s, _ := testDBStore(t, Options{})

	var id string
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("user", "ada")
		id = From(c).ID()
	})
	if len(cookies) != 1 {
		t.Fatalf("want 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Value != id {
		t.Fatalf("cookie should carry the opaque session id: got %q want %q", cookies[0].Value, id)
	}
	if strings.Contains(cookies[0].Value, "ada") {
		t.Fatal("session data leaked into the cookie")
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])
	var got any
	do(s, req, func(c *core.Context) { got = c.Session().Get("user") })
	if got != "ada" {
		t.Fatalf("round-trip lost value: %v", got)
	}
}

func TestDBStoreUnknownIDIsEmpty(t *testing.T) {
	s, _ := testDBStore(t, Options{})
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: DefaultName, Value: "no-such-session"})
	var got any
	do(s, req, func(c *core.Context) { got = c.Session().Get("user") })
	if got != nil {
		t.Fatalf("unknown id should yield an empty session, got %v", got)
	}
}

func TestDBStoreDestroyDeletesRow(t *testing.T) {
	s, db := testDBStore(t, Options{})
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		c.Session().Set("k", "v")
	})
	if countSessions(t, db) != 1 {
		t.Fatalf("setup: want 1 row, got %d", countSessions(t, db))
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])
	_, after := do(s, req, func(c *core.Context) { c.Session().Destroy() })
	if len(after) != 1 || after[0].MaxAge >= 0 || after[0].Value != "" {
		t.Fatalf("destroy did not clear the cookie: %+v", after)
	}
	if countSessions(t, db) != 0 {
		t.Fatal("destroy left a session row behind")
	}
}

func TestDBStoreExpiryAndSweep(t *testing.T) {
	ctx := context.Background()
	s, db := testDBStore(t, Options{})
	past := time.Now().Add(-time.Hour).Unix()
	if _, err := data.Exec(ctx, db,
		`insert into sessions (id, data, created_at, expires_at) values (?, ?, ?, ?)`,
		"expired", `{"user":"ada"}`, past, past); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: DefaultName, Value: "expired"})
	var got any
	do(s, req, func(c *core.Context) { got = c.Session().Get("user") })
	if got != nil {
		t.Fatalf("expired session should be dead, got %v", got)
	}

	if err := s.Sweep(ctx); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if countSessions(t, db) != 0 {
		t.Fatal("Sweep left expired rows")
	}
}

func TestDBStoreAbsoluteCap(t *testing.T) {
	ctx := context.Background()
	s, db := testDBStore(t, Options{Absolute: time.Minute})
	now := time.Now().Unix()
	if _, err := data.Exec(ctx, db,
		`insert into sessions (id, data, created_at, expires_at) values (?, ?, ?, ?)`,
		"capped", `{"user":"ada"}`, now-3600, now+3600); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: DefaultName, Value: "capped"})
	var got any
	do(s, req, func(c *core.Context) { got = c.Session().Get("user") })
	if got != nil {
		t.Fatalf("session past its absolute cap should be dead, got %v", got)
	}
}

func TestDBStoreCSRF(t *testing.T) {
	s, _ := testDBStore(t, Options{})

	var tok string
	_, cookies := do(s, httptest.NewRequest("GET", "/", nil), func(c *core.Context) {
		tok = c.Session().CSRFToken()
	})
	if tok == "" || len(cookies) != 1 {
		t.Fatalf("GET did not mint a token/cookie: %q %v", tok, cookies)
	}

	if code, _ := do(s, httptest.NewRequest("POST", "/", nil), func(c *core.Context) {}); code != 403 {
		t.Fatalf("POST without token: code %d, want 403", code)
	}

	form := httptest.NewRequest("POST", "/", strings.NewReader("csrf="+tok))
	form.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form.AddCookie(cookies[0])
	if code, _ := do(s, form, func(c *core.Context) {}); code != 200 {
		t.Fatalf("POST with token: code %d, want 200", code)
	}
}

func TestGetInt64(t *testing.T) {
	// JSON decode yields float64; GetInt64 must cope.
	sess := &Session{data: map[string]any{"n": float64(42)}}
	if sess.GetInt64("n") != 42 {
		t.Fatalf("GetInt64 = %d", sess.GetInt64("n"))
	}
	if sess.GetString("n") != "" {
		t.Fatal("GetString on a number should be empty")
	}
}
