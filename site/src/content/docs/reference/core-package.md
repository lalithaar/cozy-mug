---
title: "core package"
description: "Request context, responses, errors, and logging."
---

## Context

`core.Context` carries `W`, `R`, `Router`, `Views`, and request-scoped values.
Methods include `Render`, `Error`, `Param`, `Query`, `Form`, `Set`, `Global`.

## Responses

- `core.String(status, body)`
- `core.Text(status, body)`
- `core.Redirect(status, url)`
- `core.Error(code, msg)`
- `core.Status(code)`

## Errors and middleware

- `core.Errors(core.Options)` recovers panics and renders error pages.
- `core.Logger(core.Options)` assigns request ids and logs one line.
- `core.Processors(...)` installs global value processors.
- `core.HTTPError`, `core.NotFound`, `core.Conflict` carry status codes.
