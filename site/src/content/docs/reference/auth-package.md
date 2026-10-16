---
title: "auth package"
description: "Opt-in identity helpers."
---

Import path: `github.com/lalithaar/cozy-mug/auth`.

## Functions

- `HashPassword(password string) (string, error)`
- `CheckPassword(hash, password string) bool`
- `Login(c *core.Context, userID any) error`
- `Logout(c *core.Context) error`
- `UserID(c) (any, bool)`
- `Authenticated(c) bool`
- `UserProcessor(loader UserLoader) core.Processor`

The app owns the `User` model and migrations. `auth` only stores and resolves
identity in the session.
