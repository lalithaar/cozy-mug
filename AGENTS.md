# AGENTS.md

An opinionated SSR framework for Go. This file is **hot context** — keep it to
about one screen. Deep reasoning lives in `docs/` and is pulled on demand; do
not inline it here.

## Read every session
- `docs/heuristics.md` — the operational rulebook (conditional rules).
- `docs/idioms.md` — how *we* implement handlers, templates, errors, data, layout.
- `docs/decisions/` — one ADR per load-bearing decision.
- `docs/gap.md` — stdlib-vs-Rails/Django gap map + per-area stance.
- `docs/LOG.md` — append-only trail; read the last ~20 lines to catch recent drift.

## Meta-goal: memory for future sessions
These docs are the agent's persistent memory. There is no recall between
sessions except these files. Rules:
- Made a non-obvious call a human could contest -> append a decision + LOG entry.
- Deviated from a rule -> record why at the site *and* in LOG.
- Guessed something reversible -> mark it `assumption` and flag for review.
- New rules enter as `soft`. Promote to `firm` only with human sign-off.
- Never delete reasoning; supersede it (mark old entry superseded, link the new one).

## Decide vs ask
- Default: **ask** when a choice is (a) hard to reverse, (b) cross-cutting, or
  (c) collides with a `firm` rule. Otherwise decide, record, move on.
- If a case isn't covered by a rule: implement the smallest reversible thing,
  log the gap, and ask only if it generalizes.

## Opinion ledger
Firm opinions live in `docs/decisions/`. Project status:
**early skeleton — handler contract, routing, templating, errors/logging
decided; do not guess `open` items.**

### Firm
- **Frugality is the top-level filter** — every addition (dep, line, export,
  knob, feature, byte) is denied by default; stdlib-only, DB driver the sole dep
  exemption, security primitives never hand-rolled. See `docs/decisions/0001`.
  Filter all other opinions through it.
- **Scope is narrow and stated** — server-rendered HTML apps only. NOT an API
  backend for JS SPA frameworks, NOT a REST framework. Non-goals are explicit on
  purpose. A dev-only agent CLI (browserless e2e testing) is in scope,
  post-skeleton. See `docs/decisions/0002`.
- **Default UI is minimal and opt-out** — scaffold ships tiny system-color CSS
  (`Canvas`/`CanvasText`) + smallest components, fully plug-out. Not Bootstrap,
  not Tailwind, no build step. Core never depends on it. See `docs/decisions/0003`.
- **Handlers return a value** — `func(*Context) Response`; still an
  `http.Handler`; raw `HandlerFunc` is the escape hatch. See `docs/decisions/0004`.
- **Routing wraps `ServeMux`** — named routes + reverse URLs, groups,
  middleware, canonical-slash; params via `PathValue`. See `docs/decisions/0005`.
- **Templating is stdlib `html/template`** — `{{block}}` layouts + partials,
  buffered `c.Render`, parse-once; reverse URLs via an injected `.URL` method.
  See `docs/decisions/0006`.
- **Errors are tiered + one boundary** — boot-only panic/`Must`; loaders return
  `core.HTTPError`; `c.Error` makes a `Failure` Response (contract stays `0004`);
  `core.Errors` recovers + renders dev page / prod `views/errors/<status>.html`.
  `core.Logger` = slog line + request id. Tier-3 `{{render}}` boundary deferred.
  See `docs/decisions/0007`.

### Open — do not guess, surface to the human
Per-area gap map + stance: `docs/gap.md`. Items:
- data / DB layer
- project layout / boilerplate contract
