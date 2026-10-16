---
title: "Sessions and CSRF"
description: "Server-side sessions and CSRF protection, on by default."
---

cozy-mug ships server-side sessions. The cookie holds only an opaque id; a
small JSON map lives in a framework-owned `sessions` table. That makes sessions
revocable and removes the need for a global secret.

## Wiring

```go
sessions := core.Must(session.NewStore(db, session.Options{
    Secure: os.Getenv("DEV") == "0",
}))
core.Must(sessions.EnsureSchema(ctx))

a := app.Must(app.Config{
    Session: sessions,
})
```

## Reading a session

```go
sess := session.From(c)
sess.Set("flash", "Note added.")
msg := sess.GetString("flash")
```

## CSRF

Unsafe methods require a token in form field `csrf` or header `X-CSRF-Token`.
Render it with:

```html
{{.CSRF}}
```

The token is compared in constant time.

## Sealed-cookie store

Use `session.New(session.Options{Key: key})` when you prefer stateless cookies.
This requires a 32-byte key from `SESSION_KEY`.
