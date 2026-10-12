# 0013 — Sessions & CSRF: server-side store, sealed-cookie escape hatch

- Status: proposed
- Date: 2026-10-12
- Supersedes: an earlier draft of *this* decision that defaulted to a sealed
  cookie; the reasoning for the reversal is recorded under **Because**.

## Opinion
A session is a small `map[string]any` kept **server-side**, in a `sessions` table
that the framework owns. The browser holds only an opaque 256-bit id in a
`HttpOnly` cookie. A `session` package provides the `*Store` and two middlewares:
`Store.Middleware` loads the session and writes it back only when it changed;
`Store.CSRFMiddleware` enforces **CSRF on unsafe methods** just before the
handler. The `app` package installs CSRF *after* the value processors and the
view renderer, so a rejected unsafe request still renders its error page with the
global values (`0014`).

- `session.NewStore(db, Options{Secure, …})` — the **default**. `db` is a
  `*data.DB` (`0012`). Sessions are stored in a `sessions` table; the framework
  owns the table and creates it with `Store.EnsureSchema(ctx)` (idempotent, not
  an app migration). `Store.Sweep(ctx)` deletes expired rows.
- `session.New(Options{Key, …})` — a sealed-cookie store (below). No server
  state; **the escape hatch** for apps without a database or that must stay
  stateless. It cannot revoke a session.
- Operations: `Get`/`GetString`/`GetInt64`, `Set`, `Delete`, `Clear`, `ID`,
  `Dirty`, `Regenerate` (login), `Destroy` (logout), `CSRFToken`.
- Wire it with `app.Config{Session: store}`; `app` inserts it between Errors and
  Views. `nil` disables sessions.
- Handlers read it via `c.Session()` (the `core.Session` seam) or
  `session.From(c)` for the typed accessors.
- Templates get a ready form field: `{{.CSRF}}` →
  `<input type="hidden" name="csrf" …>`.

**Server store schema** (dialect-specific SQL, no placeholder rewriting — `0012`):

```sql
create table if not exists sessions (
  id         text primary key,
  data       text not null,   -- JSON map
  created_at integer not null,-- unix seconds
  expires_at integer not null
);
```

Writes are an upsert (`on conflict(id) do update set data, expires_at`), so a
`Set` never races itself into duplicate rows. Expiry is enforced on read **and**
by `Sweep`.

**CSRF:** the token lives in the session (server-side for the default store); a
request must echo it in the `csrf` form field or the `X-CSRF-Token` header,
compared in constant time. Safe methods (GET/HEAD/OPTIONS/TRACE) are exempt.
Enforcement is on by default (`Store.CSRFMiddleware`, installed last so error
pages keep their global values — `0014`); `Options.DisableCSRF` opts out.

## Because
Two facts drove the reversal from the cookie default. The first draft argued the
DB store "pulls in a table and a migration we deferred". That was wrong: the
framework can own one idempotent table (`EnsureSchema`) without touching the
app's migration story (`gap` 9), and frugality (`0001`) is about cutting glamour,
not essentials. A session you cannot revoke and that leaks every session on a key
compromise is not an essential made cheap — it is a **security default that only
looks fine until there is auth**. We will not rationalize that with "no auth
yet".

- **Revocation and real expiry.** Only server state can be revoked: "log out
  everywhere", kill one session, shorten a lifetime. A sealed cookie is valid
  until it expires; you cannot take it back.
- **No global secret.** The default needs no signing key, so there is nothing to
  leak or rotate. A key compromise forges *every* session at once — a
  single point of total failure we can simply not have (secrets heuristic).
- **The DB is already here.** `0012` ships both drivers and a `data` package; the
  session store is a thin, dialect-aware use of it, not a new dependency.
- **CSRF on by default.** Cookies are ambient credentials; a token is the only
  reliable defense for unsafe methods. `SameSite=Lax` is a useful second layer,
  not a substitute (legacy browsers, top-level form POSTs).

The sealed-cookie store is retained because it is genuinely useful when there is
no DB (static demos, edge) and because a stateless option is worth one code path.
It is **not** the default; choosing it is a deliberate trade (no revocation) the
ADR records.

## API
```go
// default: server-side, revocable, no key
sessions := core.Must(session.NewStore(db, session.Options{
	Secure: !isDev, // Secure cookie on HTTPS
}))
if err := sessions.EnsureSchema(ctx); err != nil { log.Fatal(err) }
app.Must(app.Config{ /* … */ Session: sessions })
```
```go
// escape hatch: sealed cookie, no server state, needs a key
sessions := core.Must(session.New(session.Options{
	Key: core.Must(session.Key("SESSION_KEY")), // 32 bytes, base64
}))
```
```go
// in a handler
s := session.From(c)        // *session.Session (c.Session() is the seam)
s.Set("user_id", id)        // JSON-encoded into the session row
if s.GetString("user_id") == "" { /* anonymous */ }
s.Regenerate()              // on login: new id, drops fixation
s.Destroy()                 // on logout: deletes the row, clears the cookie
```
```html
<form method="post" action="{{.URL "notes.create"}}">
  {{.CSRF}}
  …
</form>
```

## Security
- **Revocation + expiry:** deleting the row ends the session immediately; both
  `expires_at` (sliding) and an optional `Absolute` cap (from creation) are
  enforced on read.
- **Session fixation:** `Regenerate` mints a new id and drops the payload; call it
  whenever privilege changes.
- **CSRF:** the token is stored in the session and required on unsafe methods,
  compared in constant time; the cookie is `HttpOnly`, `SameSite=Lax`, and
  `Secure` in production.
- **Opaque id:** a 256-bit random id, not derived from data; the cookie carries no
  session contents.
- **Secrets:** the default store needs no key. The cookie store's key comes from
  `SESSION_KEY` and is never logged or echoed (`session.Key` names the variable,
  not its value); prod without a key fails loud. Dev generates an ephemeral key so
  `go run` needs no setup.
- **Write failures fail loud:** if a session cannot be persisted the request is
  errored (`c.Error`), not silently continued — a session that cannot be written
  is not a session.
- **Authorization is not here.** A session identifies; it does not authorize.
  Scope queries by owner (`0012`).

## Alternatives rejected
- **Sealed cookie as the default** — no revocation, and a key leak forges every
  session. Kept only as the escape hatch.
- **Server-side in-memory store** — lost on restart; not multi-process; a second
  store to own.
- **Sign-only (HMAC, readable payload)** — leaks session contents for no real
  saving over AEAD.
- **Hand-rolled store/crypto** — security primitives are never hand-rolled
  (`0001`); the cookie store uses `crypto/aes` + `cipher.NewGCM`, the DB store
  uses bound parameters (`0012`).
- **Let the app own the session table** — pushes a migration we deferred (`gap`
  9) onto every app; the framework owns this one idempotent table instead.

## Superseded draft
An earlier draft of this ADR proposed the sealed cookie as the default. It was
superseded before sign-off (this ADR is still `proposed`); see **Because**.

## Apply when
Any handler needs per-visitor state across requests (login, cart, flash) or
enforces CSRF on an unsafe method.

## Deviate / ask when
- A deployment genuinely cannot use a database — use `session.New` (the sealed
  cookie escape hatch) and accept that sessions cannot be revoked.
- You want `Sweep` scheduled automatically — a `StartSweeper` helper is deferred
  (see **Open**); call `Sweep` from a ticker for now.
- Sessions must be shared across processes without a shared database — a new ADR.
- **Never** disable CSRF on unsafe methods without another layer in front.

## Consequences
- Every app using the default store needs a database (`0012`) — already the data
  layer's assumption, but now a hard session dependency.
- `sessions` rows accumulate; `Sweep` must be scheduled (a tiny periodic goroutine
  or an external tick). Expired rows are ignored on read regardless.

## Open
- `Sweep` scheduling helper (a `Store.StartSweeper`?) — deferred; apps call
  `Sweep` from a ticker for now.
