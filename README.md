# cozy-mug

A small, opinionated framework for building **server-rendered HTML apps in Go**.

Go gives you excellent primitives and decides nothing. So every project
re-assembles the same wiring by hand: routing, layouts, forms, sessions, an error
page, a request log. **cozy-mug decides that 90% for you** — the
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
- **Frugal by default.** Standard library only; database drivers are the one
  allowed dependency. Small enough to read in one sitting — no ORM, no template
  language, no build step, no dependency tree.
- **Opinionated, not configurable.** A good default you can opt out of, instead
  of a knob you have to set.
- **Decides the 90%.** Routing, layouts and partials, forms, sessions, errors,
  logging, configuration, and a tiny opt-out UI.

## Status

Early and **pre-release** — the API may change. The framework core, data layer,
and sessions/CSRF are built, and apps are assembled by the `app` package; the
remaining app-level pieces (forms, global view context) are still in progress. A
roadmap ships in the repository.

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
| `core` | Request context, handlers, responses, errors, logging |
| `router` | Named routes, reverse URLs, groups, middleware |
| `views` | Block layouts and partials over `html/template` |
| `app` | Assembles the pieces (logger, errors, views, routes) |
| `data` | `database/sql` helpers: SQLite or Postgres, no ORM |
| `session` | Server-side sessions (revocable) and CSRF protection, on by default |
| `funcs` | Optional template helpers (loop metadata, formatting) |
| `ui` | Optional component library (opt-in) |
| `examples/app` | Canonical example app (uses `app` glue + 0009 layout) |

## Try it

From a clone of this repository:

```
go run ./examples/app      # http://localhost:8080 (SQLite; creates notes.db)
```

Set `DATABASE_URL` to a `postgres://…` URL to run the same app on Postgres.
Sessions live in that database by default (revocable; no key to manage). To run
without server state, use the sealed-cookie store (`session.New`) and set
`SESSION_KEY` (base64, 32 bytes — `openssl rand -base64 32`).

## Non-goals

- Not an API backend for JavaScript SPA frameworks.
- Not a REST or JSON framework.
- No ORM, no third-party template engine, no asset build step.
