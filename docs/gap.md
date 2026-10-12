# Gap analysis — stdlib vs a Rails/Django experience

Purpose: this framework exists to close the gap between Go's primitives and a
"batteries-included" full-stack experience, **without importing Rails/Django's
weight**. This is the map of that gap and our stance per area, under the
frugality filter (`docs/decisions/0001`).

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
- **Partial + hatch** — give ergonomics; escape hatch is raw `net/http` / SQL.
- **Hatch** — helpers only, never a full subsystem.
- **Skip / defer** — out of scope for now; reopening is an ADR.

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
| 8 | Data access | `database/sql` + driver | built (`0012`): `data` package — dialect-inferring `Open`, `All`/`One`/`Exec`/`db.Tx`, `db:"col"` scanner, closure DI; **no ORM** | Partial + hatch |
| 9 | Migrations | none | schema versioning + runner | Fill small |
| 10 | Auth / users | `crypto/*` | users, login/logout, hashing, permissions | Hatch |
| 11 | Assets / static | `embed`, `FileServer` | serving conventions, cache-busting | Minimal + hatch |
| 12 | Errors & logging | `http.Error`, `log/slog` | 404/500 conventions, dev error pages, request log + ID, panic recovery | Fill |
| 13 | Settings | `os.Getenv` | a settings/config convention | Minimal |
| 14 | Testing | `net/http/httptest` | test client, fixtures, DB reset | Partial + agent CLI |
| 15 | Admin / CRUD gen | none | Django admin, `rails generate scaffold` | **Skip** -> generator |
| 16 | Dev console / CLI | none | `manage.py`, `rails console`, generators | Agent CLI (`0002`) |
| 17 | i18n, jobs, mailers | none | translations, background jobs, Mailer | **Defer** / hatch |

## How this maps to work
- Rows 1-7, 9, 12 = the walking skeleton + its conventions (kept minimal).
- Rows 8, 11, 13, 14 = minimal + escape hatches.
- Rows 10, 15, 16 = helpers / generators, never full subsystems.
- Row 17 = deferred; reopening any deferred row is an ADR and may hit the scope
  non-goals (`docs/decisions/0002`).
