# cozy-mug

A small, opinionated framework for building **server-rendered HTML apps in Go**.

Go gives you excellent primitives and decides nothing. So every project
re-assembles the same wiring by hand: routing, layouts, data access, sessions,
an error page, a request log. **cozy-mug decides that 90% for you** — the
batteries-included feel of Rails or Django, without their weight.

```go
r.Get("/users/{id}", func(c *core.Context) core.Response {
	u, err := loadUser(c)          // a loader: plain Go, returns (value, error)
	if err != nil {
		return c.Error(err)        // 404, 500, ... rendered for you
	}
	return c.Render(http.StatusOK, "users/show", &UserView{User: u})
}).As("user")                      // named route -> reverse URLs in templates
```

## What it stands for

- **Server-rendered HTML only.** Handlers return HTML, not JSON. No SPA glue, no
  REST ceremony.
- **The essentials are the product, not a saving.** Standard library first, the
  database drivers the one dependency, no build step — *and* data access,
  sessions, forms, settings, errors, and secure defaults included. Their absence is a wall,
  not a frugality.
- **Opinionated, not configurable.** A good default you can opt out of, instead
  of a knob you have to set.
- **Decides the 90%.** Routing, layouts and partials, a model-first data layer,
  sessions and CSRF, an opt-in auth package, one-shot flash messages, errors,
  logging, global context, and a tiny opt-out UI.

## Data access

Your struct is the schema. Tag it once, and the framework derives the table,
columns, primary key, timestamps, optimistic-lock version, and relationships:

```go
type Post struct {
	data.Model `db:"posts"`
	ID        int64     `db:"id auto"`
	Title     string    `db:"title"`
	AuthorID  int64     `db:"author_id index"`
	Author    *User     `db:"ref:author_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Version   int64     `db:"version"`
}
```

```go
posts, err := db.Table[Post]().
	With(P.Author).                    // relations load in batches - no N+1
	Where(P.AuthorID.Eq(uid)).
	OrderBy(P.CreatedAt.Desc()).
	Limit(10).
	All(ctx)
```

Columns are typed tokens, so a mistyped column is a compile error. Mutations are
whole-row with an optimistic lock — a stale write returns `409 Conflict` instead
of clobbering someone else's edit. The same code runs on SQLite or Postgres, and
raw parameterized SQL stays as the escape hatch.

## Status

Early and **pre-release** — the API may change. Built: the core, the `app` glue,
the opt-in auth package, the model-first data layer (with the `cozy migrate new`
model→SQL generator), server-side sessions and CSRF, global view context, the
optional template helpers, server-side forms, settings, and the `cozy` CLI with
project scaffolding and migrations (schema via `cozy migrate`; `EnsureSchema`
removed). Deferred: the UI component set.
A roadmap ships in the repository.

## Layout

An app follows a conventional layout, assembled by the `app` package:

```
myapp/
  main.go        # tiny: build the app, run it
  handlers/      # handlers + loaders + view models, one file per feature
  views/         # templates: layouts/, partials/, errors/
  static/        # assets, served under /static/
```

| Package | What it does |
| --- | --- |
| `core` | Request context, handlers, responses, errors, logging, global values |
| `router` | Named routes, reverse URLs, groups, middleware |
| `views` | Block layouts and partials over `html/template` |
| `app` | Assembles the pieces (logger, errors, sessions, processors, views, routes) |
| `data` | Model-first data layer over `database/sql`: SQLite or Postgres |
| `session` | Server-side sessions (revocable) and CSRF protection, on by default |
| `funcs` | Optional template helpers (loop metadata, formatting) |
| `ui` | Optional component library (opt-in) |
| `settings` | Minimal environment-based config convention |
| `cmd/cozy` | `cozy` CLI: scaffold apps (`cozy new`), run migrations (`cozy migrate`) |
| `examples/app` | Canonical example app (uses `app` glue + 0009 layout) |

## Try it

From a clone of this repository (apply migrations first):

```
go run ./cmd/cozy migrate up --dir examples/app/migrations
go run ./examples/app      # http://localhost:8080 (SQLite; creates notes.db)
```

Set `DATABASE_URL` to a `postgres://…` URL to run the same app on Postgres.
Sessions live in that database by default (revocable; no key to manage). To run
without server state, use the sealed-cookie store (`session.New`) and set
`SESSION_KEY` (base64, 32 bytes — `openssl rand -base64 32`).

## Non-goals

- Not an API backend for JavaScript SPA frameworks.
- Not a REST or JSON framework.
- Not a general-purpose ORM — data access is model-first but bounded (a fixed
  operator set, no lazy loading), with raw SQL as the escape hatch.
- No third-party template engine, no asset build step.
