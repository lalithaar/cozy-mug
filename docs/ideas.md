# Ideas

Captured but **undecided**. Nothing here is a commitment. An idea graduates by
becoming an ADR (`docs/decisions/`) or a `docs/roadmap.md` item. Keep entries
short; record the reason, not just the wish.

---

## Contrast-aware theming + a few default themes

- **Status:** idea (undecided). Sits on top of the `ui` library (`0003`); revisit
  after the component library exists.
- **Idea:** The `ui` library already exposes CSS variables (`--cm-*`) as the
  theming seam. Use that seam to (a) check / warn / fix color contrast, and
  (b) ship 3–4 curated default themes users swap, Obsidian-style — pick a theme
  or drop in custom colors, no classes involved.
- **Why:** Variable theming is already the plan, so contrast checking is nearly
  free on top of it, and it turns "looks good by default" into "looks good *and*
  accessible by default". It also lets us demo a few nice themes instead of one.
- **Precedent:** [CM-Colors](https://github.com/comfort-mode-toolkit/cm-colors/) —
  "you pick the colors, we make them readable"; finds minimally-changed colors
  that meet WCAG. Note: Python + **GPL-3.0**, so reference/inspiration only —
  never vendored or depended on (frugality + license).
- **Open questions:**
  - **When** to check: dev-only check/warn, a build step (we have none), or
    runtime? Probably dev-only.
  - **Themes**: one stylesheet + a `data-theme` attribute, or separate files?
  - **How many / which**: light, dark, maybe one high-contrast — 3–4 at most.
  - **Scope**: checking contrast is small (WCAG relative luminance is a few
    stdlib lines); *fixing* is an optimization — likely out of scope, leave to an
    external tool.
- **Links:** https://github.com/comfort-mode-toolkit/cm-colors/,
  https://cm-colors.readthedocs.io/
