# 0021 — File uploads: separate model reference from bytes (storage-backed)

- Status: soft
- Date: 2026-10-15
- Supersedes: none

## Opinion

File uploads follow the Django/Rails shape, kept minimal:

- A model stores a short **storage key** (string column), not the bytes. The
  bytes live in a storage backend. This keeps rows small and lets large files
  sit on disk/object storage.
- The framework ships one backend, `upload.Local` (filesystem), behind a small
  `Local` API: `Save` / `SaveFile`, `Open`, `Delete`, `URL`, and an
  `http.Handler` for serving. Apps swap it for their own backend by
  implementing the same calls.
- `form.Bind` binds a model field of type `*multipart.FileHeader`
  automatically from the request (via `r.FormFile`), so a form can `required:""`
  an upload and re-render with errors like any other field. Other types are
  still read manually.
- `app.Config.Uploads` accepts an `upload.PublicStore` (interface), which
  `*upload.Local` satisfies; a larger/remote backend can implement the
  `upload.Store` contract and register its own route instead of using auto-mount.
- `app.Config.Uploads *upload.Local`, if set, is mounted at the store's prefix
  (default `/uploads/`) so stored keys are publicly addressable without
  hand-wiring a file server. That mount disables directory listing and sets
  `X-Content-Type-Options: nosniff`, cache-control, and inline/attachment
  disposition automatically.

## Production hardening (added)
- Dangerous active-content extensions/types are rejected by default unless
  `AllowDangerous` is set or an explicit allow list is given.
- `upload.Local.Serve(w, r, key)` streams a single key after the caller has
  performed authorization, for private uploads; `app.Config.Uploads` is the
  public path.
- Keep `form.MaxMultipartBytes` aligned with `upload.Options.MaxBytes`; the
  upload bytes are validated during Save before any disk write beyond the
  multipart parse itself.
- Security defaults live in the backend: random key names (no user-controlled
  path reaches disk), cleaned/contained paths, size ceilings, optional
  content-type and extension allow-lists, and atomic temp-file + rename writes.

## Because

- Storing bytes in the DB (`[]byte` columns) is already possible but poor for
  anything beyond small images; a reference + backend is the established shape
  and matches the framework's "essential nutrient, minimal implementation"
  stance (0015).
- Auto-binding `*multipart.FileHeader` removes the one piece of boilerplate
  (calling `c.R.FormFile`) that made uploads feel unprovided, while keeping
  the coercion list bounded (0017).
- A concrete `*upload.Local` instead of an interface in `app.Config` keeps the
  common case zero-ceremony: `app.Config{ Uploads: upload.New("uploads",
  "/uploads/") }`.

## Deviate / ask when

- A new storage backend or signed-URL flow — implement outside the package;
  the core stays dependency-free (0001).
- Per-model cleanup on delete, variants/processing, or direct-to-storage
  browser uploads — larger features, separate ADR.
- Binding arbitrary non-file types — reopen the coercion list (0017).

## Revisit if

Developers consistently want DB-backed bytes, signed URLs, or automatic model
cleanup; or the local-filesystem default proves insufficient for the "small
real app" target.<|end|><|start|>write to=assistant<|channel|>analysis<|message|>Created file successfully: docs/decisions/0021-file-uploads.md