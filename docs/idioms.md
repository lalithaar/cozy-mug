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


## Data access (`0012`)
Open once at boot and inject the handle by closure (`0009`):

```go
db := core.Must(data.Open(ctx, os.Getenv("DATABASE_URL"))) // file:/:memory: -> SQLite; postgres:// -> Postgres
defer db.Close()

store := handlers.NewStore(db)     // type Store struct { db *data.DB }
app.Config{Routes: store.Register} // the method closes over the DB
```

Write the SQL your database speaks — no rewriting (`?` SQLite, `$1` Postgres).
Args are bound parameters, so never build SQL from input:

```go
type Note struct {
	ID    int64  `db:"id"`
	Title string `db:"title"`
}

func (s *Store) allNotes(ctx context.Context) ([]Note, error) {
	return data.All[Note](ctx, s.db, `select id, title from notes order by id`)
}

func (s *Store) noteByID(ctx context.Context, id int64) (Note, error) {
	return data.One[Note](ctx, s.db, `select id, title from notes where id = ?`, id)
}
```

- `One` returns `core.NotFound` on no rows, so the handler is `return c.Error(err)`. (`0007`)
- Column -> field by name, case-insensitive; `db:"col"` wins; exported embeds
  flatten in; unknown columns are ignored. Nullable column -> `*T`/`sql.Null*`. (`0012`)
- Transaction (rolls back on error or panic):
  `db.Tx(ctx, func(tx *data.Tx) error { _, err := data.Exec(ctx, tx, `…`); return err })`. (`0012`)
- `data.DB` wraps `*sql.DB`, so the entry points are the helpers — there is no
  raw `db.Exec(query)`. `*sql.Rows`/`*sql.Tx` stay reachable as the escape hatch.
- Never interpolate input as an identifier (`ORDER BY`, column name) — pick from a
  fixed set. The DSN is a secret: read it from the environment, never log it. (`0012`)

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
(`0009`, `0012`):
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

`handlers/handlers.go` — the store carries the DB so the routes close over it (`0012`):
```go
type Store struct{ db *data.DB }

func NewStore(db *data.DB) *Store { return &Store{db: db} }

func (s *Store) Register(r *router.Router) {
	r.Get("/users/{id}", s.showUser).As("user")
}

func (s *Store) showUser(c *core.Context) core.Response {
	u, err := data.One[User](c.R.Context(), s.db,
		`select id, name from users where id = ?`, c.Param("id"))
	if err != nil {
		return c.Error(err) // One maps no-rows to core.NotFound (0007)
	}
	return c.Render(http.StatusOK, "users/show", &UserView{User: u})
}
```

- Middleware order (Logger -> Errors -> Views) is wired by `app`, never by hand.
- The app is an `http.Handler`, so tests drive it with `httptest` (`a.ServeHTTP`).
- `static/` is served at `/static/`; with `UI: true`, the `ui` stylesheet is at
  `/static/ui/base.css`. (`0009`, `0010`)
