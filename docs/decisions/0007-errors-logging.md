# 0007 — Error handling & logging

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
Errors work in three tiers.

1. **Fatal / bootstrap** — `panic` / `core.Must` only at **startup**, or when the
   server genuinely cannot continue. Never for a request-time failure that could
   render a page.
2. **Route / loader errors** — loaders are plain Go returning `(T, error)`; a
   structured `core.HTTPError` (or the `NotFound`/`BadRequest`/… sugar) carries
   the status; the route maps it with `c.Error(err)`.
3. **Render boundaries** — a `{{render}}` partial boundary so a broken component
   degrades instead of blanking the page. **Deferred** (see Revisit).

A single `core.Errors` middleware is the request boundary: it recovers panics,
turns a `core.Failure` response into an error page, and logs it. `core.Logger`
emits one slog line per request with a request id. Dev shows the detail in the
browser; prod logs it and shows a generic or app-supplied page.

## Because
Handlers return a value (`0004`), so the cleanest status-aware path is a **value**
(`c.Error`) an outer middleware consumes — no signature change, no panic for
expected errors. Loaders returning `(T, error)` stay ordinary, testable Go.
Buffering (`0006`) plus a recover boundary means a template error or panic still
yields a complete page. Dev/prod divergence is the single thing that makes errors
frustrating: a stack you can read in dev, a generic page + correlated log line in
prod, both from one pipeline.

## Alternatives rejected
- **Panic for every error** (tier-2 as throw) — puts stack capture and a second
  control-flow mechanism on the common path; panic stays a boot word.
- **`func(*Context) (Response, error)`** — changes the firm handler contract
  (`0004`) and the plain-function seam.
- **Crash the process on mid-request infra failure** (e.g. DB drop) — takes down
  every other in-flight request; runtime infra failure is a **503**, not death.

## Apply when
Loaders return errors; handlers map them with `c.Error`; the app is wrapped with
`Logger` + `Errors`.

## Deviate / ask when
A genuine cross-request **fatal shutdown** path is needed (a "stop the server"
signal) — raise an ADR; that is not tier 2.

## Consequences / rules (load-bearing)
- **Tier 1 = boot only.** `core.Must` panics at startup; a mid-request infra
  failure is tier 2 (e.g. 503), never process death.
- **Tier 2**: loaders return `core.HTTPError`; handlers `return c.Error(err)`.
  `errors.As` unwraps the status; a plain error → 500 `Failure`.
- `Failure` **is a Response**, so handlers stay `func(*Context) Response`
  (`0004` unchanged). With no `Errors` middleware it still writes a plain page —
  a handler can never leave the client with nothing.
- `core.Errors` is the single boundary: recovers panics **and** converts
  `Failure`. Place it **inside** `Logger` (Logger sees the rendered status via
  `StatusCoder`).
- **Dev vs prod**: dev renders status + message + stack + request id inline; prod
  renders `views/errors/<status>.html` if present, else a plain page, and
  **never leaks the cause**. `Errors` logs 5xx at `Error`, 4xx at `Warn`.
- Request id is set by `Logger` (also sent as `X-Request-Id`) and read via
  `c.RequestID()`.
- **Tier 3 (render boundary) is deferred**, not implemented.

## Revisit if
Tier-3 boundaries are needed, or a real cross-request fatal-shutdown path emerges.
