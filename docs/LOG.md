# Log

Append-only. Newest at bottom. One line each: `date — what changed — why`.

- 2026-10-11 — seeded memory system (`AGENTS.md` + `docs/`). Chose thin-rulebook-first, append-freely/promote-on-sign-off, single root hot file.
- 2026-10-11 — adopted frugality as the top-level filter (`docs/decisions/0001`): stdlib-only, DB driver the sole dep exemption, security carve-out. Numeric budgets deferred to post-skeleton measurement.
- 2026-10-11 — fixed scope as firm exclusions (`docs/decisions/0002`): server-rendered HTML only; NOT a JS-SPA backend, NOT a REST framework. Agent CLI noted but left `open` (needs dev-vs-shipped decision).
- 2026-10-11 — resolved agent CLI (`docs/decisions/0002`): dev-time, stdlib-only, lives in repo, never shipped; purpose is browserless e2e driving of an HTML app; lands after the skeleton. Added soft heuristic `Optimize for the agent dev loop`.
- 2026-10-11 — added default-UI opinion (`docs/decisions/0003`): minimal system-color CSS + tiny opt-out components; core must not depend on it; implementation deferred. Added firm heuristics `Good-looking by default, never forced` and `Convention over configuration`. Embed-vs-copy mechanism left `open`.
- 2026-10-11 — wrote `docs/gap.md`: stdlib-vs-Rails/Django gap map with per-area stance (fill/hatch/skip); cited from AGENTS.md read list + open items.
