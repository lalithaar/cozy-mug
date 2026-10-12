# AGENTS.md

An opinionated SSR framework for Go. This file is **hot context** — keep it to
about one screen. Deep reasoning lives in `docs/` and is pulled on demand; do
not inline it here.

## Read every session
- `docs/heuristics.md` — the operational rulebook (conditional rules).
- `docs/idioms.md` — how *we* implement handlers, templates, errors, data, layout.
- `docs/decisions/` — one ADR per load-bearing decision.
- `docs/roadmap.md` — release criteria; the source of truth for what a tag promises.
- `docs/ideas.md` — captured-but-undecided ideas (not commitments).
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
**decided and built through sessions/CSRF and global context / processors; the
remaining v0.1 work is forms, plus settings, docs, and UI. Nothing is undecided;
deferred items live in `docs/roadmap.md`.**

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
- **Versioning tracks decided-and-built readiness** — pre-1.0 numbers track the
  decided 90%, not time; the v0.1 gate is "a stranger ships one small app with no
  wall"; release criteria live in `docs/roadmap.md`. See `docs/decisions/0008`.
- **App layout is conventional, assembled by glue** — `main.go`, `handlers/`
  (handlers + loaders + view models), `views/`, `static/` (served at `/static/`);
  the `app` package wires `Logger → Errors → Views → Router` so apps never
  hand-wire order. Module path `github.com/lalithaar/cozy-mug`. See `docs/decisions/0009`.
- **UI is an optional, import-only component library** — the `ui` package exposes
  components as template funcs (Vue-like), backed by internal partials for safe
  escaping; internal `.cm-*` classes + public `--cm-*` variables. Opt-in; core
  never depends on it. Architecture only — designs/styling not yet decided. See
  `docs/decisions/0010`.
- **Template helpers are an opt-in package; `super` is an idiom** — `funcs`
  (opt-in via `Config.Funcs`) adds what `html/template` lacks (loop metadata +
  arithmetic now). Additive blocks use the partial-call idiom (factor a base
  region into a partial; a view calls it then adds). Engine stays `0006`. See
  `docs/decisions/0011`.
- **Data access is `database/sql` + a thin `data` package; no ORM** — both drivers
  carried (SQLite default via `modernc.org/sqlite`; Postgres via `pgx/v5` stdlib),
  dialect inferred from the DSN; raw SQL pass-through, no placeholder rewriting
  (`?` SQLite, `$1` Postgres); `All[T]`/`One[T]`/`Exec` + `db.Tx` generic over
  `*DB`/`*Tx`; `db:"col"` scanner (exported fields only); `One` maps no-rows to
  `core.NotFound`; DB by closure DI. `DB` wraps (never embeds) `*sql.DB` so there
  is no raw `db.Exec` footgun. Injection/secrets guards baked in (see the two SQL
  heuristics). Migrations deferred. See `docs/decisions/0012`.
- **Sessions & CSRF are server-side by default** — `session` package: a small JSON
  map lives in a framework-owned `sessions` table (idempotent `EnsureSchema`), and
  the cookie holds only an opaque 256-bit id; the CSRF token lives in that session
  and is required (form field `csrf` / `X-CSRF-Token`) on unsafe methods, compared
  constant-time. Revocable, no global secret. A sealed-cookie store
  (`session.New`, AES-256-GCM, key from `SESSION_KEY`) is the escape hatch. Wired
  by `app.Config.Session`: `Store.Middleware` loads/persists, `Store.CSRFMiddleware`
  enforces CSRF last (after processors and views) so a rejected unsafe request's
  error page keeps its global values. Access via `c.Session()` / `session.From(c)`;
  `{{.CSRF}}` renders the field. See `docs/decisions/0013`.
- **Global context is a request-scoped value bag + processors** — `c.Set` /
  `c.Global` fill a per-request map exposed to every view as `{{.Values.name}}`;
  `app.Config.Processors` (`core.Processor`, `func(*Context) map[string]any`) run
  after sessions and before the CSRF barrier/views, so a processor may read the
  session (flash) and its value reaches the template — including a rejected unsafe
  request's error page. CSRF stays first-class `{{.CSRF}}`. See
  `docs/decisions/0014`.

### Open — do not guess, surface to the human
Per-area gap map + stance: `docs/gap.md`. Nothing is currently undecided. Deferred
items (migrations, auth/users, admin/scaffold-gen, agent CLI, i18n/jobs/mail,
tier-3 `{{render}}` boundary) live in `docs/roadmap.md`; reopening any is an ADR.
