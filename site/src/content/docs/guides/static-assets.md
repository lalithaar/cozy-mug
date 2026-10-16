---
title: "Static assets"
description: "Serve files and embedded assets."
---

Place files in `static/` and cozy-mug mounts them at `/static/`.

## Uploads

Use `upload.New("uploads", "/uploads/")` and call `r.Mount(store.Prefix, store.Handler())`.
Configuring `app.Config.Uploads` mounts it for you.

## User assets

Serve a single directory with `r.Static` or an `fs.FS` with
`r.StaticFS`. For storage backends, use `r.Mount`.
