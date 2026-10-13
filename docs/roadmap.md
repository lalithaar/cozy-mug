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
- [x] **App glue** — `app` package (`0009`): `app.New(Config)` wires Logger -> Errors -> Session -> Processors -> Views -> CSRF -> Router, mounts static, exposes an `http.Handler`.
- [ ] **Data access rework** (`gap` 8–9, `0016`, supersedes `0012`) — model-first: one struct derives schema + queries (generated tokens) + CRUD (whole-row, optimistic `.version` lock → `core.Conflict`) + relations (`*T`/`[]*T`, eager `With(...)`); auto timestamps; reviewable per-dialect (sqlite/postgres) migration artifacts + an explicit `migrate` step. The `data` package was rebuilt (`db.Table[M]()` entry point, `data/C`+`data/R` tokens, `Query` pipe; app tables now come only from `cozy migrate`, `EnsureSchema` removed — `SchemaSQL` is the exported generator entry point for tools/tests); verified on SQLite, example rewired + seeded, guarded by `data/model_test.go`. Remaining: a model→migration *generator* (today `cozy migrate new` writes stubs you fill, or copy from `SchemaSQL`) + a live Postgres test. The safe guards (parameterization, dialect inference, DSN scrubbing, secret rules) carried over.
- [x] **CLI** (`gap` 16) — built: `cozy new` scaffolds apps and `cozy migrate` (`up` + `migrate new`) applies reviewable per-dialect SQL (`migrations/<dialect>/*.sql`). Non-migration subcommands (generators/console, agent e2e `0002`) are sequenced after but share the binary.
- [x] **Global context / processors** (`gap` 7, `0014`) — request-scoped values (`c.Set` / `{{.Values.name}}`) plus `app.Config.Processors` (`core.Processor`), run after sessions and before views; CSRF stays `{{.CSRF}}`. Example uses a processor for flash.
- [x] **Forms** (`gap` 6) — `form` package: bind -> validate (tag rules + model hook) -> re-render-with-errors; raw repopulation; seeded Forms for GET/edit. See `docs/decisions/0017`.
- [x] **Sessions + CSRF** (`gap` 4, `0013`) — server-side sessions by default (`session.NewStore(db)`, framework-owned `sessions` table, opaque id cookie, revocable, no global key) with a sealed-cookie escape hatch (`session.New`); CSRF tokens required on unsafe methods; on by default via `app.Config.Session`; `c.Session()` / `session.From(c)`, `{{.CSRF}}`.
- [x] **Settings** (`gap` 13) — built: minimal env-based config convention (`settings` package: `DATABASE_URL`, `APP_ENV`/`DEV`, `PORT`).
- [ ] **Default UI** (`0003`, `0010`) — architecture decided + `ui` scaffolded; component set and styling TBD.
- [x] **Template helpers** (`0006`, `0011`) — `funcs` package (opt-in): loop metadata + arithmetic + formatting (text/join/default/pluralize/comma/date/linebreaksbr) done; `super`/nesting idiom documented.
- [ ] **Identity & docs** — module path now `github.com/lalithaar/cozy-mug` (`0009`); TODO: README + getting-started, and a tag with the new path.
- [ ] **Testing convention** (`gap` 14) — documented `httptest` / test-client seam (agent CLI may slip).

### WIP / follow-ups (not release gates)
- **Data-layer verification** (`0016`) — `EnsureSchema` removed from the app
  path; schema now arrives via `cozy migrate` (app tables) / `SchemaSQL` for
  tools+tests. The security guards carried from `0012` (dialect detection, DSN
  redaction, `PRAGMA foreign_keys` enforcement) are still unproven by tests; a
  live Postgres integration test is env-gated (needs a server/container). The
  model layer itself is SQLite-verified in `data/model_test.go`.
- **Supply chain** (`0001`, `0012`) — `.github/workflows/ci.yml` runs `go vet`,
  `go test`, `go build`, and `govulncheck` (golang/govulncheck-action). Drivers
  kept current (`modernc.org/sqlite v1.60.1`, `pgx/v5 v5.11.0`); `golang.org/x/text`
  bumped to v0.39.0. Local toolchain is go1.27.1, which carries stdlib advisories
  fixed in 1.27.2 — CI pins `go-version: "1.27"` (toolchain).

### Deferred past v0.1 — sequencing, not scope (reopening = ADR)
**Deferred means "not yet built," never "optional."** The nutrients among these
(migrations, auth) are in scope (`0015`); this section orders work, it does not
mark anything discretionary.
- Auth/users (`gap` 10), admin/scaffold-gen (`gap` 15), i18n/jobs/mail (`gap` 17),
  tier-3 `{{render}}` boundary (`0007`), asset cache-busting (`gap` 11).
  (Migrations **and the CLI** were deferred here; `0016` designs the migrations
  and both are now **pulled into v0.1** — `cozy migrate` first.)

### Ideas (not commitments)
- Captured-but-undecided work — e.g. contrast-aware theming + default themes —
  lives in `docs/ideas.md`; it is **not** in scope and **not** sequenced. (`ui`
  is a scaffold: no components/themes are decided yet.)
