# 0006 — Templating: stdlib html/template with {{block}} layouts

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
Views render with stdlib **`html/template`** — no third-party template language.
Composition is the stdlib **`{{block}}`** model: a base layout defines named
blocks, each view overrides them by re-defining the same names. Handlers render
with `c.Render(status, name, data)`, which **buffers the whole page** and returns
a `core.Response`; a render error becomes a complete 500 before any byte is
written. Templates are parsed **once** by `views.New(fsys, funcs)` (`layouts/` +
`partials/` shared; one clone per view). Reverse URLs reach templates through
`core.Globals`, a `.URL` method auto-injected into the view struct.

## Because
`html/template` already auto-escapes (XSS) — the same reason we start "ahead" in
`gap.md`. Any other engine means a dependency *plus* hand-rolled/re-imported
escaping, both denied by frugality (`0001`). Buffering makes a broken template
fail **loud and complete** instead of truncating a response whose status is
already committed — the failure mode that eats a debugging session. Parse-once
keeps production cheap; per-view clones give layout inheritance without a
template *language*.

## Alternatives rejected
- **Liquid/Handlebars/Jinja-style engine** — a dependency, a foreign runtime,
  and a re-do of escaping. Borrow the *vocabulary* (layout, block, partials),
  not the engine.
- **Lazy/streaming responses** (`core.View` rendered in `ServeHTTP`) — forces the
  `Context` through `Response.Write` and can emit half a page before an error;
  streaming buys nothing for server-rendered HTML.
- **Partials only, no layout** — duplicates page chrome; doesn't deliver the
  "layout" promise (`gap.md` row 5).
- **Custom `Component` interface** — a parallel renderer duplicating
  `html/template`; keep as a future opt-in hatch, not the default.
- **Per-request parse with request-bound funcs** — enables an in-template `url`
  func and free hot-reload, but re-parses every request and risks dev/prod
  divergence (works in dev, breaks in prod).

## Apply when
Rendering any HTML view; adding layouts/partials; generating links in templates.

## Deviate / ask when
Needing typed components with slots/children, or any engine other than
`html/template` — raise an ADR first.

## Consequences / rules (load-bearing)
- Engine is **stdlib `html/template` only**. No third-party template language,
  no hand-rolled escaping.
- **`{{block}}` layout inheritance**: base defines blocks, views override by
  name, parse base-first (`views.New` clones the parsed base per view).
- Convention: `layouts/` + `partials/` are shared; every other `*.html` is a view
  named by path minus extension (e.g. `users/show`). `DefaultLayout = "base"`.
- Rendering is **buffered** (`c.Render`): errors surface before any write → a
  complete 500. `Response` stays a plain byte value; no streaming.
- `.URL` must be a **method** on `core.Globals`, not a func-typed field —
  `html/template` cannot invoke a func-typed field with arguments
  ("has arguments but cannot be invoked as function").
- `core.Globals` is injected by reflection into an embedded `Globals` field only
  when data is a **pointer to a struct**; value/non-struct data gets no helpers.
- Parse errors are **fail-fast** at startup, with file+line.

## Revisit if
A genuine need for typed components/slots or streaming emerges, or
`html/template`'s block model proves insufficient for nested layouts.

## See also
`0011` — the opt-in template func library (`funcs`) and the `super` / additive
block idiom (partial-call), which cover two `html/template` gaps without changing
this model.
