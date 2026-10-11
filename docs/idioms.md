# Idioms

Concrete "how we do it here" techniques (handlers, templates, errors, data,
layout).

Rule: every idiom must cite the decision that justifies it (`docs/decisions/NNN`).
If you can't cite a decision, write the decision first.

## Render a view (`0006`)
Pass a **pointer** to the view struct — `core.Globals` injection needs it — and
embed `core.Globals` only when the view generates links.

```go
type ShowView struct {
	core.Globals        // gives the template .URL
	User User
}

func show(c *core.Context) core.Response {
	return c.Render(http.StatusOK, "users/show", &ShowView{User: load(c.Param("id"))})
}
```

```html
{{define "title"}}{{.User.Name}}{{end}}
{{define "content"}}<h1>{{.User.Name}}</h1>{{end}}
```

- Layout lives in `views/layouts/base.html` as
  `{{define "base"}}…{{block "content" .}}…{{end}}…{{end}}`; partials in
  `views/partials/`. A view file defines the blocks it fills. (`0006`)
- Views are named by path minus extension (`views/users/show.html` → `"users/show"`). (`0006`)
- Wire once at startup: `set, _ := views.New(fsys, nil)` then
  `r.Use(views.Middleware(set))`. (`0006`)
- `.URL` is `{{.URL "user" "id" .User.ID}}` (name/value pairs); it errors rather
  than emitting an empty link. (`0006`)
