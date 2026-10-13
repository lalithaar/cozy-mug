# 0020 — optional flash message package

- Status: firm (implemented)
- Date: 2026-10-13
- Supersedes: none

## Opinion
`flash` is an opt-in package that queues one-shot messages in the session and
consumes them on the first render. The framework's existing global
context/processor system (`0014`) supplies the rendering path, so no new core
middleware ordering is needed.

## Because
The app glue already wires sessions and processors. Storing the queue in the
session and popping it in a processor reuses that seam instead of adding a
separate messages subsystem. It also works across redirects and on CSRF-rejected
unsafe requests because processors run before the CSRF barrier's error render.

## Alternatives rejected
- Core messages API — rejected; messages are ergonomics, not an essential
  nutrient, and can remain an opt-in package.
- Cookie-only flash — rejected because it reuses session machinery and keeps
  server-side revocation semantics.

## Apply when
An app wants Django-style flash messages without a messages subsystem.

## Deviate / ask when
A change would make messages a core middleware or add DB-backed inbox behavior.