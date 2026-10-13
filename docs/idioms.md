# Idioms

Concrete "how we do it here" techniques (handlers, templates, errors, data,
layout).

Rule: every idiom must cite the decision that justifies it (`docs/decisions/NNN`).
If you can't cite a decision, write the decision first.

## Render a view (`0006`)
Pass a **pointer** to the view struct — `core.Globals` injection needs it — and
embed `core.Globals` only when the view generates links.

```go
type ShowView struct {
	core.Globals        // gives the template .URL
	User User
}

func show(c *core.Context) core.Response {
	return c.Render(http.StatusOK, "users/show", &ShowView{User: load(c.Param("id"))})
}
```

```html
{{define "title"}}{{.User.Name}}{{end}}
{{define "content"}}<h1>{{.User.Name}}</h1>{{end}}
```

- Layout lives in `views/layouts/base.html` as
  `{{define "base"}}…{{block "content" .}}…{{end}}…{{end}}`; partials in
  `views/partials/`. A view file defines the blocks it fills. (`0006`)
- Views are named by path minus extension (`views/users/show.html` → `"users/show"`). (`0006`)
- Wire once at startup: `set, _ := views.New(fsys, nil)` then
  `r.Use(views.Middleware(set))`. (`0006`)
- `.URL` is `{{.URL "user" "id" .User.ID}}` (name/value pairs; values may be any
  type, stringified, so integer IDs need no conversion); it errors rather than
  emitting an empty link. (`0006`)

## Template funcs (`0011`)
`html/template` ships almost no helpers. Add the opt-in `funcs` package to the
func map:

```go
a := app.Must(app.Config{Assets: assets, Routes: handlers.Register, Funcs: funcs.Funcs()})
```

```html
{{range $i, $n := .Notes}}
  <li class="{{cycle $i "odd" "even"}}">{{counter $i}}. {{$n.Title}}</li>
{{end}}
<p>{{len .Notes}} {{pluralize (len .Notes) "note" "notes"}}</p>
<p class="body">{{linebreaksbr .Note.Body}}</p>
```

- Loop/arith: `counter` (1-based), `first`, `last` (needs the collection),
  `even`/`odd`, `cycle`, `add`, `sub`.
- Format: `upper`, `lower`, `capitalize`, `truncatechars`, `truncatewords`,
  `slugify`, `join`, `default`, `pluralize`, `comma`, `date`, `linebreaksbr`.
- Pipelines put the value last: `{{.Body | truncatechars 40}}`,
  `{{.Created | date "Jan 2, 2006"}}`, `{{.Name | default "anon"}}`. (`0011`)

## Add to a base region — `super` (`0011`)
`{{block}}` **replaces** wholesale; there is no `super`. Recover "parent + my
addition" by factoring the region into a partial and calling it:

`views/layouts/head.html`:
```html
{{define "layout-head"}}<link rel="stylesheet" href="/static/style.css">{{end}}
```
`views/layouts/base.html` (`{{block}}` default calls the partial):
```html
{{block "head" .}}{{template "layout-head" .}}{{end}}
```
A view that wants the default **plus** its own content re-calls the partial:
```html
{{define "head"}}{{template "layout-head" .}}<meta name="note-id" content="{{.Note.ID}}">{{end}}
```
Multi-level layouts compose the same way (a base region calling another base's
partial). No engine change, no new model — just `{{define}}`/`{{template}}`. (`0011`)


## Data access — model-first (`0016`)
Open once at boot and inject the handle by closure (`0009`). Then build queries
against the **model**, not hand-written SQL:

```go
db := core.Must(data.Open(ctx, os.Getenv("DATABASE_URL"))) // file:/:memory: -> SQLite; postgres:// -> Postgres
defer db.Close()

// The model is the single source of truth: table, columns, pk, maintained
// timestamps/version, relations. The derived SQL is the same in both dialects.
type Note struct {
	data.Model `db:"notes"`
	ID        int64     `db:"id auto"`
	Title     string    `db:"title"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Version   int64     `db:"version"`
}

// Typed column tokens make column typos compile errors. cozy gen will emit this
// block; hand-written today.
var notes = struct {
	ID, Title, Body data.Col
}{
	ID:    data.C[Note]("id"),
	Title: data.C[Note]("title"),
	Body:  data.C[Note]("body"),
}

func (s *Store) allNotes(ctx context.Context) ([]Note, error) {
	return s.db.Table[Note]().OrderBy(notes.ID.Asc()).All(ctx)
}

func (s *Store) noteByID(ctx context.Context, id int64) (Note, error) {
	return s.db.Table[Note]().Where(notes.ID.Eq(id)).One(ctx)
}

func (s *Store) addNote(ctx context.Context, title, body string) (*Note, error) {
	n := &Note{Title: title, Body: body}
	if err := s.db.Table[Note]().Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil // back-filled: id, timestamps, version
}
```

- Query pipe: `Table[M]().Where(…).OrderBy(…).Limit(n).Offset(n).Page(p, per)` —
  every step returns a fresh `Query`; terminal calls are `Count`, `All`, `One`,
  `Page`. Go has no external receivers, so `db.Posts()` is impossible — the
  tokens keep the pipe checked instead. (`0016`)
- `One` returns `core.NotFound` on no rows; a loader hands it to `c.Error` (`0007`).
- Operators are bounded: `Eq Ne Gt Gte Lt Lte Like In NotIn Nil NotNil`. A
  predicate or sort on an unknown column is a loud error, never silent `NULL`.
- Mutations are whole-row with an optimistic lock: `Update` writes the row under
  `WHERE id = ? AND version = ?` (the version column is auto-maintained) and
  returns `core.Conflict` (409) on a stale write — the handler re-renders
  "saved elsewhere, reload". `Delete` is whole-row under the pk; deleting a
  missing row is an error. (`0016`)
- A `created_at`/`updated_at` column is maintained automatically when present. (`0016`)
- Relations are declared as `*Child` (belongs-to) / `[]*Child` (has-many) fields
  tagged `db:"ref:<fk-column>"`. They load **only when asked**, one extra query
  per relation: `.With(notes.Author)` — no lazy N+1. (`0016`)
- Schema at boot: none. `Table[M]().SchemaSQL()` exposes the model-derived DDL
  for generators/tools/tests, but app tables arrive only through `cozy migrate`
  (reviewable, per-dialect files under `migrations/<dialect>`). DB seeding
  (`store.Seed`) is allowed; DDL at boot is not. (`0016`)
- The raw escape hatch stays: `data.All`/`One`/`Exec`/`Tx` for a query a model
  can't express. args are always bound, never interpolated — and never as
  identifiers (`ORDER BY`, column name): pick from a fixed set. The DSN is a
  secret: read from the environment, never log it. (`0016`, `0012`, secrets heuristic)

## Sessions & CSRF (`0013`)
Sessions live server-side by default (a framework-owned `sessions` table); wire the
store in `app.Config` (`store.Middleware` loads it, then processors, views, and
finally `store.CSRFMiddleware`) and read it in handlers.

```go
sessions := core.Must(session.NewStore(db, session.Options{
	Secure: !isDev, // Secure cookie on HTTPS
}))
if err := sessions.EnsureSchema(ctx); err != nil { log.Fatal(err) } // idempotent
app.Must(app.Config{ /* … */ Session: sessions })
```

```go
func login(c *core.Context) core.Response {
	s := session.From(c) // *session.Session; c.Session() is the seam
	if !valid(c.Form("password")) {
		return c.Error(core.Unauthorized(nil))
	}
	s.Regenerate() // new id on a privilege change (session fixation)
	s.Set("user_id", user.ID)
	return core.Redirect(http.StatusSeeOther, "/")
}
```

- Any form that posts includes `{{.CSRF}}`; the middleware rejects a missing or
  wrong token with 403 *before* the handler runs. For JS, send `X-CSRF-Token`. (`0013`)
- Safe methods (GET/HEAD/OPTIONS/TRACE) need no token; logout calls `s.Destroy()`. (`0013`)
- Values are JSON-encoded into the session row, so keep them small and
  JSON-friendly. (`0013`)
- Need no server state? `session.New(session.Options{Key})` is the sealed-cookie
  escape hatch — but it cannot revoke; prefer the DB store. The key is a secret:
  `session.Key("SESSION_KEY")` reads it without echoing it. (`0013`, secrets heuristic)
- `Sweep(ctx)` drops expired rows; expired rows are ignored on read regardless. (`0013`)

## Global context (`0014`)
Values that every layout needs — current user, flash, a site title — go in the
request's value bag, not threaded through each view model. A **processor** fills
it before the view runs; templates read it as `{{.Values.name}}`.

```go
// app.Config
Processors: []core.Processor{
	func(c *core.Context) map[string]any {
		s := session.From(c)
		if s == nil || s.GetString("flash") == "" {
			return nil
		}
		msg := s.GetString("flash")
		s.Delete("flash") // consumed once; the session middleware persists the delete
		return map[string]any{"flash": msg}
	},
},
```
```html
{{with .Values.flash}}<p class="flash">{{.}}</p>{{end}}
```

- A handler can also set one: `c.Set("title", t)`; read it back with
  `c.Global("title")`. (`0014`)
- Processors run **after sessions, before views** — so they may read the session,
  and their values are present when the view renders. (`0014`)
- Keep the bag for display state; do not stash secrets or trust it as a permission
  check. CSRF stays `{{.CSRF}}`, not a bag value. (`0014`, `0013`)
- A processor returns `nil` when it has nothing; the view's `{{if}}`/`{{with}}`
  handles absence. Later processors win a key collision. (`0014`)

## Report an error (`0007`)
Loaders return a structured error; the route maps it with `c.Error`. Wire
`Logger` + `Errors` once, Logger outermost.

```go
func loadUser(c *core.Context) (User, error) {
	u, err := db.UserByID(c.Param("id"))
	if err != nil {
		return User{}, core.NotFound(err)
	}
	return u, nil
}

func showUser(c *core.Context) core.Response {
	u, err := loadUser(c)
	if err != nil {
		return c.Error(err) // 404, rendered by the Errors middleware
	}
	return c.Render(http.StatusOK, "users/show", &ShowView{User: u})
}
```

```go
opts := core.Options{Dev: isDev, Logger: slog.Default()}
// minimal hand-wiring: Logger outermost, Errors next. app.New inserts
// Session -> Processors -> Views -> CSRF after Errors (0013, 0014, 0009).
r.Use(core.Logger(opts), core.Errors(opts), views.Middleware(set))
```

- Use `panic`/`core.Must` **only** for boot; the `Errors` middleware recovers a
  stray panic and renders 500. (`0007`)
- Plain error (not an `HTTPError`) → 500. Return `core.NotFound`/`BadRequest`/…
  or the `core.HTTPError` struct to set the status. (`0007`)
- Add `views/errors/<status>.html` to brand error pages; `Errors` falls back to a
  plain page. Dev shows status/stack/id; prod hides causes. (`0007`)

## Assemble an app (`0009`)
An app is a conventional layout assembled by the `app` package. `main.go` stays
tiny; handlers live in `handlers/`; routing is one `Register` function.

```
myapp/
  main.go
  handlers/
  views/    static/
```

`main.go` — open the DB once and inject it by handing over the store's method
(`0009`, `0016`):
```go
//go:embed views static
var assets embed.FS

func main() {
	db := core.Must(data.Open(context.Background(), databaseURL()))
	defer db.Close()
	store := handlers.NewStore(db)

	a := app.Must(app.Config{
		Assets: assets,         // views/ (+ static/ if present)
		Routes: store.Register, // func(*router.Router); closes over the DB
		Dev:    os.Getenv("APP_ENV") != "production",
		UI:     true,           // opt into the default component library
	})
	log.Fatal(a.ListenAndServe(":8080"))
}
```

`handlers/handlers.go` — the store carries the DB so the routes close over it (`0016`):
```go
type Store struct{ db *data.DB }

func NewStore(db *data.DB) *Store { return &Store{db: db} }

func (s *Store) Register(r *router.Router) {
	r.Get("/users/{id}", s.showUser).As("user")
}

func (s *Store) showUser(c *core.Context) core.Response {
	u, err := s.db.Table[User]().Where(users.ID.Eq(c.Param("id"))).One(c.R.Context())
	if err != nil {
		return c.Error(err) // One maps no-rows to core.NotFound (0007)
	}
	return c.Render(http.StatusOK, "users/show", &UserView{User: u})
}
```

- The full order (Logger -> Errors -> Session -> Processors -> Views -> CSRF ->
  Router) is wired by `app`, never by hand.
- The app is an `http.Handler`, so tests drive it with `httptest` (`a.ServeHTTP`).
- `static/` is served at `/static/`; with `UI: true`, the `ui` stylesheet is at
  `/static/ui/base.css`. (`0009`, `0010`)
