# 0023 — Dev-only auto-schema (opt-in, supersedes part of 0016 for development)

- Status: firm (implemented, dev-only)
- Date: 2026-10-16
- Supersedes: part of `0016` ("DDL at boot is not") — narrowed to production only.

## Opinion

`app.Config.AutoMigrate` (default `false`) applies model-derived DDL (`data.Table[M]().SchemaSQL()`) on boot **only when `Dev` is true**. It is never applied in production. This removes the manual `SchemaSQL()` exec step from `cozy dev` and test setup while keeping production strictly migration-driven (`cozy migrate`).

## Because

`0016` correctly keeps production schema explicit and reviewable via migrations. But during development and in tests, re-applying `SchemaSQL()` after every model change is friction that slows the `cozy dev` loop and bloats test setup (see P8 test helpers, which still apply schema explicitly for clarity). Since `AutoMigrate` is gated on `Dev: true` and cannot be enabled in production (`app.New` errors if `AutoMigrate && !Dev`), the production guarantee of `0016` is intact: app tables in production arrive only through `cozy migrate`.

## Alternatives rejected

- **Enable auto-schema in all modes** — rejected; violates `0016` and the reviewable-migration guarantee.
- **Require `cozy migrate up` in dev too** — retained as the production path; dev auto-schema is a convenience, not a replacement.
- **Auto-schema via `EnsureSchema` at boot** — rejected; `EnsureSchema` was explicitly removed (`0016`) because silent DDL at boot is opaque.

## Apply when

`app.Config{ Dev: true, AutoMigrate: true }` during local development or test bootstrapping.

## Deviate / ask when

A change would enable auto-schema in production, apply migrations automatically, or remove the `cozy migrate` step from production deployment. That reopens `0016` and requires a new ADR + sign-off.
