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
- **Frugal by default.** Standard library only; a database driver is the one
  allowed dependency. Small enough to read in one sitting — no ORM, no template
  language, no build step, no dependency tree.
- **Opinionated, not configurable.** A good default you can opt out of, instead
  of a knob you have to set.
- **Decides the 90%.** Routing, layouts and partials, forms, sessions, errors,
  logging, configuration, and a tiny opt-out UI.

## Status

Early and **pre-release**: the framework core is built (see the latest release),
while the app-level pieces are still in progress. It is **not yet installable** —
the module path is a placeholder. A roadmap ships in the repository.

## Layout

| Package | What it does |
| --- | --- |
| `core` | Request context, handlers, responses, errors, logging |
| `router` | Named routes, reverse URLs, groups, middleware |
| `views` | Block layouts and partials over `html/template` |
| `examples/hello` | Minimal app |
| `dummy` | A slightly larger app (groups, params, a partial, an error page) |

## Try it

From a clone of this repository:

```
go run ./examples/hello   # http://localhost:8080
go run ./dummy            # http://localhost:8080
```

## Non-goals

- Not an API backend for JavaScript SPA frameworks.
- Not a REST or JSON framework.
- No ORM, no third-party template engine, no asset build step.
