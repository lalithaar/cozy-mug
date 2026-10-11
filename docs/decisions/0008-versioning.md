# 0008 — Versioning & release criteria

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
Pre-1.0 version numbers track **how much of the "decided 90%" is decided *and
built*** — not time.

- **v0.0.x** = spikes / nucleus. No external promise; each tag may break freely.
- **v0.1.0** = the first version with a documented path from an empty dir to a
  **deployed, small, real app**, with no undecided or undocumented wall.
- **1.0** = the firm set is complete and the API is stable enough to promise no
  breaking changes.

The **single gate for v0.1**: *can a stranger build and ship one small CRUD app
following the framework's opinions without hitting an `open` decision?*

## Because
Semver gives `0.x` no shared meaning, so we supply one that maps to
user-facing readiness instead of the calendar. "One real app, no wall" is
falsifiable and is exactly the 90% the gap map (`docs/gap.md`) claims to serve.
It also keeps the number honest: the spine being green does **not** earn v0.1 if
a user would still hit an `open` item at step three.

## Alternatives rejected
- **Date-based versions** — communicates nothing about readiness.
- **v0.0.1 the moment the spine compiles** — would imply a usable release; the
  spine alone leaves three `open` walls (layout, data, context).
- **Wait for 1.0 to tag anything** — no checkpoints, no release notes, harder to
  bisect "what changed".

## Apply when
Deciding what to tag, what a tag promises, and whether a release is ready.

## Deviate / ask when
External users depend on a pre-1.0 version and need a stability promise earlier
than 1.0 — that reopens this ADR.

## Consequences / rules (load-bearing)
- `docs/roadmap.md` is the **source of truth** for release criteria; a tag is cut
  only when its checklist is green.
- Pre-1.0 breakage is allowed but must be logged in `docs/LOG.md`. No
  deprecation dance before 1.0.
- Security fixes are never gated on a feature checklist; they cut a patch tag
  immediately.
- **v0.0.1** = the spine (router, middleware, views, errors/logging) tagged as a
  prerelease for the record; it carries **no adoption promise**.

## Revisit if
The pre-1.0 stability expectations change, or the v0.1 gate is met and 1.0
planning begins.
