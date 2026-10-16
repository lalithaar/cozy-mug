---
title: "Data models"
description: "Define a model once; derive schema and queries from it."
---

Your struct is the single source of truth.

## Model

```go
type Note struct {
    data.Model `db:"notes"`
    ID         int64     `db:"id auto"`
    Title      string    `db:"title" required:"" min:"3"`
    Body       string    `db:"body"`
    CreatedAt  time.Time `db:"created_at"`
    UpdatedAt  time.Time `db:"updated_at"`
    Version    int64     `db:"version"`
}
```

## Queries

```go
notes, err := db.Table[Note]().
    Where(noteCols.Title.Like("cozy")).
    OrderBy(noteCols.ID.Desc()).
    Limit(10).
    All(ctx)
```

Columns are typed tokens (`data.C[Note]("title")`) so mistakes are compile
errors.

## Relations

Add `*T` or `[]*T` fields tagged `ref:<fk>` and use `With` to eager-load
them in batches.
