# 0013 — Sessions & CSRF: a sealed cookie, no server store

- Status: proposed
- Date: 2026-10-12
- Supersedes: none

## Opinion
A session is a small `map[string]any` stored **entirely in a client cookie**,
sealed with **AES-256-GCM** (authenticated encryption). There is **no server-side
session store**. A `session` package provides the `*Store` (seal/open + cookie
policy) and a middleware that loads the session, enforces **CSRF on unsafe
methods**, and writes the cookie back only when the session changed.

- `session.New(Options{Key, Secure, …})` — `Key` is exactly 32 bytes, read from
  the environment (`SESSION_KEY`, base64; `session.Key("SESSION_KEY")`).
- Operations: `Get`/`GetString`/`GetInt64`, `Set`, `Delete`, `Clear`, `ID`,
  `Regenerate` (login), `Destroy` (logout), `CSRFToken`.
- Wire it with `app.Config{Session: store}`; `app` inserts it between Errors and
  Views. `nil` disables sessions.
- Handlers read it via `c.Session()` (the `core.Session` seam) or
  `session.From(c)` for the typed accessors.
- Templates get a ready form field: `{{.CSRF}}` →
  `<input type="hidden" name="csrf" …>`.

**CSRF:** the token lives in the sealed session cookie; a request must echo it in
the `csrf` form field or the `X-CSRF-Token` header, compared in constant time.
Safe methods (GET/HEAD/OPTIONS/TRACE) are exempt. Enforcement is on by default;
`Options.DisableCSRF` opts out.

## Because
Frugality (`0001`) plus two facts about sessions:

- **Cookie, not a server store.** Rails defaults to a cookie session, Django to a
  DB store. A server store needs a home — memory dies on restart and does not
  scale past one process; a DB table is a migration we deferred (`gap` 9). A
  sealed cookie needs neither, and matches the target: small apps, one process.
  The price is the ~4 KB cookie limit — fine for a user id, a flash, and a token.
- **Encrypt, not merely sign.** A signed cookie blocks tampering but leaves
  contents readable; sessions carry user ids and flashes, and reading them is a
  needless leak. AES-GCM is stdlib and one code path gives integrity **and**
  confidentiality. Security primitives are never hand-rolled (`0001`): we use
  `crypto/aes` + `cipher.NewGCM`, not a scheme of our own.
- **Key from the environment.** The seal is only as secret as its key; a key in
  source is a breach. `SESSION_KEY` is read at boot (secrets heuristic). Dev
  generates an ephemeral key so `go run` needs no setup; prod fails loud without
  one.
- **CSRF on by default.** Cookies are ambient credentials; a token is the only
  reliable defense for unsafe methods. `SameSite=Lax` is a useful second layer,
  not a substitute (legacy browsers, top-level form POSTs).

## API
```go
sessions := core.Must(session.New(session.Options{
	Key:    core.Must(session.Key("SESSION_KEY")), // 32 bytes, base64
	Secure: !isDev,                                // Secure cookie on HTTPS
}))
app.Must(app.Config{ /* … */ Session: sessions })
```
```go
// in a handler
s := session.From(c)        // *session.Session (c.Session() is the seam)
s.Set("user_id", id)        // JSON-encoded into the sealed cookie
if s.GetString("user_id") == "" { /* anonymous */ }
s.Regenerate()              // on login: new id, drops fixation
s.Destroy()                 // on logout: clears the cookie
```
```html
<form method="post" action="{{.URL "notes.create"}}">
  {{.CSRF}}
  …
</form>
```

## Security
- **Confidentiality + integrity:** AES-256-GCM; the cookie is opaque ciphertext. A
  tampered or truncated cookie fails to open and is discarded — a fresh, empty
  session is used.
- **Session fixation:** `Regenerate` mints a new id and drops the payload; call it
  whenever privilege changes.
- **CSRF:** the token is stored in the session and required on unsafe methods,
  compared in constant time; the cookie is `HttpOnly`, `SameSite=Lax`, and
  `Secure` in production.
- **Secrets:** the key comes from `SESSION_KEY`, is never logged or echoed
  (`session.Key` names the variable, not its value); prod without a key fails loud.
- **Size:** a sealed session over the cookie limit is **dropped, never partially
  written**. Keep sessions small — data in the session is data the client holds.
- **Authorization is not here.** A session identifies; it does not authorize.
  Scope queries by owner (`0012`).

## Alternatives rejected
- **Server-side in-memory store** — lost on restart; not multi-process; a store
  to own and lock.
- **Server-side DB store** — durable, but pulls in a session table and a
  migration (`gap` 9) for state a cookie can carry.
- **Sign-only (HMAC, readable payload)** — leaks session contents for no real
  saving over AEAD.
- **Hand-rolled cipher/format** — never (`0001`); use `crypto/aes` +
  `cipher.NewGCM`.
- **CSRF opt-in / off by default** — the safe default must be the default.
- **A global `session.Get` singleton** — hidden dependency; the request holds it.
- **Ephemeral key generated in prod** — silently invalidates every session on
  each restart; fail loud instead.

## Apply when
Any app that keeps per-user state across requests (login, cart, flash) or accepts
a state-changing request.

## Deviate / ask when
- A server-side store is genuinely required (large sessions, revocation lists) —
  reopening this is a store decision.
- A cross-origin API needs a CSRF exemption — scope the opt-out and document it.
- Multiple signing keys / rotation for zero-downtime deploys — additive, but an ADR.
