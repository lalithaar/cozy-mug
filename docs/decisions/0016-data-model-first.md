# 0016 — Data access rework: model-first source of truth, derived schema & queries

- Status: firm
- Date: 2026-10-13
- Supersedes: `0012`

## Opinion

The data layer is rebuilt around a **single Go declaration** — the model struct —
as the one home for an entity's shape. From that declaration the framework
*derives* everything: the schema, the migrations, the query surface, and (later)
forms + validation. The developer **never maintains a second representation** of
an entity — no hand-written DDL, no hand-mirrored struct, no hand-written CRUD.

```go
type Post struct {
	data.Model              // identity marker; carries pk + conn for navigation
	ID        int64         `db:"id auto"`
	Title     string        `db:"title" required min:"3" max:"200"`
	Body      string        `db:"body"`
	AuthorID  int64         `db:"author_id" index`
	Author    *User         `db:"ref:author_id"`  // belongs-to / has-one
	CreatedAt time.Time     `db:"created_at"`
}
```

- **One declaration, two tag surfaces** that share the field but never the value:
  schema qualifiers (`db:"col" auto index unique ref:...`) and bare validation
  rules (`required`, `min`, `max`, `email`, `url`, `oneof:a,b`). A `unique`
  index and a `required` rule are different enforcements; being on the same
  field does not merge them.

- **Querying is a fluent builder over generated column tokens.** Fields are
  compile-checked — `Post.CreatedAt` as a *token*, never `"created_at"` as a
  string. Tokens are emitted from the model once (`go:generate`); the model stays
  the source of truth, so codegen sharpens `A`, it is not schema-first.

```go
posts, err := db.Table[Post]().
	With(P.Author).                      // eager batch-load the relation
	Where(P.AuthorID.Eq(uid)).
	OrderBy(P.CreatedAt.Desc()).
	Limit(10).
	All(ctx)                             // []Post, relations already loaded
```

  The entry point is a generic call — `db.Table[Post]()` rather than a sketched
  `db.Posts()` — because Go has no external receivers: a per-model method on
  `data.DB` is impossible. The generic one-loses-nothing: tokens keep every
  column reachable and compile-checked. (`P` above is a declared token struct —
  `data.C[Post]("author_id")` — emitted by codegen, hand-written for now.)

- **Relations are declared fields, loaded eagerly, in batches, opt-in.**
  `*T` = belongs-to/has-one, `[]*T` = has-many (the reverse side). Following a
  relation is free because `.With(name)` prefetched it for the whole row set in
  one query. **Lazy loading is rejected**: `p.Author` firing a query per row is
  the N+1 trap, and the framework never introduces it silently.

- **Mutations are whole-row, guarded by an optimistic lock.**
  `Create(ctx, &post)` inserts and back-fills pk + timestamps; `Update(ctx,
  &post)` rewrites the loaded row under `WHERE id = ? AND version = ?`, bumping
  the auto `.version` column; a stale write returns `core.Conflict` so the
  handler re-renders "edited elsewhere, reload." `Delete(ctx, &post)` by pk. No
  dirty-tracking: it cannot fix lost updates (only a lock can), so it would pay
  per-row runtime for the wrong problem.

- **Timestamps are a convention**: `created_at` / `updated_at` are maintained
  automatically when present.

- **Migrations are generated, reviewable, versioned SQL artifacts.** The
  generator diffs the model against the last applied schema state and emits
  numbered migration files — one `sqlite/` and one `postgres/` variant, so the
  file a DBA reviews is literally the SQL that runs. `cozy migrate` applies
  pending files in order and records them in `schema_migrations`. **The happy
  path never writes DDL.** Additions generate; destructive changes (drop,
  rename, type change) are *surfaced*, never guessed, and become the one place a
  developer writes a manual migration. **Never auto-migrate at boot** — `migrate`
  is an explicit deploy step. Forward-only; no auto-rollback (that is how data
  is lost).

  **Invocation is the CLI — firm.** `migrate` runs through the CLI (`cozy
  migrate`), never an app-invoked command in `main.go`, never a library call at
  boot. Until that CLI (and the artifact generator) exists, `EnsureSchema` is
  the interim boot-time creator: model-derived DDL applied with `IF NOT EXISTS`
  only — create-only, idempotent, no versioning, no destructive op. It is
  **deleted** when `cozy migrate` lands; the two never coexist for app tables
  (a fragment of the interim haunting the production path is the tangle — the
  cutover is removal, not "also do migrations"). Whether framework-owned tables
  (`sessions`) stay outside app migration files is still open.

- **Every builder op compiles to parameterized SQL** (`SQL values are parameters;
  SQL identifiers are code`). Tokens make identifiers a fixed, generated set —
  the heuristic becomes *structural* instead of a discipline. Raw, parameterized
  SQL stays as the escape hatch for the 5% builders do badly (aggregates, window
  functions, exotic joins).

- **Bounded surfaces, denied by default beyond the line:** operators
  (`Eq, Ne, In, NotIn, Nil, NotNil, Lt, Lte, Gt, Gte, Like`), validation rules
  (`required, min, max, email, url, oneof`; no `regexp` — custom logic is a
  `Validate() error` method on the model), paging (`.Page(page, per)` returns
  rows + total count; listing pages need the count). Soft delete is **declined**
  — real machinery, and the escape hatch is a manual table — not an omission.

## Because

Tested against `0015`'s nutrient test: for any entity the previous layer
(`0012`) left the developer maintaining **four hand-synced copies** — the
`CREATE TABLE`, the struct + tags, the query string per access, and the arg
lists. Every one can silently drift from every other. That is not frugality; it
is the exact translation tax Rails/Django built a whole layer to delete, and
`0015` names the absence of an essential as a **wall**. The frictions we remove
are not conveniences; they are the product.

`0012`'s reasoning was sound *under* `0001`'s framings. `0015` re-ranks:
frugality shapes the implementation; it does not settle whether the capability
exists. The connection wrapper, parameterization, dialect support, scanner
discipline, and the two SQL/secrets heuristics all survive — carried forward,
not discarded. What changes is where the developer writes code: **one Go file
per entity, nothing else.** Accepting the traded-off weight: the framework grows
a schema-derivation generator (the "compiler" `0012` declined), queries become
slightly more indirect than raw SQL, and migration diffing must run at dev time.
The cost of absence — a wall on every real app — exceeds all of it.

## Alternatives rejected

- **`0012` as-is** — removes friction but not translation; four copies of every
  entity. Superseded.
- **Schema-first + codegen** — migrations/hand SQL are the source of truth.
  Raw schema maintenance is brittle to hand-drift and painful to migrate over
  time; rejected on DX.
- **Fluent builder with string column names** — the typo/rename tax, precisely
  the drift we are deleting. Generated tokens exist to make it impossible.
- **Query-by-example (struct literal as predicate)** — clean Go, but
  equality-only; dies at the first `!=`, range, or NULL. Kept as a *shortcut*
  for pure-equality matches, never as the surface.
- **Lazy relation proxies** — N+1 by design; rejects the reason relations were
  wanted.
- **Dirty-tracking updates** — pays runtime to update only changed columns, but
  cannot prevent lost updates; the optimistic lock does that, and whole-row
  update on a fully-loaded entity is correct.
- **Live boot-time migrate-diff** (`data.Migrate(ctx)`) — spooky, nondeterministic
  across two dialects, and the classic "prod tables diverged" footgun. Migrations
  are reviewable artifacts and an explicit step.
- **Dirty/soft-delete default, `regexp` validation tag** — unbound surface;
  denied by default.

## Apply when

Any handler or loader reads or writes a database; any entity that needs a
schema, CRUD, a listing page, or validation in a form.

## Deviate / ask when

- Relations beyond the declared `*T` / `[]*T` shapes (join tables with payload,
  polymorphic FKs, etc.) — reopen this.
- A migration shape the generator cannot derive (rename, drop, column-type
  change) — the generator must surface it; the human writes the manual file.
- A need outside the operator/validation/paging caps and the raw-SQL escape
  hatch — new ADR, not a tag.
- **Never** interpolate request data into SQL text; dynamic identifiers come
  from the fixed (model-derived) token set.

## Revisit if

The generator's SQL opacity becomes the real tax — i.e. developers stop
reading the emitted migrations and trust the model blindly — then making review
louder (or render-vs-diff reporting) is on the table. Also if evidence emerges
that a real workload needs partial updates (a genuine measured case, not a
preference) — the whole-row + lock default is open to a *bounded* `Set` surface.

## Consequences

- `0012` is **superseded**; its connection, safety, and dialect discipline
  carries into the new layer.
- Gap rows 8 (data access) and 9 (migrations) are now **designed**; building
  them is the same rework.
- The `db:` tag vocabulary becomes a **stable API** — forms/validation (gap 6)
  will read the same declaration, so its semantics are load-bearing now.
- Heuristic `SQL values are parameters; SQL identifiers are code` is
  strengthened: identifiers are a *generated fixed set*, enforced by tokens.
- Status **soft** (per meta-rule): all forks were human-directed in design
  discussion; promote to **firm** on sign-off of this written ADR.