# AGENTS.md

An opinionated SSR framework for Go. This file is **hot context** — keep it to
about one screen. Deep reasoning lives in `docs/` and is pulled on demand; do
not inline it here.

## How to interact with the human
- **Never use a multiple-choice picker** to ask a clarifying question (e.g. the
  question tool). Ask in prose. Pickers are for products the human builds, not
  for our conversation.
- **Don't jump to solution mode.** Respond to what the human actually said.
- **Talk like a collaborator, not a debate opponent.** No performative framing —
  no "let me push back", "I'd sharpen your wording", "second push". Don't treat
  the human's statements as claims awaiting your critique or approval. Just talk.
- **No verbal tics or verdict openers.** Banned outright: "Fair." (and its
  cousins). Don't open a reply by grading the human's point; just answer it.

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
- **The goal is Django/Rails-caliber DX for a secure SSR app; frugality is the
  means, not the goal** — a stranger goes from an empty dir to a deployed, small,
  real, secure server-rendered app with no wall. Frugality cuts the *unnecessary*
  (deps, knobs, bytes, gobuffalo-style glue; every addition denied by default;
  stdlib-only, DB driver the sole dep exemption, security primitives never
  hand-rolled) but **never an essential capability** — data access, migrations,
  forms, auth/identity, settings, sessions, errors + secure defaults are the
  *product*, and their absence is a wall, not a saving. See `docs/decisions/0015`
  (reorders `0001`). Filter all other opinions through this.
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
  the `app` package wires `Logger → Errors → Session → Processors → Views → CSRF →
  Router` so apps never hand-wire order. Module path `github.com/lalithaar/cozy-mug`. See `docs/decisions/0009`.
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
- **Data access is model-first, derived from one declaration** — the model struct
  (embedding `data.Model`) is the single source of truth: schema, generated
  column tokens for the fluent query builder (`db.Posts().Where(P.AuthorID.Eq(uid))…`),
  CRUD (whole-row + optimistic `.version` lock → `core.Conflict`), auto
  timestamps, and relations (`*T`/`[]*T` fields, loaded eagerly via explicit
  `With(...)` — never lazy N+1). Migrations are generated, reviewable,
  per-dialect SQL artifacts applied by an explicit `migrate` step (never at
  boot); destructive changes are surfaced, not guessed. Bounded operator/validation
  vocabulary; parameterized raw SQL stays as the escape hatch. See
  `docs/decisions/0016` (supersedes `0012`).
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
Per-area gap map + stance: `docs/gap.md`. Nothing is currently undecided. Items
deferred to later releases (auth/users, admin/scaffold-gen, agent CLI, i18n/jobs/
mail, tier-3 `{{render}}` boundary) live in `docs/roadmap.md` —
**deferred means "not yet built," never "optional"** (`0015`); reopening any is an
ADR.
