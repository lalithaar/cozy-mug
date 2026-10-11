# Roadmap — release criteria

Source of truth for what gates each tag (`docs/decisions/0008`). A tag is cut
only when its checklist is green. Area stances: `docs/gap.md`.

Gate for **v0.1**: a stranger can go from an empty dir to a deployed, small,
real app following the framework's opinions, with no undecided or undocumented
wall.

## v0.0.1 — the spine (prerelease)

The nucleus, for the record; **no adoption promise** (module path is still the
`cozymug` placeholder).

- [x] Handler contract — `func(*Context) Response` (`0004`)
- [x] Routing — ServeMux wrapper, named/reverse URLs, groups, middleware, canonical slash (`0005`)
- [x] Templating — `html/template`, `{{block}}` layouts + partials, buffered render (`0006`)
- [x] Errors + logging — tiered errors, one boundary, slog request line + request id (`0007`)

## v0.1.0 — first usable release

- [ ] **Project layout / scaffolding** (`gap` 1, `open`) — conventional dirs, `main.go` wiring, how run vs build works; ADR.
- [ ] **App glue** — one constructor wiring Logger + Errors + Views + Router (no per-app hand-wiring).
- [ ] **Data access** (`gap` 8, `open`) — decide the idiom (`database/sql` + thin helpers + raw-SQL hatch; no ORM) and document it.
- [ ] **Global context / processors** (`gap` 7) — current user, flash, CSRF injected into every view.
- [ ] **Forms** (`gap` 6) — bind -> validate -> re-render-with-errors.
- [ ] **Sessions + CSRF** (`gap` 4) — signed cookie store + tokens, on by default.
- [ ] **Settings** (`gap` 13) — minimal env-based config convention.
- [ ] **Default UI** (`0003`) — minimal system-color CSS + tiny components.
- [ ] **Identity & docs** — replace the `cozymug` placeholder module path; README + getting-started.
- [ ] **Testing convention** (`gap` 14) — documented `httptest` / test-client seam (agent CLI may slip).

### Deferred past v0.1 (reopening = ADR)
- Migrations (`gap` 9), auth/users (`gap` 10, hatch), admin/scaffold-gen (`gap` 15),
  agent CLI (`gap` 16, `0002`), i18n/jobs/mail (`gap` 17), tier-3 `{{render}}`
  boundary (`0007`), asset cache-busting (`gap` 11).
