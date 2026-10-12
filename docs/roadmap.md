# Roadmap — release criteria

Source of truth for what gates each tag (`docs/decisions/0008`). A tag is cut
only when its checklist is green. Area stances: `docs/gap.md`.

Gate for **v0.1**: a stranger can go from an empty dir to a deployed, small,
real app following the framework's opinions, with no undecided or undocumented
wall.

## v0.0.1 — the spine (prerelease)

The nucleus, for the record; **no adoption promise** (module path is still the
`cozymug` placeholder).

- [x] Handler contract — `func(*Context) Response` (`0004`)
- [x] Routing — ServeMux wrapper, named/reverse URLs, groups, middleware, canonical slash (`0005`)
- [x] Templating — `html/template`, `{{block}}` layouts + partials, buffered render (`0006`)
- [x] Errors + logging — tiered errors, one boundary, slog request line + request id (`0007`)

## v0.1.0 — first usable release

- [ ] **Project layout / scaffolding** (`gap` 1) — decided (`0009`): conventional dirs, handlers/, views/, static/. TODO: canonical example app at `examples/app` (built; polish + getting-started).
- [x] **App glue** — `app` package (`0009`): `app.New(Config)` wires Logger -> Errors -> Views -> Router, mounts static, exposes an `http.Handler`.
- [x] **Data access** (`gap` 8, `0012`) — `data` package: `Open` infers the dialect (SQLite default / Postgres) and carries both drivers; `All`/`One`/`Exec`/`db.Tx`; `db:"col"` scanner; `One` -> `core.NotFound`; DB by closure DI; injection/secret guards. Example app wired to SQLite. (Getting-started write-up lands with *Identity & docs*.) Migrations stay deferred (`gap` 9).
- [ ] **Global context / processors** (`gap` 7) — current user, flash, CSRF injected into every view.
- [ ] **Forms** (`gap` 6) — bind -> validate -> re-render-with-errors.
- [x] **Sessions + CSRF** (`gap` 4, `0013`) — sealed-cookie sessions (AES-256-GCM, no server store) + CSRF tokens on unsafe methods, on by default via `app.Config.Session`; `c.Session()` / `session.From(c)`, `{{.CSRF}}`; key from `SESSION_KEY`. (`0013` drafted as proposed — pending human sign-off.)
- [ ] **Settings** (`gap` 13) — minimal env-based config convention.
- [ ] **Default UI** (`0003`, `0010`) — architecture decided + `ui` scaffolded; component set and styling TBD.
- [ ] **Template helpers** (`0006`, `0011`) — `funcs` package (opt-in): loop metadata + arithmetic + formatting (text/join/default/pluralize/comma/date/linebreaksbr) done; `super`/nesting idiom documented.
- [ ] **Identity & docs** — module path now `github.com/lalithaar/cozy-mug` (`0009`); TODO: README + getting-started, and a tag with the new path.
- [ ] **Testing convention** (`gap` 14) — documented `httptest` / test-client seam (agent CLI may slip).

### WIP / follow-ups (not release gates)
- **Data-layer verification** (`0012`) — the guards are written but unproven:
  tests for Postgres dialect detection (`detectDialect`), DSN redaction
  (`safeErr`), and `PRAGMA foreign_keys` enforcement; a live Postgres integration
  test gated behind an env var (needs a server/container).
- **Supply chain** (`0001`, `0012`) — run `govulncheck` in CI and keep the two
  drivers (`modernc.org/sqlite`, `pgx/v5`) current.

### Deferred past v0.1 (reopening = ADR)
- Migrations (`gap` 9), auth/users (`gap` 10, hatch), admin/scaffold-gen (`gap` 15),
  agent CLI (`gap` 16, `0002`), i18n/jobs/mail (`gap` 17), tier-3 `{{render}}`
  boundary (`0007`), asset cache-busting (`gap` 11), contrast-aware theming +
  default themes (`docs/ideas.md`).
