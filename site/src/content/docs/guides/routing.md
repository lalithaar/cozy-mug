---
title: "Routing"
description: "Named routes, groups, middleware, and reverse URLs."
---

cozy-mug wraps `net/http`'s `ServeMux`.

## Basic routes

```go
r.Get("/", s.home).As("home")
r.Post("/notes", s.createNote).As("notes.create")
r.Get("/notes/{id}", s.showNote).As("note")
```

## Reverse URLs in handlers and templates

Handlers access `c.URL("name", map[string]string{...})`. Templates get
`{{.URL "name" "id" .ID}}` via `core.Globals`.

## Groups and middleware

```go
notes := r.Group("/notes")
notes.Use(withExampleHeader)
notes.Get("/{id}", s.showNote)
```

Middleware order runs as registered; the first added is outermost.

## Static files

```go
r.Static("/static/", "static")
r.StaticFS("/static/", staticFS)
r.Mount("/uploads/", store.Handler())
```

## 404 handling

```go
r.NotFound(notFound)
```
