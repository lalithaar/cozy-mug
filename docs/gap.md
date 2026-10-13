# Gap analysis — stdlib vs a Rails/Django experience

Purpose: this framework exists to close the gap between Go's primitives and a
"batteries-included" full-stack experience, **without importing Rails/Django's
weight**. This is the map of that gap and our stance per area. **Scope** — what
is owed at all — follows the goal/means split (`docs/decisions/0015`); the
frugality filter (`docs/decisions/0001`) governs only *how* an area is built.

## The core insight
Stdlib gives *primitives*; Rails/Django give *decisions already made*. The real
gap is not features — it's that **stdlib decides nothing**. Rails/Django win by
having opinions; Go's power and its tax are the same thing: every project
re-decides. Our job is to decide the 90%, frugally.

## We start ahead
`html/template` auto-escapes (XSS) and `database/sql` parameterizes (SQLi) by
default — things Django/Rails had to engineer. **Do not rebuild these.**

## Stance legend
- **Fill** — table-stakes for the experience, cheap, serves the 90%.
- **Fill small** — fill it, but the minimal version.
- **Minimal** — a naming/serving convention only; no helpers, no subsystem.
- **Partial + hatch** — give ergonomics; escape hatch is raw `net/http` / SQL.
- **Hatch** — helpers only, never a full subsystem.
- **Skip / defer** — out of scope for now; reopening is an ADR.

This legend rates **weight/ergonomics, not scope**. An area's scope is `0015`'s
call: the essential nutrients (data access, migrations, forms, auth/identity,
settings, sessions, errors + secure defaults) are **in scope and owed** — never
"optional." A lighter word (`Hatch`, `Minimal`, `Fill small`) says *how much*
framework ergonomics we add, not whether the capability exists.

## The gap, by area
| # | Area | Stdlib today | The gap (why it's not Rails/Django) | Stance |
|---|------|--------------|--------------------------------------|--------|
| 1 | Scaffolding & layout | nothing | `rails new` / `startproject` emit a working app + conventional dirs | **Fill** |
| 2 | Routing | `ServeMux` (1.22 patterns) | named routes + reverse URL (`url_for`), grouping, per-route middleware | Fill small |
| 3 | Middleware pipeline | `Handler` chaining | a standard set + ordering convention; request-scoped context | Fill |
| 4 | Sessions & CSRF | `net/http` cookies, `crypto/*` | signed session store + CSRF tokens, wired by default | Fill — built (`0013`): server-side DB store by default, sealed-cookie escape hatch |
| 5 | Views | `html/template` | layouts/partials + loop metadata + arithmetic + formatting + `super` idiom done (opt-in `funcs`, `0011`); urlize/striptags/wordwrap still open | Fill |
| 6 | Forms | `net/url` parsing | bind -> validate -> re-render-with-errors loop | Fill |
| 7 | Global context | `context.Context` | built (`0014`): request-scoped values (`c.Set` / `.Values.name`) + `app.Config.Processors`; CSRF stays `{{.CSRF}}` | Fill |
| 8 | Data access | `database/sql` + driver | built (`0016`, supersedes `0012`): model-first — one struct derives schema, querying (fluent builder over generated column tokens), CRUD (whole-row + optimistic lock → `core.Conflict`), relations (`*T`/`[]*T`, eager `With`); `Table[M]()` + `C`/`R` tokens; schema via `cozy migrate` (`EnsureSchema` removed, `SchemaSQL` for tools/tests); verified on SQLite; connection/dialect/secrets guards carried over | Fill |
| 9 | Migrations | none | designed + runner built (`0016`): `cozy migrate` applies reviewable per-dialect SQL; `EnsureSchema` removed. A model->migration generator is still open; destructive changes surfaced, never auto-guessed. | Fill |
| 10 | Auth / users | `crypto/*` | users, login/logout, hashing, permissions | **Fill** — nutrient (`0015`) |
| 11 | Assets / static | `embed`, `FileServer` | serving conventions, cache-busting | Minimal + hatch |
| 12 | Errors & logging | `http.Error`, `log/slog` | 404/500 conventions, dev error pages, request log + ID, panic recovery | Fill |
| 13 | Settings | `os.Getenv` | a settings/config convention | Minimal |
| 14 | Testing | `net/http/httptest` | test client, fixtures, DB reset | Partial + agent CLI |
| 15 | Admin / CRUD gen | none | Django admin, `rails generate scaffold` | **Skip** -> generator |
| 16 | Dev console / CLI | none | `manage.py`, `rails console`, generators | **Pulled into v0.1** — `cozy migrate` first; generators/console + agent e2e (`0002`) sequenced after |
| 17 | i18n, jobs, mailers | none | translations, background jobs, Mailer | **Defer** / hatch |

## How this maps to work
- Rows 1-7, 12 = the walking skeleton + its conventions (kept minimal).
- Rows 8, 11, 13, 14 = minimal + escape hatches.
- Row 10 (auth/users) is an **essential nutrient** (`0015`): in scope and owed,
  deferred only in sequencing (`docs/roadmap.md`); only its *depth* is open —
  migrations are now designed with the data rework (`0016`). Rows 15, 16 =
  generators, never full subsystems.
- Row 17 = deferred; reopening any deferred row is an ADR and may hit the scope
  non-goals (`docs/decisions/0002`).
