---
title: "Testing"
description: "Test handlers and apps in-process."
---

Apps implement `http.Handler`, so you can use `httptest` directly.

```go
a := app.Must(app.Config{Assets: assets, Routes: store.Register, Dev: true})
rec := httptest.NewRecorder()
a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
```

Inject a test database with `data.Open(ctx, ":memory:")` and
`SchemaSQL`. For CSRF, capture the session cookie on the GET and submit it again.
