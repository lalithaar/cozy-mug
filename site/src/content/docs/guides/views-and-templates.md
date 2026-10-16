---
title: "Views and templates"
description: "Block layouts, partials, and buffered rendering."
---

cozy-mug uses standard `html/template` (auto-escaping). Layouts define blocks,
views override them.

## Convention

- `views/layouts/` holds shared layouts.
- `views/partials/` holds partials.
- Every other `*.html` is a view.

`c.Render(status, "home", data)` fills an embedded `Globals` field
automatically, so templates can build reverse URLs with `{{.URL}}`.

## Partials

Factor shared regions into partials and call them directly:

```html
{{template "note-body" $n}}
```

## Template helpers

Opt in with `Config.Funcs`. The `funcs` package adds loop metadata, arithmetic,
pluralization, and formatting. See `funcs/funcs.go` for the full list.

## Dev reload

In development, cozy-mug injects a script that polls `/__cozy/dev-build` and
reloads the browser when the server restarts.
