---
title: "Project structure"
description: "The conventional layout of a cozy-mug app."
---

An app follows a small, fixed layout that the `app` package assembles for you.

## Files and folders

```text
myapp/
  main.go        # tiny: build the app, run it
  handlers/      # handlers, loaders, and view models
  views/         # templates: layouts/, partials/, errors/
  static/        # assets, served under /static/
  migrations/    # generated SQL artifacts (optional, for data apps)
  go.mod
```

## What each part does

- **`main.go`** opens the database, creates a session store, and calls `app.Must`.
- **`handlers/`** contains feature code. Each handler has the signature
  `func(*core.Context) core.Response`. Loaders are plain Go functions that return
  `(value, error)`.
- **`views/`** uses Go `html/template` layouts and partials. Files under `layouts/`
  and `partials/` are shared; every other `*.html` file is a view.
- **`static/`** is mounted at `/static/` when present.
- **`migrations/`** holds SQL artifacts produced by `cozy migrate new` and applied
  by `cozy migrate up`.

## Environment

Configuration is environment-based (12-factor). The important variables are:

- `DATABASE_URL` (defaults to `app.db`).
- `APP_ENV=production` or `DEV=0` (disables dev mode and secure cookies off).
- `PORT` (defaults to `8080`).
- `SESSION_KEY` (base64, 32 bytes, only for the sealed-cookie store).

## Keep it small

Core never depends on the optional `ui`, `funcs`, or auth packages. You opt in by
importing them and wiring them in `app.Config`.
```<|end|><|start|>write to=assistant<|channel|>analysis<|message|>Wrote file successfully: site/src/content/docs/getting-started/project-structure.md
