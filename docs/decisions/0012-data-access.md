# 0012 — Data access: `database/sql` + a thin `data` package (no ORM)

- Status: firm
- Date: 2026-10-12
- Supersedes: none

## Opinion
Raw **`database/sql`**, wrapped by a small `data` package. No ORM, no query
builder, no schema DSL. **SQLite is the default; Postgres is first-class.** The
developer writes the SQL their database understands; the package wires the
connection and returns structs. **No placeholder rewriting** (`?` for SQLite,
`$1` for Postgres).

- `data.Open(ctx, dsn) (*data.DB, error)` infers the dialect from the DSN:
  `postgres://`/`postgresql://` -> Postgres (`github.com/jackc/pgx/v5` stdlib
  adapter); anything else is a SQLite path or DSN (`:memory:`, `file:...`,
  `app.db`) via `modernc.org/sqlite`. Both drivers live here; apps import neither.
- `DB` **wraps** `*sql.DB` rather than embedding it, so the only query entry
  points are the helpers — there is no raw `db.Exec(query)` to grab by mistake.
- Helpers, generic over the `Conn` interface both `*DB` and `*Tx` satisfy:
  - `data.All[T](ctx, c, sql, args...) ([]T, error)`
  - `data.One[T](ctx, c, sql, args...) (T, error)`
  - `data.Exec(ctx, c, sql, args...) (sql.Result, error)`
  - `db.Tx(ctx, func(*data.Tx) error) error`
- `One` maps `sql.ErrNoRows` -> `core.NotFound`, so a loader is just
  `return c.Error(err)` (`0007`).
- The scanner maps columns to struct fields by name, case-insensitively;
  a `db:"col"` tag overrides the field name; exported embedded structs flatten in;
  unknown columns are ignored. Only exported fields are ever written.
- Handlers receive the DB by **dependency injection**, not a global handle:
  `store := handlers.NewStore(db)`, then `app.Config{Routes: store.Register}`
  (the method closes over the DB).
- **Escape hatch:** raw `*sql.Rows`/`*sql.Tx` remain reachable and `sql.Open` is
  always available; the helpers are ergonomics, not a wall.

## Because
Frugality (`0001`) is cutting *glamour, not nutrients*. We cut the ORM — the fast
food that looks productive but adds weight and hides the query — and keep what an
app cannot live without: a real database and its driver. Making the developer
pick or import a driver is exactly the mental load this framework exists to
remove, so the framework carries both drivers and infers which to use from the
connection string. Postgres is a first-class path, not an afterthought — but that
means *supporting* it well, **not pretending the two dialects are the same**. You
write the SQL your database understands; the switch is a DSN change plus the SQL
you already write for that engine. Security: `database/sql` parameterizes; we
never hand-build SQL or hand-roll escaping. DX: write SQL, get structs and
one-line error mapping; `data.Open` is a single call.

## Databases (both first-class)
- **SQLite — pure-Go (`modernc.org/sqlite`):** the default. Zero-setup `go run`,
  no cgo, no server for dev or tests; enough for the small apps this framework
  targets (Rails and Django both default to SQLite).
- **Postgres — `github.com/jackc/pgx/v5` (stdlib adapter):** the maintained
  standard; chosen over `lib/pq` (lighter, but in maintenance mode) on the
  maintenance/security lens.

Both drivers live in `data`; apps import neither.

## Security
Parameterization covers values, not identifiers, and configuration is a secret
surface. Two heuristics carry this: `SQL values are parameters; SQL identifiers
are code` and `Secrets are read from the environment and never echoed`.

- **Injection:** every value a helper takes is bound by the driver — never
  string-concatenated; the package never builds SQL. **But identifiers cannot be
  bound.** A table/column name, an `ORDER BY`, or any other identifier taken from
  the request is injection; choose identifiers from a fixed set in code.
- **Second-order injection:** data read from the DB is untrusted input; passing a
  stored value back in as an identifier is still injection.
- **NULL:** scanning SQL `NULL` into a non-pointer field (`string`, `int`) is a
  returned error, not a silent zero — use `*T` or `sql.Null*` for nullable columns.
- **Scanner:** only exported fields are written; a type mismatch errors, never panics.
- **Transactions:** `db.Tx` rolls back on a returned error and on a panic (then
  re-panics), so a connection is never left open mid-transaction.
- **Secrets:** the DSN is never logged and is scrubbed from driver errors; a
  password in a Postgres URL is redacted before any error surfaces. Read the DSN
  from the environment.
- **TLS:** Postgres's default `sslmode=prefer` is not verification — use
  `sslmode=verify-full` for production DSNs.
- **Resource exhaustion:** SQLite opens a single connection; Postgres is pooled
  (10). Bound result sets (`LIMIT`) and pass a deadline via the request context
  (`c.R.Context()`).
- **Local SQLite files** may hold the whole dataset: keep the file out of
  `static/` and out of git (`.gitignore`).
- **Supply chain:** the two drivers are the framework's only dependencies, so
  `go.mod` is the single watch point; keep them current.
- **Authorization is not here.** The data layer answers "does this row exist",
  not "may this user see it". Every query returning user-owned data must be
  scoped by the owner in the SQL. (Auth/users is `gap` 10, deferred.)

## Alternatives rejected
- **Make the app pick/import a driver** — the mental load this layer removes.
- **Query rewriting / "agnostic" placeholders (`?` -> `$n`)** — rejected: it
  hides the dialect and footguns (Postgres's jsonb `?`); the developer writes
  their database's SQL.
- **Embedding `*sql.DB` on `DB`** — exposes `db.Exec(query)` (no context, no
  helper), which is exactly the call we don't want to invite; wrapping keeps the
  intended path obvious.
- **ORM / active-record (GORM, ent, …)** — the glamour we cut (`0001`).
- **Query builder / schema DSL** — the on-ramp to an ORM.
- **SQLite-only with Postgres as an "escape hatch"** — makes production feel like
  leaving the path; both-first-class removes that.
- **Global DB handle / service locator** — hidden deps, hard tests.
- **Mandatory manual `rows.Scan`** — the boilerplate the helpers exist to remove.

## Apply when
Any handler or loader reads or writes a database.

## Deviate / ask when
- Schema changes — migrations are their own decision (`gap` 9).
- A need for relations appears — reopen this.
- A feature needs dynamic identifiers — build them from a fixed set in code; an
  arbitrary-expression builder is a new ADR.
- **Never** interpolate request data into SQL text.
