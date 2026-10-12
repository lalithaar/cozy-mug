# 0010 — UI component library (`ui`)

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
An optional **`ui`** package provides small, accessible components as Go
**template functions** plus a base stylesheet. It is opt-in: `core` never depends
on it, and an app is fully functional without it (heuristic "just enough styling
to be functional").

- **Components are funcs, Vue-like.** `{{button "Save" (props "variant" "primary")}}`.
- **Mechanism A:** each func executes an **internal template partial** and returns
  `template.HTML` — so `html/template` does the escaping; we never hand-build
  markup with user data.
- **`Props` is variadic:** `func Button(label string, p ...Props)`; template funcs
  may be variadic, so `{{button "Save"}}` and `{{button "Save" (props …)}}` both work.
- **Classes are internal**, namespaced `.cm-*`, and **not a public API** — a
  selector hook for the stylesheet, documented as unstable.
- **CSS variables are the public theming seam** (`--cm-*`): rebrand by overriding
  a variable, no class knowledge, no fork.
- **Escape hatches:** props pass through arbitrary attributes (`id`, `aria-*`,
  `class`); full control = opt out at scaffold and write your own HTML/CSS.

## Because
Vue-like component calls give uniform, accessible markup without a build step and
without making devs memorize a class library. `html/template` funcs are the
native mechanism (no new machinery — `views.New` already takes a `FuncMap`).
Backing funcs with partials keeps escaping safe by construction (`0001`: security
primitives are never hand-rolled). Namespaced classes avoid colliding with whatever
else a page loads; variables make "looks good, never blocks you" true.

## Alternatives rejected
- **A CSS class library** — devs must memorize classes; the "blocker" failure.
- **Funcs that `fmt.Sprintf` markup** — bypasses auto-escaping → XSS risk.
- **Inline `style` on components** — no `:hover`/`:focus`/media queries; bloats HTML.
- **Baking the UI into `core`** — violates `0003` (core must not depend on UI).
- **Copying CSS into each app's `static/`** — drift; "import-only" is one source.

## Apply when
An app wants default styling/components; authoring or extending `ui`.

## Deviate / ask when
Adding a component that needs new markup conventions, a build step, or a
preprocessor — that is an ADR.

## Consequences / rules (load-bearing)
- `ui.Funcs()` returns the component `template.FuncMap`; the `app` glue merges it
  when `Config.UI` is set. `ui.FS()` returns the embedded stylesheet, mounted at
  `/static/ui/` (`base.css`).
- **STATUS: scaffold.** This ADR fixes the *architecture*; the **component set and
  the styling (colors, spacing, actual CSS) are intentionally undecided**. No
  design decisions have been made.
- Contrast-checking / swappable default themes are a captured **idea**
  (`docs/ideas.md`), not decided here.
- Any styling the core depends on, or that becomes mandatory, is a bug.

## Revisit if
Components need templating beyond funcs, or theming needs more than CSS variables.
