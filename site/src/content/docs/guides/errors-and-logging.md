---
title: "Errors and logging"
description: "Tiered errors, one boundary, request logging."
---

cozy-mug recovers panics and renders an error page for you. In development it
shows status, message, stack, and request id. In production it renders
`views/errors/<status>.html` and hides causes.

## Returning errors

Loaders return errors. Handlers convert them:

```go
n, err := s.noteByID(c.R.Context(), id)
if err != nil {
    return c.Error(err)
}
```

`core.NotFound`, `core.Conflict`, and `core.HTTPError` carry status codes.

## Logging

`core.Logger` wraps the writer with a status recorder and logs one line per
request: method, path, status, duration, and request id. It also sends
`X-Request-Id`.

## Dev pages

Dev pages are styled with system colors and show the request summary, exception,
and stack. They exist only when `Config.Dev` is true.
