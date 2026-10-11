# 0002 — Scope: what this is, and what it explicitly is not

- Status: firm (exclusions + agent CLI)
- Date: 2026-10-11
- Supersedes: none

## Opinion
This framework does **one** job: build **server-rendered HTML applications in
Go**. Stated non-goals — we will not grow into these, even on request:

- **NOT a backend / API for modern JS frontend frameworks.** Not a headless JSON
  API serving a React/Vue/Next/Svelte SPA. HTML is the output; the server owns
  rendering. If your app renders itself in the browser, this is the wrong tool.
- **NOT "the REST API framework for Go."** Stateless JSON/CRUD service
  frameworks are a different tool's job.

Explicitly **included**: a small **agent CLI** (dev-time) — see below.

## Because
A framework that pretends to fit everyone serves no one. Naming the non-goals
(a) keeps the design frugal by refusing the features that pull toward
generality, (b) sets honest expectations so people self-select instead of
feeling misled or left out, and (c) gives future-me a rule to cite when
declining scope creep. The exclusion is deliberate and stated up front — it is
a boundary, not a gap.

## Alternatives rejected
- A general web framework that also does JSON APIs — that is how you become
  `net/http` + a router + a config file + everyone's ORM.
- Staying silent on scope and letting users discover the limits later — erodes
  trust and invites ill-fitting adoption.

## Apply when
Any feature request, PR, or design that pushes toward being an API backend or a
general-purpose/REST framework. Also when writing README / positioning.

## Deviate / ask when
If someone needs a small JSON endpoint *inside* an HTML app (e.g. an AJAX hook),
that is an escape hatch to raw `net/http` — not a pivot to API-framework status.
Any real expansion of scope = ADR + human sign-off.

## Revisit if
The HTML-app use case proves too narrow to sustain the project and the
maintainer chooses to redefine its purpose.

## Included — agent CLI (resolved 2026-10-11)
- Purpose: let agents drive and test **end-to-end flows of an HTML app without a
  headless browser**. Introspect registered routes/handlers and invoke them
  directly, asserting on returned HTML + status.
- Shape: **dev-time only, stdlib-only, lives in the repo.** Does NOT ship in
  users' production binaries and does NOT become a module dependency.
- Timing: **after the walking skeleton** — its shape depends on the core API.
- Why frugal: it is tooling for the framework's own agent-driven workflow, not
  runtime surface shipped to users; and it removes a far heavier dependency (a
  headless browser) from the dev loop.
