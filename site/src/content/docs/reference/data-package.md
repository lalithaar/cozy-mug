---
title: "data package"
description: "Model-first data access for SQLite and Postgres."
---

## Core types

- `data.Model` embedding (table tag names the table).
- `data.Col` and `data.R` tokens.
- `data.DB`, `data.Table[M]`, `data.Tx`.

## Query methods

`Where`, `OrderBy`, `Limit`, `Offset`, `With`, `All`, `One`, `Count`, `Page`,
`Create`, `Update`, `Delete`.

## Schema helpers

- `SchemaSQL()` returns DDL for tests.
- `data.Open(ctx, dsn)` connects SQLite or Postgres based on the DSN.

Relations use `ref:<fk>`; mutations are whole-row with optimistic `Version`.
