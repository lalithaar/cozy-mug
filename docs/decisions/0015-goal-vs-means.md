# 0015 — The goal is DX; frugality is the means

- Status: firm
- Date: 2026-10-13
- Supersedes: amends `0001`'s *ranking* (frugality is a means, not the top-level
  goal). `0001`'s discipline stands.

## Opinion
The framework has a **goal** and a **means**, and they are not the same thing.

- **Goal — the product.** A stranger goes from an empty directory to a
  **deployed, small, real, secure, server-rendered app** about as easily as with
  Django or Rails. That DX is what the framework is *for*.
- **Means — how we get there.** **Frugality** (`0001`): build with the stdlib,
  avoid needless dependencies, keep the surface small and the output tiny. It is
  *how* we build, not *what* we build.

The line that matters:

- **Essential nutrients (the goal) — always built.** The capabilities a real app
  cannot ship without: data access, schema/migrations, forms + validation,
  identity/auth, settings, sessions, errors + secure defaults. Their absence is a
  **wall**, not a saving. Frugality governs *how* each is implemented (small,
  stdlib-first, no needless deps); it **never decides whether it exists**.
- **Fast food (what frugality cuts) — denied by default.** The unnecessary: a
  gobuffalo-style kitchen sink, an avoidable dependency, a config knob for every
  case, glamour that inflates surface and weight.

Frugality aimed at the **means** is sound. Frugality aimed at the **goal** —
refusing to build an essential capability because it "costs lines" — is a diet
that cuts out nutrients. `0001` is reordered beneath the goal accordingly: its
discipline is kept; its claim to be the *top-level filter* is retired.

## Because
A virtue turns into a vice when it stops serving the objective and becomes the
objective. The framework's own rationale was DX: *"The framework exists because
Go's assemble-it-yourself default forces re-scaffolding on every project; its
value is concentrated, pre-made decisions"* (`0001`, its "Because"). Frugality
was introduced to *protect* that value from weight — a means. Left unranked and
labelled "top-level," it began making **goal** decisions — whether a real app's
core capabilities get built — using the **means** vocabulary of line-count and
dependencies. That is the inversion this ADR fixes.

The tradeoff we name and accept: the framework will carry **more surface** than
a bare router — a migrations runner, a forms layer, an auth story, settings.
That surface *is* the product. What we keep refusing is the *other* kind of
surface: the gobuffalo mistake of adding things nobody needs. The answer to a
bloated framework is not a starved one; it is one that adds only what a real app
truly needs, built lean.

## The test (nutrient vs fast food)
Ask, in order:
1. **Can a stranger ship a small, real, secure app without it?** No → **essential
   nutrient** → build it; frugality shapes the implementation only.
2. **Would Django/Rails ship it as part of the framework?** Yes → strongly a
   nutrient; No → suspect.
3. **Is it convenience, an avoidable dependency, a rare-case knob, or output
   bytes?** Yes → **fast food** → denied by default (`0001`).
4. **Still close?** A **means-rule must not settle a goal question**. Weigh the
   **cost of absence** (a wall a stranger hits) against the **cost of inclusion**
   (surface/weight); if it's near, **ask the human**.

The heuristic to hold onto: *don't buy fast food, but don't starve the diet of
nutrients either.* Frugal is not a synonym for thin — it means "no **unnecessary**
weight." A missing essential is not leanness; it is malnutrition.

## Apply when
Every "should we build this?" call — especially when the answer is trending to
"no, it isn't frugal" for something that would otherwise be a core capability.

## Deviate / ask when
- **Security / correctness primitives** — never hand-rolled to save lines; the
  `0001` carve-out stands, unchanged.
- A capability looks essential but **collides with a firm rule** (scope `0002`,
  etc.) — that is an ADR + sign-off, not a quiet frugality veto.
- Genuinely torn between nutrient and fast food → surface it. A goal question
  must not be decided by a means-rule.

## Revisit if
The opposite failure appears: "essential capability" becomes the excuse to add
**fast food** and the surface drifts toward the kitchen sink — then `0001`'s
deny-by-default returns to the fore. Also if the nutrient list itself is
contested: whether something is goal or means is exactly a human call.

## Consequences
- `0001` is **amended**: frugality is an implementation constraint, subordinate
  to the goal; its "top-level filter" framing is retired, the discipline kept.
- Data access, migrations, forms, auth/identity, settings, and sessions are
  **core capabilities**, not discretionary glamour. Building them is meeting the
  goal; omitting them incurs a wall.
- **Sequencing is separate from scope.** *When* each ships is
  `docs/roadmap.md`'s job; *whether* each is in scope is this ADR's.
  "Deferred" means "not yet built," never "optional."
- The `AGENTS.md` ledger leads with the goal and lists frugality as the means.
