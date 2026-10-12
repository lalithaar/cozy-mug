# Heuristics

Operational rules. Every entry uses this shape:

```
### <name>
- Opinion:     what we do
- Because:     the real reason (name the tradeoff we accept)
- Apply when:  trigger condition
- Deviate/ask: when to override, or stop and ask the human
- Status:      firm | soft | open
```

---

### Prefer reversible first
- Opinion: When no rule covers a case, ship the smallest reversible version and log the gap.
- Because: Reversible guesses cost little to undo; blocking on every unknown kills velocity.
- Apply when: You hit an uncovered decision that is easy to change later.
- Deviate/ask: If the change is hard to reverse or cross-cutting, stop and ask.
- Status: firm

### Record the reason, not the conclusion
- Opinion: Every decision/idiom carries its rationale and the alternative it beat.
- Because: Future sessions relitigate conclusions, but accept stated reasoning.
- Apply when: Writing any rule, decision, or idiom.
- Deviate/ask: Never deviate.
- Status: firm

### Frugal by default (burden of proof inverts)
- Opinion: New dependency, exported API, config knob, feature, or client byte is denied by default. It must justify why stdlib cannot do it.
- Because: See `docs/decisions/0001` — weight is permanent tax; small = fast, auditable, in-context.
- Apply when: Any proposed addition. Also the tie-breaker between equal designs.
- Deviate/ask: Security primitives -> never hand-roll; DB driver -> standing exemption; anything cross-cutting -> ADR + sign-off.
- Status: firm

### Escape hatch over feature
- Opinion: When a case isn't served, expose raw `net/http` rather than add config/options.
- Because: Keeps the core small; serves the 90% well and refuses the 10% instead of growing.
- Apply when: Tempted to add flexibility/config for an edge case.
- Deviate/ask: If the 10% is actually the 90%, that's an ADR, not a knob.
- Status: soft

### Frugality is measured, not vibed
- Opinion: Frugality targets are numeric and empirical, set from the skeleton's measured baseline.
- Because: An unmeasured constraint decays into an aspiration (a vision statement).
- Apply when: After the walking skeleton exists; enforce in CI.
- Deviate/ask: Deferred until baseline measured (see LOG 2026-10-11).
- Status: soft

### Name the non-goals
- Opinion: Every scoping decision states what we refuse, not just what we build.
- Because: Frugality is mostly refusal; explicit non-goals prevent drift, set honest expectations so people self-select, and give future-me a rule to cite.
- Apply when: Positioning, README, feature requests, design reviews.
- Deviate/ask: Never — hitting a non-goal is an ADR, not a negotiation.
- Status: firm

### Optimize for the agent dev loop
- Opinion: Expose testable seams so an agent can exercise end-to-end flows programmatically (no headless browser).
- Because: The framework's own workflow is agent-driven; a drivable app is cheaper to verify than one that needs a browser. See `docs/decisions/0002`.
- Apply when: Designing handlers/routes and the dev CLI.
- Deviate/ask: Don't add runtime surface for it — seams must be free at runtime.
- Status: soft

### Good-looking by default, never forced
- Opinion: Scaffold ships a minimal working UI (system-color CSS + tiny components) and it is fully opt-out.
- Because: Removes the "how do I make it look decent?" friction; convention over configuration. See `docs/decisions/0003`.
- Apply when: Scaffolding; authoring CSS/components.
- Deviate/ask: Framework core never depends on the UI layer; any CSS build step/preprocessor = ADR.
- Status: firm

### Just enough styling to be functional
- Opinion: The default UI provides the minimum to ship a usable, accessible page without touching CSS — and it is **not a dependency unless you opt in**. Import it or don't; it's never required.
- Because: Styling must never be a blocker to shipping; defaults should carry you, and removing them should cost nothing. Spend exactly enough to clear the bar, no more.
- Apply when: Building the default UI / component library (`ui`).
- Deviate/ask: Any styling the core depends on, or that becomes mandatory, is a bug — flag it.
- Status: firm

### Convention over configuration
- Opinion: Prefer a good default the user can reject over a knob they must set.
- Because: Decisions cost the user time; a default costs zero until rejected. Opt-out beats opt-in.
- Apply when: Any place you'd add a config option or required choice.
- Deviate/ask: If the *default* is wrong for most users, fix the default, don't add the knob.
- Status: firm

### Redirects are security surfaces
- Opinion: Never build a `Location` from raw request input without `path.Clean`/rooting; reject protocol-relative targets.
- Because: `//host` in `Location` is an open redirect. Our own canonical-slash did exactly this before the fix.
- Apply when: Any redirect whose target derives from the request.
- Deviate/ask: Never.
- Status: firm

### Fail loud in dev, quiet in prod
- Opinion: Error detail goes to the developer in dev and to the logs in prod; prod responses never leak causes.
- Because: A stack you can read in the browser is the fastest 11pm fix; a stack in a prod response is an information leak.
- Apply when: Any error page or diagnostic surface.
- Deviate/ask: Never leak in prod.
- Status: firm

### Public surfaces introduce, they don't log
- Opinion: README, release notes, and any user-facing copy describe what a user gets and who it's for — never ADR numbers, `docs/`/`LOG` pointers, or process/commit language.
- Because: Those artifacts are written for the future maintainer, not the visitor; leaking them reads as changelog noise and buries the pitch. This regressed twice (release notes shipped ADR refs and a `docs/LOG.md` pointer).
- Apply when: Writing any public artifact — README, release announcement, docs site, user-visible error text.
- Deviate/ask: Never in user-facing copy; process detail stays in `AGENTS.md`/`docs/`.
- Status: firm
