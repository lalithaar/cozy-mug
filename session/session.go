// Package session provides sessions and CSRF protection for cozy-mug apps
// (ADR 0013).
//
// The default store keeps session data **server-side** in a `sessions` table
// (via the data package): the browser holds only an opaque 256-bit id, so a
// session can be revoked (delete the row) and there is no global secret whose
// leak forges every session. A sealed-cookie store — data encrypted into the
// cookie itself, no server state — is available as an escape hatch for apps that
// cannot or will not use the database.
//
// CSRF is on by default: the token lives in the session (server-side for the DB
// store) and must be echoed on unsafe methods.
package session

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
)

// DefaultName is the cookie name used when Options.Name is empty.
const DefaultName = "cozy_session"

// csrfSlot is the session key holding the CSRF token.
const csrfSlot = "csrf"

// maxCookieValue keeps the cookie (name + value) well under the 4 KB limit.
const maxCookieValue = 3600

// defaultMaxAge is the session lifetime when Options.MaxAge is zero.
const defaultMaxAge = 30 * 24 * time.Hour

// Options configures a Store.
type Options struct {
	// Name is the cookie name; DefaultName when empty.
	Name string

	// Secure sets the cookie's Secure attribute. Turn it on in production (HTTPS).
	Secure bool

	// SameSite defaults to Lax when zero.
	SameSite http.SameSite

	// MaxAge is the session lifetime, refreshed on each change. Zero uses
	// defaultMaxAge (30 days).
	MaxAge time.Duration

	// Absolute caps the total lifetime from creation, regardless of activity.
	// Zero means no cap.
	Absolute time.Duration

	// DisableCSRF turns off CSRF enforcement in Middleware (sessions still work).
	DisableCSRF bool

	// Key seals the cookie for the cookie store (New). It must be exactly 32
	// bytes and is ignored by the database store (NewStore).
	Key []byte
}

// Store loads, saves, and expires sessions, and enforces CSRF. It is safe for
// concurrent use.
type Store struct {
	backend     backend
	name        string
	secure      bool
	sameSite    http.SameSite
	maxAge      time.Duration
	absolute    time.Duration
	enforceCSRF bool
}

// NewStore returns a database-backed Store: session data lives in a `sessions`
// table and the cookie holds only an opaque id. This is the recommended store.
// Call EnsureSchema once at boot, and Sweep periodically to drop expired rows.
func NewStore(db *data.DB, opts Options) (*Store, error) {
	if db == nil {
		return nil, errors.New("session: NewStore requires a non-nil *data.DB")
	}
	return newStore(&dbBackend{db: db}, opts), nil
}

// New returns a cookie-backed Store: the session is sealed with AES-256-GCM into
// the cookie itself, with no server state. It needs no database but cannot revoke
// a session; prefer NewStore unless you specifically want stateless cookies.
func New(opts Options) (*Store, error) {
	if len(opts.Key) != 32 {
		return nil, fmt.Errorf("session: Options.Key must be 32 bytes, got %d", len(opts.Key))
	}
	block, err := aes.NewCipher(opts.Key)
	if err != nil {
		return nil, fmt.Errorf("session: cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("session: gcm: %w", err)
	}
	return newStore(&cookieBackend{aead: aead}, Options{
		Name: opts.Name, Secure: opts.Secure, SameSite: opts.SameSite,
		MaxAge: opts.MaxAge, Absolute: opts.Absolute, DisableCSRF: opts.DisableCSRF,
	}), nil
}

func newStore(b backend, opts Options) *Store {
	name := opts.Name
	if name == "" {
		name = DefaultName
	}
	sameSite := opts.SameSite
	if sameSite == 0 {
		sameSite = http.SameSiteLaxMode
	}
	maxAge := opts.MaxAge
	if maxAge == 0 {
		maxAge = defaultMaxAge
	}
	return &Store{
		backend:     b,
		name:        name,
		secure:      opts.Secure,
		sameSite:    sameSite,
		maxAge:      maxAge,
		absolute:    opts.Absolute,
		enforceCSRF: !opts.DisableCSRF,
	}
}

// Key decodes a base64, 32-byte key from the named environment variable, for the
// cookie store. A missing or malformed key is an error; the value is never echoed
// (secrets heuristic). Generate one with: openssl rand -base64 32.
func Key(env string) ([]byte, error) {
	v := os.Getenv(env)
	if v == "" {
		return nil, fmt.Errorf("session: %s is not set", env)
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("session: %s is not valid base64", env)
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("session: %s must decode to 32 bytes, got %d", env, len(b))
	}
	return b, nil
}

// EnsureSchema creates the sessions table (database store). It is idempotent;
// call it once at boot, like the app's other schema setup.
func (s *Store) EnsureSchema(ctx context.Context) error { return s.backend.ensureSchema(ctx) }

// Sweep deletes expired rows. Call it periodically (the DB store only). It is a
// no-op for the cookie store.
func (s *Store) Sweep(ctx context.Context) error { return s.backend.sweep(ctx) }

// Middleware loads the session, runs the handler, then persists any change.
// Install it inside the Errors middleware and before the views so templates can
// read the session.
//
// It does not enforce CSRF: that is CSRFMiddleware, installed separately after
// any value processors so a rejected unsafe request still renders its error page
// with the global values (ADR 0014).
func (s *Store) Middleware(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		ctx := c.R.Context()
		sess := s.load(ctx, c.R)
		c.R = core.WithSession(c.R, sess)

		resp := next(c)
		if err := s.persist(ctx, c.W, sess); err != nil {
			// A session that cannot be written is not a session. Fail loud.
			return c.Error(err)
		}
		return resp
	}
}

// CSRFMiddleware rejects unsafe methods (POST/PUT/PATCH/DELETE) whose submitted
// token (header X-CSRF-Token, else form field "csrf") does not match the
// session's, in constant time. Install it after the session and any value
// processors but before the views/handler, so a rejection still carries global
// values into the error page. It is a no-op when Options.DisableCSRF was set.
func (s *Store) CSRFMiddleware(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		if !s.enforceCSRF || safeMethod(c.Method()) {
			return next(c)
		}
		sess, ok := c.Session().(*Session)
		if !ok {
			return c.Error(core.Forbidden(errors.New("session: no session to check CSRF against")))
		}
		if err := s.checkCSRF(c.R, sess); err != nil {
			return c.Error(err)
		}
		return next(c)
	}
}

func (s *Store) load(ctx context.Context, r *http.Request) *Session {
	if ck, err := r.Cookie(s.name); err == nil {
		if sess, ok := s.backend.load(s, ctx, ck.Value); ok {
			return sess
		}
	}
	return &Session{id: newID(), data: map[string]any{}}
}

func (s *Store) persist(ctx context.Context, w http.ResponseWriter, sess *Session) error {
	if sess.destroyed {
		if err := s.backend.delete(s, ctx, sess.id); err != nil {
			return err
		}
		s.clearCookie(w)
		return nil
	}
	if !sess.dirty {
		return nil
	}
	if err := s.backend.save(s, ctx, sess); err != nil {
		return err
	}
	value, err := s.backend.value(s, sess)
	if err != nil {
		return err
	}
	if len(value)+len(s.name) > maxCookieValue {
		return nil // too large for a cookie
	}
	s.setCookie(w, value)
	return nil
}

func (s *Store) setCookie(w http.ResponseWriter, value string) {
	maxAge := int(s.maxAge.Seconds())
	http.SetCookie(w, &http.Cookie{
		Name: s.name, Value: value, Path: "/", HttpOnly: true,
		Secure: s.secure, SameSite: s.sameSite, MaxAge: maxAge,
	})
}

func (s *Store) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: s.name, Value: "", Path: "/", HttpOnly: true,
		Secure: s.secure, SameSite: s.sameSite, MaxAge: -1,
	})
}

// checkCSRF compares the submitted token (header X-CSRF-Token, else form field
// "csrf") against the session's token in constant time.
func (s *Store) checkCSRF(r *http.Request, sess *Session) error {
	want, _ := sess.data[csrfSlot].(string)
	got := r.Header.Get("X-CSRF-Token")
	if got == "" {
		got = r.PostFormValue("csrf")
	}
	if want == "" || got == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return core.Forbidden(errors.New("session: CSRF token missing or invalid"))
	}
	return nil
}

// Session is a per-request bag of values with an id and a CSRF token. It
// satisfies core.Session and adds typed getters. It is not safe for concurrent
// use; a request is single-goroutine.
type Session struct {
	id        string
	data      map[string]any
	dirty     bool
	destroyed bool
	createdAt int64
}

// From returns the request's session as the concrete *Session, or nil when the
// app did not configure sessions. It is the typed accessor for the richer
// helpers (GetString, GetInt64, ID) that the core seam does not expose.
func From(c *core.Context) *Session {
	s, _ := c.Session().(*Session)
	return s
}

// ID is the session's identifier (regenerated by Regenerate).
func (s *Session) ID() string { return s.id }

// Get returns the value stored at key, or nil.
func (s *Session) Get(key string) any { return s.data[key] }

// GetString returns the value at key as a string ("" if absent or not a string).
func (s *Session) GetString(key string) string {
	v, _ := s.data[key].(string)
	return v
}

// GetInt64 returns the value at key as an int64, accepting JSON's float64, or 0.
func (s *Session) GetInt64(key string) int64 {
	switch v := s.data[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}

// Set stores a value. Values are JSON-encoded when persisted, so keep them small
// and JSON-friendly.
func (s *Session) Set(key string, value any) {
	s.data[key] = value
	s.dirty = true
}

// Delete removes a key.
func (s *Session) Delete(key string) {
	if _, ok := s.data[key]; ok {
		delete(s.data, key)
		s.dirty = true
	}
}

// Clear removes every key but keeps the session (and its id).
func (s *Session) Clear() {
	s.data = map[string]any{}
	s.dirty = true
}

// Dirty reports whether the session changed and needs to be written back.
func (s *Session) Dirty() bool { return s.dirty }

// Regenerate gives the session a fresh id and empty contents — call it on login
// to prevent session fixation. The CSRF token is regenerated with it.
func (s *Session) Regenerate() {
	s.id = newID()
	s.data = map[string]any{}
	s.dirty = true
	s.createdAt = 0
}

// Destroy empties the session and deletes it (row + cookie).
func (s *Session) Destroy() {
	s.data = map[string]any{}
	s.destroyed = true
	s.dirty = true
}

// CSRFToken returns the CSRF token, creating one on first use. With the database
// store it never leaves the server except in the rendered form.
func (s *Session) CSRFToken() string {
	if t, ok := s.data[csrfSlot].(string); ok && t != "" {
		return t
	}
	t := randToken()
	s.data[csrfSlot] = t
	s.dirty = true
	return t
}

func newID() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("session: rand: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func randToken() string { return newID() }

func safeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

// --- backends -----------------------------------------------------------------

type backend interface {
	load(s *Store, ctx context.Context, value string) (*Session, bool)
	save(s *Store, ctx context.Context, sess *Session) error
	delete(s *Store, ctx context.Context, id string) error
	value(s *Store, sess *Session) (string, error)
	ensureSchema(ctx context.Context) error
	sweep(ctx context.Context) error
}

// cookieBackend seals session data into the cookie — no server state, no
// revocation. The escape hatch.
type cookieBackend struct{ aead cipher.AEAD }

type cookiePayload struct {
	ID   string         `json:"id"`
	Data map[string]any `json:"d"`
}

func (b *cookieBackend) load(_ *Store, _ context.Context, value string) (*Session, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, false
	}
	ns := b.aead.NonceSize()
	if len(raw) < ns {
		return nil, false
	}
	plain, err := b.aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return nil, false
	}
	var p cookiePayload
	if err := json.Unmarshal(plain, &p); err != nil {
		return nil, false
	}
	if p.Data == nil {
		p.Data = map[string]any{}
	}
	return &Session{id: p.ID, data: p.Data}, true
}

func (b *cookieBackend) save(_ *Store, _ context.Context, _ *Session) error { return nil }
func (b *cookieBackend) delete(_ *Store, _ context.Context, _ string) error { return nil }
func (b *cookieBackend) ensureSchema(context.Context) error                 { return nil }
func (b *cookieBackend) sweep(context.Context) error                        { return nil }

func (b *cookieBackend) value(_ *Store, sess *Session) (string, error) {
	raw, err := json.Marshal(cookiePayload{ID: sess.id, Data: sess.data})
	if err != nil {
		return "", err
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := b.aead.Seal(nil, nonce, raw, nil)
	return base64.RawURLEncoding.EncodeToString(append(nonce, out...)), nil
}

// dbBackend stores session data in a `sessions` table; the cookie holds only the
// id. Revocable, no global secret.
type dbBackend struct{ db *data.DB }

type sessionRow struct {
	ID        string `db:"id"`
	Data      string `db:"data"`
	CreatedAt int64  `db:"created_at"`
	ExpiresAt int64  `db:"expires_at"`
}

const (
	sqlSelectSQLite = `select id, data, created_at, expires_at from sessions where id = ?`
	sqlSelectPg     = `select id, data, created_at, expires_at from sessions where id = $1`
	sqlUpsertSQLite = `insert into sessions (id, data, created_at, expires_at) values (?, ?, ?, ?)
		on conflict(id) do update set data = excluded.data, expires_at = excluded.expires_at`
	sqlUpsertPg = `insert into sessions (id, data, created_at, expires_at) values ($1, $2, $3, $4)
		on conflict(id) do update set data = excluded.data, expires_at = excluded.expires_at`
	sqlDeleteSQLite = `delete from sessions where id = ?`
	sqlDeletePg     = `delete from sessions where id = $1`
	sqlSweepSQLite  = `delete from sessions where expires_at <= ?`
	sqlSweepPg      = `delete from sessions where expires_at <= $1`
)

func (b *dbBackend) pg() bool { return b.db.Dialect() == "postgres" }

func (b *dbBackend) pick(sqlite, pg string) string {
	if b.pg() {
		return pg
	}
	return sqlite
}

func (b *dbBackend) ensureSchema(ctx context.Context) error {
	_, err := data.Exec(ctx, b.db, `create table if not exists sessions (
		id         text primary key,
		data       text not null,
		created_at integer not null,
		expires_at integer not null
	)`)
	return err
}

func (b *dbBackend) sweep(ctx context.Context) error {
	_, err := data.Exec(ctx, b.db, b.pick(sqlSweepSQLite, sqlSweepPg), time.Now().Unix())
	return err
}

func (b *dbBackend) load(s *Store, ctx context.Context, value string) (*Session, bool) {
	if value == "" {
		return nil, false
	}
	row, err := data.One[sessionRow](ctx, b.db, b.pick(sqlSelectSQLite, sqlSelectPg), value)
	if err != nil {
		return nil, false
	}
	now := time.Now().Unix()
	if row.ExpiresAt <= now {
		return nil, false
	}
	if s.absolute > 0 && row.CreatedAt+int64(s.absolute.Seconds()) <= now {
		return nil, false
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(row.Data), &m); err != nil {
		return nil, false
	}
	return &Session{id: row.ID, data: m, createdAt: row.CreatedAt}, true
}

func (b *dbBackend) save(s *Store, ctx context.Context, sess *Session) error {
	raw, err := json.Marshal(sess.data)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	created := sess.createdAt
	if created == 0 {
		created = now
		sess.createdAt = now
	}
	expires := now + int64(s.maxAge.Seconds())
	if s.absolute > 0 {
		if cap := created + int64(s.absolute.Seconds()); expires > cap {
			expires = cap
		}
	}
	_, err = data.Exec(ctx, b.db, b.pick(sqlUpsertSQLite, sqlUpsertPg), sess.id, string(raw), created, expires)
	return err
}

func (b *dbBackend) delete(_ *Store, ctx context.Context, id string) error {
	_, err := data.Exec(ctx, b.db, b.pick(sqlDeleteSQLite, sqlDeletePg), id)
	return err
}

func (b *dbBackend) value(_ *Store, sess *Session) (string, error) { return sess.id, nil }
