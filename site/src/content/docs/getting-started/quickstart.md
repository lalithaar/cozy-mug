---
title: "Quickstart"
description: "Scaffold a cozy-mug app and run it in a few minutes."
---

cozy-mug targets **server-rendered HTML apps**. This quickstart takes you
from an empty directory to a running app.

## Prerequisites

- Go 1.27 or newer.
- Git.

## Run the bundled example

The repository ships `examples/app`, a small notes app. Run it from the repository
root after applying its migration:

```bash
go run ./cmd/cozy migrate up --dir examples/app/migrations
go run ./examples/app        # http://localhost:8080
```

Set `DATABASE_URL=postgres://...` to use Postgres instead of SQLite. The example
reads the DSN from the environment and defaults to `app.db`.

## Scaffold a new app

Create a named app anywhere:

```bash
go run github.com/lalithaar/cozy-mug/cmd/cozy@latest new myapp
cd myapp
go run .                     # http://localhost:8080
```

For the full interactive flow, run `cozy new` with no arguments. It asks for the
project name, template, module path, and whether to initialize git.

### Templates

- `example` (default): the notes app with data access, forms, sessions, CSRF, and flash.
- `minimal`: a hello-world page using the optional UI package only.
- `auth`: the example app plus the opt-in auth package (dummy account support included).

## Add a migration

If you use the example template, generate your first model-driven migration:

```bash
cozy migrate new init .
cozy migrate up
```

## Try dev mode

```bash
cozy dev            # rebuild -> restart -> browser refresh
```

In development, cozy-mug injects a tiny polling script into HTML responses and
replaces the server process on any source change. It builds into a stable
`.cozy/dev.exe` path so Windows does not prompt on every run.

## Non-goals

cozy-mug is not an API backend for JavaScript SPAs, not a REST framework, and
not a general-purpose ORM. See the README on GitHub for the full stance.
