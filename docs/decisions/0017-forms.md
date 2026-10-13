# 0017 — Forms: bind -> validate -> re-render-with-errors

- Status: firm
- Date: 2026-10-13
- Supersedes: none

## Opinion

Forms ship as a separate `form` package that binds an HTTP request onto a
model value, validates it, and carries the submitted raw strings back so the
same template re-renders with errors and repopulated inputs. The model
struct (0016) stays the single representation; there is no parallel form
struct.

```go
f := form.Bind[Note](c)          // parse + coerce, collect raw strings
if !f.Valid() {                  // tag rules -> model hook -> handler adds
    return c.Render(422, "home", &homeView{Notes: notes, Form: f})
}
s.db.Table[Note]().Create(c.R.Context(), &f.Value)
```

```html
<input name="title" value="{{with .Form}}{{index .Raw "title"}}{{end}}">
{{with .Form}}{{range .Errors "title"}}<p class="error">{{.}}</p>{{end}}{{end}}
```

- A form is always present in a view (via `form.New(value)` on GET/edit), so
  templates read `.Form.Raw "field"` unconditionally and never branch on
  whether a submission happened.
- Binding targets a model `T`. Non-persisted inputs (password, confirm) ride
  on `db:"-"` fields; they still bind and validate but are not columns.
- Input names default to the db column name when present, otherwise the
  lowercased Go field name; `form:"name"` / `form:"-"` override.
- Validation runs lazily on the first `Valid()`/`Validate()` call: bounded tag
  rules (`required:""`, `min`, `max`, `email:""`, `url:""`, `oneof`) then a
  model hook, then handler additions via `AddError`. `AddError` after that
  re-invalidates.
- The model hook is cycle-free: `Validate() []core.FieldError` (and
  `Validate() error`), defined against `core` only, so shared model packages
  never import `form`. A `Validate(*form.Form[T])` hook is also supported for
  app-package models that want field-level additions.
- Re-display uses raw submitted strings (`f.Raw`), not the (possibly failed)
  typed value. `BindOver`/`New` seed `Raw` from a loaded base so edit pages
  use the identical template.
- Invalid submissions re-render `422 Unprocessable Entity`. Success stays PRG:
  flash via session, then `core.Redirect(303, ...)`.
- Files are not bound; use `c.R.FormFile`. DB/constraint errors are mapped
  manually with `AddError`, never auto-guessed.

## Because

- 0016 promised forms would read the same declaration, including validation
  rules captured from struct tags. Maintaining a second representation or a
  separate rule set was the drift 0016 exists to delete.
- Re-render-with-errors is the wall the raw example had (manual parse, then
  `c.Error(400)` losing input). A request-bound Form object closes it with
  one value in the view model.
- Manual HTML plus a server-side Form object matches the framework's stance:
  stdlib `html/template`, no auto-generated markup, no client framework.
  Templates stay explicit; the Form makes the tedious parts (values, errors)
  cheap.

## Alternatives rejected

- **Django-style auto-rendered field widgets** — requires the UI/component
  layer to be settled (0010 is a scaffold) and hides markup the developer
  needs to control. Additive later via `ui`, never the core.
- **Vue-style two-way binding / client-side** — out of scope (0002
  server-rendered only, no SPA glue).
- **Arbitrary struct binding** — reintroduces a second representation beside
  the model; non-persisted inputs use `db:"-"` instead.
- **Cycle-bearing hook `Validate(*form.Form[T])` as the primary** —
  forces every shared model package to import `form`, which imports `data`,
  creating an import cycle. Cycle-free `[]core.FieldError` is the default;
  the Form-receiving hook remains an opt-in for app packages.
- **Auto-mapping DB errors (e.g. unique violations) to field errors** —
  0016 classifies no constraint errors; silent mapping would paper over real
  bugs. Documented idiom (parse `errors.As` / `core.Conflict`) instead.
- **String-tag parsing via bare keys (`required min:"3"`)** — Go's
  `reflect.StructTag` only parses `key:"value"` pairs; 0015-style bare keys
  silently fail. Empty-valued keys (`required:"" min:"3"`) are the convention
  and are now implicit throughout the data/form layers.

## Apply when

Any handler that accepts a POST/PUT/PATCH of user input and re-renders the
same view on failure, or a GET that seeds an edit form from a loaded model.

## Deviate / ask when

- File uploads needing binding beyond `FormFile` — reopen the coercion list.
- A new scalar/binding shape not in the coercion set (ptr depth beyond
  one, slices of non-string) — new ADR, not a tag.
- Locale-specific messages — i18n is deferred; reopen before adding.
- Forms that need per-request (contextual) validation beyond `core` — surface
  to the human.

## Revisit if

Developers consistently want auto-mapping of constraint errors or component-
level field rendering; or evidence shows `[]core.FieldError` hooks are too
ceremonious and a string-returning single-error variant would be strictly
better. Also revisit the empty-valued tag convention if a nicer declarative
syntax proves parseable without reflection surprises.
