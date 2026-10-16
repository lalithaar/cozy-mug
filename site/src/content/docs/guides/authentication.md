---
title: "Authentication"
description: "Use the opt-in auth package for identity."
---

The `auth` package is opt-in. Import it and wire it into your app.

## Hash passwords

```go
hash, err := auth.HashPassword(password)
ok := auth.CheckPassword(hash, password)
```

## Log in and out

```go
auth.Login(c, userID)
auth.Logout(c)
```

`Login` regenerates the session to prevent fixation.

## Current user processor

```go
app.Config{
    Processors: []core.Processor{
        auth.UserProcessor(store.CurrentUserLoader()),
    },
}
```

Templates read the current user as `{{.Values.user}}`. Guards can check
`auth.Authenticated(c)` or use `auth.UserID(c)`.
