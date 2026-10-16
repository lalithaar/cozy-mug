---
title: "Queries"
description: "Use the fluent query builder and raw SQL."
---

## Operators

Predicates include `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte`, `Like`, `In`,
`NotIn`, `Nil`, and `NotNil`.

## Ordering and paging

```go
db.Table[Note]().OrderBy(noteCols.ID.Asc()).Limit(20).Offset(40).All(ctx)
db.Table[Note]().Page(ctx, 2, 20) // page, size
```

## Raw SQL

Use `Tx` for multi-statement transactions. Parameterized statements are
supported through the query builder; raw SQL remains the escape hatch for
anything outside the fixed operator set.
