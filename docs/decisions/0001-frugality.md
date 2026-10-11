# 0001 — Frugality is the top-level filter

- Status: firm
- Date: 2026-10-11
- Supersedes: none

## Opinion
Every addition — a dependency, a line of code, an exported symbol, a config
knob, a feature, a byte of output (including client JS) — is **denied by
default**. It is allowed only when the stdlib genuinely cannot do the job, and
then it needs an ADR + human sign-off. When two designs are otherwise equal,
pick the smaller.

## Because
The framework exists because Go's "assemble it yourself" default forces
re-scaffolding on every project; its value is concentrated, pre-made decisions.
Weight is the enemy of that value: deps and public surface are permanent
maintenance + cognitive tax, and every byte ships to every user. Small also
means fast to build, fast to start, easy to audit for security, and easy for a
future agent to hold in context. The maintainer is frugal by disposition; the
design should reflect that.

## Alternatives rejected
- Pragmatic deps (import a router/logger/ORM) — trades our weight for someone
  else's maintenance and an unpredictable API.
- Config-driven flexibility (an option for every case) — moves weight out of the
  code and into the user's head.
- Feature completeness — "support everything" is how a framework becomes
  unforkable and unreadable.

## Apply when
Any time you would add a dependency, exported API, config option, feature, or
client bytes. Also as the tie-breaker between otherwise-equal designs.

## Deviate / ask when
- **Security / correctness**: do NOT hand-roll crypto, session, or CSRF
  primitives to save lines. Stdlib's vetted primitives cost 0 deps; if the
  stdlib cannot do it safely, that is the absolute-need exception — ADR + sign-off.
- **DB driver**: the one standing exemption (`database/sql` drivers are
  unavoidable). Pick the smallest; still worth a LOG note.
- Hand-rolled code carries full testing burden. Frugality never means less testing.

## Revisit if
Budgets prove unenforceable, or a real use case appears that the frugal stance
cannot serve without an escape-hatch-free change (i.e. the 90/10 refusal starts
hitting the 90).
