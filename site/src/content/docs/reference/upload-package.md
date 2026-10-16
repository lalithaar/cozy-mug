---
title: "upload package"
description: "Local file storage and serving."
---

- `upload.New(root, prefix)` creates a `Local` store.
- `Save(ctx, r, field, options)` persists uploaded files with size/type checks.
- `Open`, `Delete`, `URL`, `Handler`, and `Serve` cover addressability and
  authorized downloads.
- `app.Config.Uploads` mounts the store automatically.
