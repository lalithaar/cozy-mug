---
title: "File uploads"
description: "Add file uploads to a cozy-mug app, including storage, serving, and production checks."
---

This page shows you how to add file uploads to a cozy-mug app. It also
explains what cozy-mug handles for you and what you still own.

## Add an upload

### 1. Add a model field

Store the storage key as a string. Keep the upload itself out of the database
as a `*multipart.FileHeader` field.

```go
type Post struct {
    data.Model  `db:"posts"`
    Title       string                `db:"title" required:"" min:"3"`
    Photo       string                `db:"photo"`
    PhotoFile   *multipart.FileHeader `db:"-" form:"photo" required:""`
}
```

### 2. Create a storage backend

```go
uploads := upload.New("uploads", "/uploads/")
```

### 3. Save the uploaded file in your handler

```go
func (s *Store) create(c *core.Context) core.Response {
    f := form.Bind[Post](c)
    if !f.Valid() {
        return c.Render(http.StatusUnprocessableEntity, "posts/new", &postView{Form: f})
    }

    key, err := s.uploads.Save(c.R.Context(), c.R, "photo",
        upload.Options{MaxBytes: 5 << 20})
    if err != nil {
        f.AddError("photo", err.Error())
        return c.Render(http.StatusUnprocessableEntity, "posts/new", &postView{Form: f})
    }
    f.Value.Photo = key

    if err := s.db.Table[Post]().Create(c.R.Context(), &f.Value); err != nil {
        _ = s.uploads.Delete(c.R.Context(), key)
        f.AddError("", "could not save the post")
        return c.Render(http.StatusUnprocessableEntity, "posts/new", &postView{Form: f})
    }

    return core.Redirect(http.StatusSeeOther, "/posts")
}
```

### 4. Wire the store into your app

```go
a := app.Must(app.Config{
    Assets:  assets,
    Routes:  store.Register,
    Uploads: uploads, // public at /uploads/
})
```

## What cozy-mug handles

- **Multipart parsing.** `form.Bind` calls `r.ParseMultipartForm` with
  `form.MaxMultipartBytes` (default 10 MiB).
- **Field binding.** A model field of type `*multipart.FileHeader` is populated
  from the request automatically. The original filename is available through
  `Form.Raw`.
- **Required validation.** You can use `required:""` on a file field.
- **Safe storage.** `upload.Local.Save` generates a random key, validates size,
  rejects dangerous active content by default, cleans paths so they cannot
  escape the storage root, and writes through a temporary file and rename.
- **Serving.** When you set `app.Config.Uploads`, cozy-mug mounts the store's
  handler at its prefix with `nosniff`, cache headers, inline or attachment
  disposition, and no directory listing.
- **Private serving primitive.** `upload.Local.Serve(w, r, key)` streams one
  validated key after you perform authorization.

## What you still own

| Concern | Handled by cozy-mug | Your responsibility |
| --- | --- | --- |
| Multipart parsing | Yes | Keep `form.MaxMultipartBytes` in line with upload limits |
| Size, type, and extension checks | Defaults and options | Choose `AllowedExtensions` and `AllowedContentTypes` |
| Storage key generation | Random and path-contained | — |
| Authorization and ownership | `Serve` primitive | Check before serving private files |
| Private files | Do not use `app.Config.Uploads` | Add a route and call `store.Serve` |
| Orphan cleanup after a failed save | — | Delete the key if insert fails |
| Quotas, rate limiting, and scanning | `Options.Validator` hook | Implement |
| Replaced or deleted files | — | Delete the old key on update |

## Serve private files

Public mounts are convenient but expose every key without authorization. For
private uploads, do not set `app.Config.Uploads`. Instead, add a route and
call `store.Serve` after your ownership check.

```go
func (s *Store) download(c *core.Context) core.Response {
    if !s.canAccess(c, c.Param("key")) {
        return c.Error(core.NotFound(nil))
    }
    if err := s.uploads.Serve(c.W, c.R, c.Param("key")); err != nil {
        return c.Error(core.NotFound(err))
    }
    return nil
}
```

## Best practices

1. Store a key, not bytes. Keep database rows small. Use `[]byte` columns only
   for very small payloads.
2. Reject by extension, not only by content type. Browsers trust the
   `Content-Type` header; the extension deny list is the safer gate.
3. Keep limits consistent. Set `form.MaxMultipartBytes` to at least
   `upload.Options.MaxBytes`.
4. Clean up on failure. Treat `Save` and `Create` as one unit: if the row
   fails, delete the key.
5. Serve private files through authorization. Do not publish
   `app.Config.Uploads` for sensitive files.
6. Replace uploads atomically. Save the new key, update the row, then delete
   the old key. If the update fails, delete the new key.
7. Add a validator when you need more checks. Use `Options.Validator` for
   magic-byte or scanning logic without changing the core.

## Common pitfalls

- Uploading into a `[]byte` column works, but you lose the backend's size and
  path checks. Prefer the key and `upload.Local` pattern.
- You do not need to serve user-provided filenames. The key is random; the
  original name is metadata.
- For very large files, consider streaming to the backend before `form.Bind`
  parses the whole request.
- Returning `nil` from a public handler after `store.Serve` is acceptable because
  `Serve` writes the response directly.
```

But wait, handler contract returns Response; returning nil? Core.Handler expects Response, but can return nil? Errors middleware maybe panic? Let's check 0004 and context. In examples, handlers always return core.Response. Returning nil would cause nil dereference in ServeHTTP adapter likely. Need inspect router/core adapter.<|end|><|start|>Assistant to=shell<|channel|>analysis <|constrain|>xml<|message|><rfl:parameter name="command">
grep -n "ServeHTTP\|resp.WriteTo\|\.WriteTo" core/*.go app/*.go router/*.go
