# 0014 — Global context: request-scoped values and processors

- Status: proposed
- Date: 2026-10-12
- Supersedes: none

## Opinion
A request carries a small bag of **request-scoped values**. Handlers set them
with `c.Set(name, value)`; templates read them on the embedded `core.Globals` as
`{{.Values.name}}`. Apps register **processors** — `core.Processor`, a
`func(*core.Context) map[string]any` — in `app.Config.Processors` to fill the bag
before every view runs.

- `core.Context.Set(name, value)` / `Context.Global(name)` — write/read a value.
- `core.Globals.Values()` — the bag, for templates: `{{.Values.user}}`.
- `core.Processor func(*Context) map[string]any` — a per-request producer.
- `core.Processors(fns...) core.Middleware` — runs them (nil funcs skipped; a
  later processor wins a key collision).
- `app.Config.Processors` wires them **after sessions and before the CSRF barrier
  and the views**, so a processor may read the session, its writes reach the
  template, and even a rejected unsafe request's error page carries the values.

This is the "context processor" idea (Django) with an explicit bag instead of a
merge into loose template names: the current user, a flash message, a site
title, and anything else a layout needs ride the same mechanism. CSRF stays
first-class (`{{.CSRF}}`, `0013`) because a form field is too common to route
through a name lookup.

```go
// wire (example's flash processor)
app.Must(app.Config{
	// …
	Processors: []core.Processor{
		func(c *core.Context) map[string]any {
			s := session.From(c)
			if s == nil || s.GetString("flash") == "" {
				return nil
			}
			msg := s.GetString("flash")
			s.Delete("flash") // consumed once; session middleware persists it
			return map[string]any{"flash": msg}
		},
	},
})
```
```html
{{with .Values.flash}}<p class="flash">{{.}}</p>{{end}}
```

## Because
Layouts need a few values on every page; threading them through each view model
by hand is exactly the boilerplate this framework exists to remove (Rails uses
before-actions + helpers; Django uses context processors). The framework already
injects `core.Globals` into view models by reflection for `.URL`/`.CSRF` (`0006`,
`0013`), so a value bag is a small extension, not a new subsystem.

Frugality (`0001`) shapes *how*: one map and one function type, no registry, no
reflection-into-arbitrary-fields, no per-processor config. A processor returns a
plain map; the framework merges it. The bag is request-scoped — it dies with the
request, like the `Context` itself — so there is no global state and no locking.

Running **after sessions** is load-bearing: a flash is read out of the session and
deleted, and the session middleware (which wraps processors) persists that delete
on the way out (`0013`). Running **before the CSRF barrier and the views** is what
makes the value present when the view renders — and, because CSRF enforcement is
its own middleware installed after the processors (`0013`), also on the error page
the Errors middleware renders for a rejected unsafe request.

## Security
- **Auto-escaping:** values reach templates through `html/template`, so they are
  escaped like any other data; a processor never assembles HTML.
- **Do not stash secrets:** the bag is data a view *may* render. Keep tokens,
  keys, and internal state out of it; read those from the session in the handler.
- **Trust:** the bag is server-computed per request. Nothing a client sends goes
  in without the handler's explicit choice — do not use it to round-trip input.
- **Cost:** processors run on every request. Keep them cheap; do not hit the DB
  unless a layout truly needs the result on every page.
- **Authorization is not here.** A value in the bag is display state, not a
  permission check; scope queries by owner in the SQL (`0012`).

## Alternatives rejected
- **Inject values as loose template names** (`{{.user}}`) by writing struct fields
  via reflection — magic, collides with view-model fields, only works for structs.
- **A package-level registry / singleton** — global mutable state; hidden deps; a
  testing hazard. The request holds the values.
- **Processors that return `(map, error)`** — forces every processor through an
  error path for a display value; a processor that cannot produce a value returns
  `nil` and the view's `{{if}}` handles absence.
- **Method-per-value on `Globals`** (a `Globals.User()` the app writes) — same
  reflection-into-fields problem, and couples the framework type to app concepts.
- **A general merge/inheritance engine for view data** — the on-ramp to a view
  model framework; the bag is the whole feature.

## Apply when
A layout or many views need the same request-scoped value — current user, flash,
navigation state, a CSRF/site context.

## Deviate / ask when
- A value must be type-checked or validated — shape it in the handler/processor
  and document the key; an untyped bag has no compile-time contract.
- A processor needs to touch the database on *every* request — question whether
  the value belongs on every page; if it does, it is a real decision.
- A failure occurs **before** the processor middleware (e.g. a session load
  error) — that error page has no `.Values`. Everything from the processors
  inward — including the CSRF barrier — already carries them.

## Consequences
- `core.Context` gains a private map; `core.Globals` gains `Values()`.
- `app.Config` gains `Processors`. The assembled order is Logger → Errors →
  Session → Processors → Views -> CSRF -> Router; CSRF sits last so its rejections
  render through the view renderer with the bag populated (`0013`, `0014`).
- Templates may read `.Values`, so a view-model field named `Values` would shadow
  it — reserve the name.
