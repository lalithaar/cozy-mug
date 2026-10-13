# 0009 — Project layout & app glue

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
An app follows a **conventional layout**, and one **glue package** assembles it.

```
myapp/
  go.mod
  main.go        # tiny: build the app, run it
  handlers/      # handlers + loaders + view models, one file per feature
  views/         # templates (see 0006): layouts/, partials/, errors/
  static/        # assets, served under /static/
```

The framework ships an `app` package: `app.Config` + `app.New` that wires
`Logger → Errors → Session → Processors → Views → CSRF → Router`, mounts static,
and exposes an `http.Handler`. Apps never hand-wire middleware order.

## Because
Layout is the `rails new` moment — the gap (`docs/gap.md` row 1) between "stdlib
decides nothing" and "a project". Naming and defaults are the whole value of
convention over configuration (`0003` heuristic). The glue exists because the
ordering knowledge is load-bearing and easy to get wrong, and **an import cycle
forces it into its own package**: `router` and `views` both import `core`, so
`core` cannot import them.

## Alternatives rejected
- **Handlers in `main` / root package** — couples routing to the entrypoint and
  doesn't scale past a toy.
- **`controllers/` / `views/` for handlers** — "views" collides with templates;
  "controllers" is Rails jargon for the same thing.
- **Each app hand-wires middleware** — repeats the exact ordering bug the
  framework exists to prevent.
- **Glue in `core`** — impossible without an import cycle.

## Apply when
Creating an app; deciding where a handler, template, or asset lives.

## Deviate / ask when
An app wants a different tree (a flat prototype, `cmd/` binaries). The layout is
a default, not a cage — but a *different* convention should be raised, not
silently adopted in the framework.

## Consequences / rules (load-bearing)
- Handler files live in **`handlers/`**; the route table is a single
  `Register(*router.Router)` function. Routing and handlers are separate
  concerns (routing = the `router` package + `Register`).
- Templates live in **`views/`** with the `layouts/` + `partials/` convention
  (`0006`); a feature's handler renders a view whose name mirrors the handler's
  feature (`handlers/users.go` → `"users/show"` → `views/users/show.html`).
- **`static/`** is served under the `/static/` **prefix** (fingerprint/cache
  semantics). Root-mounted files (`favicon.ico`) are deferred; no `public/` yet.
- `app.Config` — `Assets fs.FS` (embeds `views/`, optional `static/`),
  `Routes func(*router.Router)`, `Dev bool`, `Logger *slog.Logger`, `UI bool`,
  `Funcs template.FuncMap`, `Session *session.Store`, `Processors []core.Processor`.
  `app.New` returns `error`; `app.Must` panics at boot (`0007`).
- The app is an `http.Handler` (`ServeHTTP`) so tests and the agent CLI drive it
  in-process; `App.Router()` is the escape hatch.
- **Module path is `github.com/lalithaar/cozy-mug`** (replaces the `cozymug`
  placeholder); imports are `github.com/lalithaar/cozy-mug/core`, etc.
- A `cozy new` scaffold **CLI is deferred**; the convention + a canonical
  example double as the template (the generator is the agent CLI, row 16).

## Revisit if
Apps regularly need a second conventional shape, or the glue grows options it
can't justify.
