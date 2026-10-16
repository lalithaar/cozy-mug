---
title: "CLI"
description: "Use the cozy CLI to scaffold apps and manage migrations."
---

The `cozy` CLI is standard-library only and ships from `cmd/cozy`.

## cozy new

```bash
cozy new <name> [--dir DIR] [--template example|minimal|auth]
      [--module PATH] [--git] [--skip-tidy]
```

- `--template` chooses `example` (default), `minimal`, or `auth`.
- `--module` overrides the Go module path.
- `--git` runs `git init`.
- `--skip-tidy` skips `go mod tidy`.

With no arguments, `cozy new` interactively asks for the project name, template,
module path, and git initialization.

Run with:

```bash
go run github.com/lalithaar/cozy-mug/cmd/cozy@latest new myapp
```

## cozy migrate

```bash
cozy migrate up [--db DSN] [--dir DIR]
cozy migrate new <name> [source-dir] [--src DIR]
```

- `migrate up` applies pending `*.sql` files from `migrations/<dialect>/`.
- `migrate new` generates migration files for both SQLite and Postgres, deriving
  tables from model structs when it can.

## cozy dev

```bash
cozy dev [--dir DIR] [--port PORT]
```

Rebuilds to `.cozy/dev.exe`, restarts the process on source changes, and prints a
fixed URL. See the dev guide for details.

## cozy version

Prints the CLI version.
