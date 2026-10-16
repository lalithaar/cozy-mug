---
title: "Handlers and responses"
description: "Value-returning handlers and the Response contract."
---

Handlers return a value instead of writing directly:

```go
func (s *Store) showNote(c *core.Context) core.Response {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        return c.Error(core.NotFound(err))
    }
    n, err := s.noteByID(c.R.Context(), id)
    if err != nil {
        return c.Error(err)
    }
    return c.Render(http.StatusOK, "notes/show", &noteView{Note: n})
}
```

## Responses

- `c.Render(status, "view", data)` renders a view into a buffer, then returns it.
- `core.String(status, body)` is an HTML response.
- `core.Text(status, body)` is plain text.
- `core.Redirect(status, url)` redirects.
- `core.Error(code, msg)` is a minimal error response.

Rendering happens in a buffer, so a template error yields a complete response,
never a half-written page.

## Raw http.Handler

For endpoints that do not fit the value model (storage downloads, probes),
use `r.Mount` with an `http.Handler`.
