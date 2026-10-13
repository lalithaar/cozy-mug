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

---

## Timezone handling default

- **Status:** idea (undecided). Revisit once settings/auth landed.
- **Idea:** Store timestamps in UTC by default (`data` already writes `.UTC()`), optionally expose `TZ`/`APP_TIMEZONE` in `settings.Config`, and provide a small formatting helper for local display instead of making every app repeat `time.LoadLocation`. Consider `TZ` env + `time.Now().In(loc)` in handlers/templates via `funcs` date helpers.
- **Why:** Django's `USE_TZ`/`TIME_ZONE` is a frequent source of subtle bugs; picking UTC-in/storage and explicit display formatting removes that class. Fits the "convention over configuration" rule without forcing a knob.
- **Open questions:**
  - Is a single app-level `TZ` enough, or do per-user timezones need data (auth/users) first?
  - Should formatting live in `funcs` (`date` helper) or in templates via a location-aware formatter?
  - Default (`UTC`) vs server-local when unset.
- **Links:** nearest analogue in docs: `settings` package and `funcs/format.go`.
