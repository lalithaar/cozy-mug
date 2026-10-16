---
title: "session package"
description: "Server-side sessions and CSRF stores."
---

- `session.NewStore(db, session.Options{Secure})` (server-side).
- `session.New(session.Options{Key})` (sealed cookie).
- `session.From(c)` reads the request session.
- `Store.Middleware` and `Store.CSRFMiddleware` wire load/persist and enforcement.

Tokens are accessed implicitly through `{{.CSRF}}` and the `X-CSRF-Token` header.
