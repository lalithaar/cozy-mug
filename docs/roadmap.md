# Roadmap — release criteria

Source of truth for what gates each tag (`docs/decisions/0008`). A tag is cut
only when its checklist is green. Area stances: `docs/gap.md`.

This file is superseded-in-place: completed items stay green, in-flight items
move to WIP, and new proposals from the vault-app build live in the
**Post-v0.1 proposals** section below with their implementation status checked
against the codebase. Nothing is deleted; old items are marked superseded with a
link to the new entry.

Gate for **v0.1**: a stranger can go from an empty dir to a deployed, small,
real app following the framework's opinions, with no undecided or undocumented
wall.

## v0.0.1 — the spine (prerelease)

The nucleus, for the record; **no adoption promise** (module path is still the
`cozymug` placeholder).

- [x] Handler contract — `func(*Context) Response` (`0004`)
- [x] Routing — ServeMux wrapper, named/reverse URLs, groups, middleware, canonical slash (`0005`)
- [x] Templating — `html/template`, `{{block}}` layouts + partials, buffered render (`0006`)
- [x] Errors + logging — tiered errors, one boundary, slog request line + request id (`0007`); tier-3 `{{render}}` boundary deferred (`0007`)

## v0.1.0 — first usable release

- [x] **App glue** — `app` package (`0009`): `app.New(Config)` wires Logger ->
  Errors -> Session -> Processors -> Views -> CSRF -> Router, mounts static,
  exposes an `http.Handler`.
- [x] **Project layout / scaffolding** (`gap` 1, `0009`) — conventional dirs
  (`main.go`, `handlers/`, `views/`, `static/`). Canonical example app lives at
  `examples/app`; `cozy new` scaffolds from embedded templates (`minimal`,
  `example`, `auth`).
- [x] **Data access rework** (`gap` 8–9, `0016`, supersedes `0012`) —
  model-first: one struct derives schema + queries (generated tokens) + CRUD
  (whole-row, optimistic `.version` lock → `core.Conflict`) + relations
  (`*T`/`[]*T`, eager `With(...)`); auto timestamps; reviewable per-dialect
  (sqlite/postgres) migration artifacts + an explicit `migrate` step. The `data`
  package was rebuilt (`db.Table[M]()` entry point, `data.C`+`data/R` tokens,
  `Query` pipe; app tables now come only from `cozy migrate`, `EnsureSchema`
  removed — `SchemaSQL` is the exported generator entry point for tools/tests);
  verified on SQLite, example rewired + seeded, guarded by `data/model_test.go`.
  The model→migration generator ships: `cozy migrate new` discovers structs
  embedding `data.Model` in the source tree and emits per-dialect DDL (falls
  back to a stub when none are found); + a live Postgres integration test
  (env-gated in `data.PostgresDataLayer` test + CI `postgres` job). The safe
  guards (parameterization, dialect inference, DSN scrubbing, secret rules)
  carried over.
- [x] **CLI** (`gap` 16) — built: `cozy new` scaffolds apps and `cozy migrate`
  (`up` + `migrate new`) applies reviewable per-dialect SQL
  (`migrations/<dialect>/*.sql`). Non-migration subcommands (generators/console,
  agent e2e `0002`) are sequenced after but share the binary.
- [x] **Global context / processors** (`gap` 7, `0014`) — request-scoped values
  (`c.Set` / `{{.Values.name}}`) plus `app.Config.Processors`
  (`core.Processor`), run after sessions and before views; CSRF stays
  `{{.CSRF}}`. Example uses a processor for flash.
- [x] **Forms** (`gap` 6, `0017`) — `form` package: bind -> validate (tag rules
  + model hook) -> re-render-with-errors; raw repopulation; seeded Forms for
  GET/edit.
- [x] **Auth** (`gap` 10, `0018`, `0019`) — opt-in `auth` package: PBKDF2
  password hashing, session-backed login/logout, current-user processor, route
  guards, in-memory lockout, verified password change. User model stays app-owned
  per `0016`; password reset and verification remain mailer-blocked.
- [x] **Sessions + CSRF** (`gap` 4, `0013`) — server-side sessions by default
  (`session.NewStore(db)`, framework-owned `sessions` table, opaque id cookie,
  revocable, no global key) with a sealed-cookie escape hatch (`session.New`);
  CSRF tokens required on unsafe methods; on by default via
  `app.Config.Session`; `c.Session()` / `session.From(c)`, `{{.CSRF}}`.
- [x] **Settings** (`gap` 13) — built: minimal env-based config convention
  (`settings` package: `DATABASE_URL`, `APP_ENV`/`DEV`, `PORT`).
- [x] **Default UI** (`0003`, `0010`) — architecture decided + `ui` scaffolded;
  styling provided by a vendored copy of inherent.css
  (`ui/static/base.css`, served at `/static/ui/base.css`); semantic HTML is the
  component set, no extra component library needed.
- [x] **Template helpers** (`0006`, `0011`) — `funcs` package (opt-in): loop
  metadata + arithmetic + formatting
  (text/join/default/pluralize/comma/date/linebreaksbr); `super`/nesting idiom
  documented.
- [x] **File uploads** (`0021`, soft) — `upload` package: `upload.Local`
  filesystem backend, `Save`/`SaveFile`/`Open`/`Delete`/`URL`/`Handler`,
  dangerous-extension rejection, `app.Config.Uploads` mount at `/uploads/`.
- [x] **Flash** (`0020`, firm) — opt-in `flash` package: one-shot session
  messages consumed by the existing processor/render seam; no new middleware
  ordering needed.
- [ ] **Identity & docs** — module path now `github.com/lalithaar/cozy-mug`
  (`0009`); TODO: README + getting-started polish, and a tag with the new path.
- [ ] **Testing convention** (`gap` 14) — documented `httptest` / test-client
  seam (agent CLI may slip).

### WIP / follow-ups (not release gates)
- **Data-layer verification** (`0016`) — `EnsureSchema` removed from the app
  path; schema now arrives via `cozy migrate` (app tables) / `SchemaSQL` for
  tools+tests. Security guards covered by tests
  (`data/security_internal_test.go`, `data/data_security_test.go`). Live Postgres
  integration test env-gated, runs in CI `postgres` job.
- **Supply chain** (`0001`, `0012`) — CI runs `go vet`, `go test`, `go build`,
  `govulncheck`; drivers current; toolchain go1.27.1 (CI pins `1.27.2`).

---

## Post-v0.1 proposals — feature requests from the vault-app build

The vault app (`tmp/vault-app/`) surfaced ergonomic gaps that `cozy-mug` leaves
to the app author. These are **new proposals**, not v0.1 gates. Each entry
lists its status against the existing codebase, whether it reopens a firm
decision (needs ADR + sign-off), and where it sequences.

| # | Area | Proposal | Status vs codebase | Decision impact |
|---|------|----------|-------------------|-----------------|
| P1 | Scaffolding & code generation | `cozy generate scaffold <Name> <cols>` and `cozy generate model <Name> <cols>` emit model struct + column tokens + migration SQL + handler stubs + views. | **Not built.** `cozy gen` exists internally in `cmd/cozy/gen.go` (model discovery + DDL render) but is not exposed as a user-facing subcommand; `cozy new` only scaffolds from embedded templates. | Reopens gap row 15 (skip -> generator) and row 16 (generators sequenced after v0.1). **Soft**; needs ADR + sign-off. |
| P2 | Routing | Resourceful routing: `r.Resource("people", PeopleController{})` conventionally maps index/show/new/create/edit/destroy with named routes + reverse URLs, plus nested resources. | **Not built.** `router.Router` supports named routes + groups + middleware; resource conventions are hand-written per app (`gap` 2 = Fill small). | Additive to `0005`; does not reopen firm decisions. **Soft**. |
| P3 | Dev-only auto-schema | Opt-in `app.Config.AutoMigrate` (dev only) applies model-derived DDL on boot so `cozy dev` / tests don't need manual `SchemaSQL()` exec. | **Not built; deviates from `0016`.** `0016` is firm: "DDL at boot is not; app tables arrive only through `cozy migrate`." | **Reopens `0016`.** Needs ADR + human sign-off; scoped to `Dev: true` only, never production. |
| P4 | Relation loading defaults | Default eager loading for declared `belongs-to` relations on `All`/`One` (opt-out via `.Without(...)`), so listing pages don't need `.With(...)` per parent. | **Not built.** `data` requires explicit `.With(data.R[M]("Field"))`; relations load only when asked (`gap` 8). | Additive to `0016`; no decision reopen. **Soft**. |
| P5 | Seeds / fixtures | `cozy seed` command + conventional `seeds/` directory run after migrations so a fresh app isn't empty on first launch. | **Not built.** `store.Seed(ctx)` exists as a hand-rolled per-app method on the example; no framework seed subsystem. | New subsystem; additive. **Soft**. |
| P6 | First-class flash + form field partials | Built-in `flash` package (decided `0020`) + form field partials (`{{field "name" .Form}}`) rendering label/input/value/errors together. | **Partial.** `flash` package exists (`flash/flash.go`, `0020` firm implemented) but is not wired into `app.Config` by default; form field partials do not exist. | `0020` already firm; form partials are new UI-layer ergonomics. **Soft**. |
| P7 | `cozy console` | REPL that boots the app's `main.go` context (DB, store, models) for interactive `Person{}.All(ctx)` style exploration. | **Not built.** Gap row 16 lists console sequenced after generators; nothing in `cmd/cozy`. | New CLI subcommand; shares `cmd/cozy` binary. **Soft**. |
| P8 | Test helpers | `test.NewApp(t, routes, models...)` stand up an in-memory DB, apply schemas, create sessions, and build the app so integration tests are ~10 lines instead of ~100. | **Not built.** Gap row 14 = Partial + agent CLI; `vault-app/main_test.go` is ~120 lines of manual setup. | New `test` package; additive seams. **Soft**. |
| P9 | Conventional auth wiring | `current_user` and `authenticate!` conventional in every handler/template when sessions are configured; `require_login` as one-line middleware rather than custom per-app. | **Partial.** `auth` package exists (`0018`, `0019`) with `auth.UserProcessor` and `auth.RequireLoginMiddleware`; vault app still wrote a custom `requireLogin`. | Additive documentation + small helpers on top of `0018`. **Soft**. |

### Execution plan

The plan below sequences work so that the highest-leverage, lowest-risk items
ship first. Items marked **(ADR)** require a decision document + human sign-off
before implementation because they touch firm opinions (`0016`, `0015`).

#### Phase 1 — Test seams & generators (no firm-decision reopen)
Goal: make the framework easier to test and extend without changing release
behavior.

1. **P8 Test helpers** — add `test` package with `NewApp` helper (in-memory DB,
   `SchemaSQL()` exec, sessions, app build). Reduces integration-test boilerplate
   across the framework and apps.
2. **P1 Code generation** — expose `cmd/cozy/gen.go` internals as user-facing
   `cozy generate model` and `cozy generate scaffold`. Reuses existing model
   discovery + DDL render; emits `migrations/sqlite/` + `migrations/postgres/`
   and handler/view stubs. **(ADR)** — reopens gap row 15/16; generator surface
   is new exported CLI area.
3. **P2 Resourceful routing** — add `r.Resource(...)` convention to `router`,
   sugar over existing named routes + groups.

#### Phase 2 — Dev ergonomics (one firm-decision reopen)
Goal: remove the manual schema boilerplate that `cozy dev` and tests require.

4. **P3 Opt-in auto-schema** — `app.Config.AutoMigrate` (dev only). **(ADR)** —
   reopens `0016` ("DDL at boot is not"). Scoped strictly to `Dev: true`;
   production remains migration-only. Human sign-off required.
5. **P4 Relation loading defaults** — default eager `belongs-to` on `All`/`One`
   with `.Without(...)` opt-out. Additive to `0016`.

#### Phase 3 — App authoring ergonomics
Goal: cut the boilerplate the vault app hand-wrote.

6. **P5 Seeds / fixtures** — `cozy seed` + `seeds/` directory; runs after
   migrations.
7. **P6 Form field partials** — built-in `{{field}}` partial; `flash` package
   wired into `app.Config` by default when sessions are present.
8. **P9 Conventional auth wiring** — one-line `require_login` helper + documented
   `current_user` pattern on top of `0018`.

#### Phase 4 — Interactive tooling
Goal: match Rails/Django's interactive story.

9. **P7 `cozy console`** — REPL booting the app context. Shares `cmd/cozy`
   binary; depends on Phase 1 test seams for app bootstrap.

### Definition of done per proposal
- [ ] ADR written (if decision impact says ADR) + human sign-off
- [ ] Code implemented + `go build`/`go vet`/`gofmt` green
- [ ] Unit tests + integration test through `httptest`
- [ ] `docs/gap.md` row updated with new stance
- [ ] `docs/idioms.md` updated with the how-to
- [ ] Starlight docs page added under `site/src/content/docs/`
- [ ] LOG entry appended: date + what changed + why

### What stays out of scope (firm exclusions, `0002`)
- No ORM / ActiveRecord replacement; `data.Table[M]()` remains the boundary.
- No API / SPA / REST support; SSR-only.
- No automatic migrations in production (even behind `AutoMigrate`, which is dev-only).
- No config files; environment variables remain the whole story (`settings`).
- i18n, jobs, mailers remain deferred (`gap` 17).
