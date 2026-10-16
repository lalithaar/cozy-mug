---
title: "app.Config"
description: "Assemble and run a cozy-mug app."
---

`app.New(cfg)` and `app.Must(cfg)` assemble the app.

## Key fields

| Field | Purpose |
| --- | --- |
| `Assets` | Embedded `views/` and `static/` (required). |
| `Routes` | Registers routes (`func(*router.Router)`). |
| `Dev` | Verbose error pages and dev reload injection. |
| `Logger` | Request logger (`*slog.Logger`, nil uses default). |
| `UI` | Opt into the `ui` component library. |
| `Funcs` | Extra template functions. |
| `Session` | Session/CSRF store (`nil` disables). |
| `Processors` | Request-scoped global values. |
| `Uploads` | Optional upload store. |

## Run

```go
app.ListenAndServe(":8080")   // timeouts + graceful shutdown
app.Server(":8080")          // tune the *http.Server
```
