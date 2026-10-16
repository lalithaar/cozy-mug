---
title: "Forms"
description: "Bind, validate, and re-render with errors."
---

Forms are bound from models.

## Bind

```go
f := form.Bind[Note](c)
if !f.Valid() {
    return c.Render(http.StatusUnprocessableEntity, "home", &homeView{Form: f})
}
n := f.Value
```

## Validation tags

Use struct tags such as `required:""`, `min:"3"`, `email:""`, and `max:""`.
`form.Bind` validates automatically.

## Repopulation

Templates read submitted values while invalid:

```html
<input name="title" value="{{with .Form}}{{.Raw "title"}}{{end}}">
```

## File fields

`*multipart.FileHeader` fields are auto-bound when binding uses `BindMultipart`
or `Bind` in a multipart request. Save them through `upload.Local`.
