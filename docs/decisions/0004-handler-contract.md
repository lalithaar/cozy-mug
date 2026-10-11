# 0004 — Handlers return a Response (the value model)

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
A handler is `func(*Context) Response` — it **returns a value** instead of
writing to an `http.ResponseWriter`. `core.Handler` still satisfies
`http.Handler` (via one `ServeHTTP` adapter), so plain `net/http` middleware and
handlers interoperate. Raw `http.HandlerFunc` stays as the escape hatch.

## Because
Matches Rails/Flask/Django — every view returns a value. It makes a handler a
near-pure function, callable with **no server and no writer**, which is the seam
the agent CLI (`0002`) and tests rely on (`TestHandlerCalledDirectly`). It lets
the framework own status/layout/rendering by convention
(`Convention over configuration`).

## Alternatives rejected
- Plain `http.HandlerFunc` (side-effect writes) — the stdlib outlier; forces
  manual `Write`/status in every view; returns nothing to introspect or wrap.
- A large response framework (renderer registry, content negotiation) — violates
  `0001`; HTML-only scope (`0002`) means it is unnecessary.

## Apply when
Defining any view/handler. Middleware wraps `Handler`s, so the value model holds
end-to-end — a middleware sees the inner handler's returned `Response`.

## Deviate / ask when
Drop to `http.HandlerFunc` for a case the value model can't express (streaming,
SSE, websockets) — escape hatch, documented at the site. Growing `Response` into
a renderer framework = ADR.

## Revisit if
Streaming/SSE/websocket needs become common enough that the value model is in
the way.

## Open (do not guess)
- Error contract: does a handler return `(Response, error)` or rely on
  panic + recovery? Deferred to the error-handling & logging decision.
