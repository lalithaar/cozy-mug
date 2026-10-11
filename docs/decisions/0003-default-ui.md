# 0003 — Ship a minimal good-looking default UI, fully opt-out

- Status: firm (in principle); implementation deferred to post-skeleton
- Date: 2026-10-11
- Supersedes: none

## Opinion
Scaffolding produces a **good-looking default UI with zero decisions**: a tiny
hand-written CSS layer built on CSS **system colors** (`Canvas` / `CanvasText`)
plus the smallest possible set of components — all **pluggable and opt-out**.
During scaffold you can decline it entirely and bring your own design system.

Explicitly *not* a design framework: neither Bootstrap nor Tailwind. No utility
classes, no preprocessor, no build step, no config.

## Because
Two goals pull together here: Go's runtime performance *and* the speed of
standing up a UI that doesn't look broken. Left unguided, users hit "how do I
even make this look decent?" — convention over configuration removes that dead
time. System colors give theme-aware, accessible defaults (light *and* dark) for
almost no bytes and no JS. Keeping it opt-out keeps it a default, not a
lock-in, so it never competes with a team's own system.

## Alternatives rejected
- Tailwind / utility CSS — needs a build pipeline (heavier dev loop; violates frugality).
- Bootstrap — large, opinionated, hard to remove piecemeal.
- Ship no CSS — maximal frugality, but leaves exactly the gap the framework
  exists to close.

## Apply when
Scaffolding a new project; authoring components; any temptation to add a CSS
build step, a class system, or a JS-dependent component.

## Deviate / ask when
- Components must stay **plug-out**: nothing in the framework core may depend on
  the UI layer.
- A component needing client JS is a budget question — default is none.
- Adding a CSS preprocessor / class system / build step = ADR + sign-off
  (collides with `0001`).

## Open — mechanism (do not guess)
- Ship the CSS **embedded** in the module (`go:embed`) vs **copied** into the
  project by the scaffold? Embed = always available, unremovable, small module
  growth. Copy = truly opt-out and user-editable, but duplicated per project.
- Byte / component budget is TBD — set from the skeleton's measured baseline.

## Revisit if
The component set starts growing toward a design framework, or the opt-out path
proves awkward in practice.
