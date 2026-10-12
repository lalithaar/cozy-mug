# 0011 — Template func library is an opt-in package; `super` via partial

- Status: firm
- Date: 2026-10-12
- Supersedes: none

## Opinion
Close the two real gaps in `html/template` (`docs/gap.md`) without changing the
engine (`0006`):

1. **Helpers live in an opt-in package, not in `views`.** A new `funcs` package
   ships small template functions the stdlib lacks — first the *loop metadata* and
   arithmetic (`counter`, `first`, `last`, `even`, `odd`, `cycle`, `add`, `sub`),
   later formatting. Apps add it with `app.Config.Funcs` / `views.New(fsys, funcs)`.
2. **`super` / additive blocks are an idiom, not a feature.** A base layout
   factors each overridable region into a named partial and exposes it through a
   `{{block}}` whose default *calls that partial*; a view that wants "parent + my
   addition" redefines the block and calls the same partial first. Multi-level
   nesting is composed the same way.

## Because
`html/template` evaluates a func map but its pipeline has **no arithmetic**, no
loop counters/first/last, and no `cycle`. Shipping these in `views` would burden
every app (frugality, `0001`) and grow core's surface; the `ui` precedent
(`0010`) already established "optional capability is an import you opt into."
The `{{block}}` model **replaces wholesale** and has no `super`; the partial-call
idiom recovers "default + addition" using only `{{define}}`/`{{template}}`, so no
new model, no dependency, no engine change.

## Alternatives rejected
- **Build helpers into `views`** — every app pays for functions most won't call;
  violates the opt-in precedent.
- **A `super`/nested-layout engine feature** — a parallel template mechanism
  duplicating `{{block}}`; reopening `0006` for ergonomics it can already express.
- **A third-party func library / template engine** — a dependency plus foreign
  escaping (denied by `0001`, `0006`).

## Apply when
An app needs arithmetic or loop positions in a template (`funcs.Funcs()`), or a
view must *add to* rather than *replace* a base region (the partial-call idiom).

## Deviate / ask when
The partial-call idiom proves too clumsy for nested layouts (reopen `0006`), or a
helper wants to *format/escape* per context (that is `html/template`'s job).

## Revisit if
Formatting helpers accrete far enough to need their own package(s), or real
nested-layout demand outgrows the idiom.
