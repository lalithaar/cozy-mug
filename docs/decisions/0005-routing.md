# 0005 — Routing wraps ServeMux

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
Routing is a **thin wrapper over `net/http`'s `ServeMux`** (Go 1.22+ patterns).
No custom matcher, no reflection. On top it adds: named routes with reverse URL
generation (`Route.As` / `Router.Path`), route groups (`Router.Group`),
middleware (`core.Middleware` + `Router.Use`), static mounts, and a default
canonical-slash redirect. Path params come from `r.PathValue`.

## Because
`ServeMux` already does method matching, `{id}` params, `{p...}` wildcards, and
precedence — reusing it is the frugal choice (`0001`) and inherits well-tested
matching. Named/reverse URLs, groups, and middleware are the ergonomics that make
it Rails/Django-like (`gap.md` rows 2-3). The whole wrapper is ~40 lines.

## Alternatives rejected
- A custom trie/router (chi/gorilla class) — reimplements what stdlib now has,
  for a dependency or a lot of code.
- No named routes — loses reverse URL generation, which templates need.

## Apply when
Registering routes; generating links (`Router.Path`); grouping with a shared
prefix + middleware.

## Deviate / ask when
If `ServeMux` patterns can't express a needed matching rule, raise an ADR before
adding a matcher.

## Consequences / rules (load-bearing)
- Middleware is applied by **wrapping each handler at registration**; the first
  middleware added is the outermost. `Use` affects only routes registered after it.
- Params come from `r.PathValue`, so a **directly-called handler sees none**.
  In-process verification (agent CLI / tests) must dispatch through the Router
  (`Router.ServeHTTP` against a recorder) — no socket, no browser.
- **Trailing slash is canonicalized**: bare form wins; `/foo/` 301-redirects to
  `/foo` (308 for non-GET/HEAD, preserving method + body). Registered
  subtree/mount patterns (ending in `/`) are exempt as directory-like. Opt out
  with `Router.DisableCanonicalSlash`.
- `Router.Routes()` exposes method/pattern/name for introspection.
- **Reverse URLs are reachable from handlers** via `c.URL(name, params)` —
  `Context` carries a `Reverser` interface that the router injects as the
  outermost middleware (templating reuses the same seam).
- Canonical redirects run the target through `path.Clean`, so `Location` is
  always a rooted path — never a protocol-relative (`//host`) **open redirect**.
- **Duplicate route names panic** at registration (fail-fast, like ServeMux).

## Revisit if
`ServeMux` semantics prove too limited, or group/middleware ordering must become
dynamic rather than registration-time.
