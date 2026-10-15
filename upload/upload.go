package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const DefaultMaxBytes int64 = 10 << 20

var (
	ErrTooLarge         = errors.New("upload: file too large")
	ErrInvalidType      = errors.New("upload: content type not allowed")
	ErrInvalidExtension = errors.New("upload: file extension not allowed")
	ErrMissingFile      = errors.New("upload: no file provided")
	ErrInvalidKey       = errors.New("upload: invalid storage key")
)

type Options struct {
	MaxBytes            int64
	AllowedContentTypes []string
	AllowedExtensions   []string
	AllowDangerous      bool
	Validator           func(*multipart.FileHeader) error
	Subdir              string
}

var DangerousExtensions = []string{
	".html", ".htm", ".xhtml", ".svg", ".xml", ".js", ".mjs", ".css",
	".php", ".jsp", ".asp", ".aspx",
}

func (o Options) normalized() Options {
	if o.MaxBytes <= 0 {
		o.MaxBytes = DefaultMaxBytes
	}
	out := o
	for i := range out.AllowedContentTypes {
		out.AllowedContentTypes[i] = strings.ToLower(strings.TrimSpace(out.AllowedContentTypes[i]))
	}
	for i := range out.AllowedExtensions {
		ext := strings.ToLower(strings.TrimSpace(out.AllowedExtensions[i]))
		if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		out.AllowedExtensions[i] = ext
	}
	return out
}

type Local struct {
	Dir    string
	Prefix string
}

// Store is the storage contract an app can program against. *Local satisfies it,
// and a remote backend (S3, GCS, Azure, ...) can satisfy it without pulling a
// cloud SDK into cozy-mug itself: implement these methods in your own package.
type Store interface {
	Save(ctx context.Context, r *http.Request, field string, opts ...Options) (string, error)
	SaveFile(ctx context.Context, f *multipart.FileHeader, opts ...Options) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

// PublicStore is a Store that can also serve its own bytes. Local and simple
// filesystem backends satisfy it; remote backends that serve through signed
// URLs or a reverse proxy need not, in which case the app registers its own
// route instead of using app.Config.Uploads.
type PublicStore interface {
	Store
	Handler() http.Handler
	StoragePrefix() string
}

func New(dir, prefix string) *Local {
	if strings.TrimSpace(dir) == "" {
		dir = "uploads"
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = "/uploads/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &Local{Dir: dir, Prefix: prefix}
}

func (s *Local) root() string { return filepath.Clean(s.Dir) }

// StoragePrefix returns the public prefix for a PublicStore.
func (s *Local) StoragePrefix() string { return s.Prefix }

func (s *Local) URL(key string) string {
	key = strings.TrimPrefix(path.Clean("/"+key), "/")
	if key == "." || key == "" {
		return s.Prefix
	}
	return path.Join(s.Prefix, key)
}

func (s *Local) Save(ctx context.Context, r *http.Request, field string, opts ...Options) (string, error) {
	if r == nil {
		return "", ErrMissingFile
	}
	_, fh, err := r.FormFile(field)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", ErrMissingFile
		}
		return "", err
	}
	return s.SaveFile(ctx, fh, opts...)
}

func (s *Local) SaveFile(ctx context.Context, fh *multipart.FileHeader, opts ...Options) (string, error) {
	if fh == nil {
		return "", ErrMissingFile
	}
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o = o.normalized()

	if o.Validator != nil {
		if err := o.Validator(fh); err != nil {
			return "", err
		}
	}

	ct := strings.ToLower(strings.TrimSpace(fh.Header.Get("Content-Type")))
	if len(o.AllowedContentTypes) > 0 {
		ok := false
		for _, allowed := range o.AllowedContentTypes {
			if strings.Contains(ct, allowed) {
				ok = true
				break
			}
		}
		if !ok {
			return "", ErrInvalidType
		}
	} else if !o.AllowDangerous {
		for _, dangerous := range []string{"text/html", "image/svg+xml", "application/xhtml+xml", "text/xml", "application/xml", "text/javascript", "application/javascript"} {
			if strings.Contains(ct, dangerous) {
				return "", ErrInvalidType
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if len(o.AllowedExtensions) > 0 {
		ok := false
		for _, allowed := range o.AllowedExtensions {
			if ext == allowed {
				ok = true
				break
			}
		}
		if !ok {
			return "", ErrInvalidExtension
		}
	} else if !o.AllowDangerous {
		for _, dangerous := range DangerousExtensions {
			if ext == dangerous {
				return "", ErrInvalidExtension
			}
		}
	}
	if ext == "." {
		ext = ""
	}

	if fh.Size >= 0 && fh.Size > o.MaxBytes {
		return "", ErrTooLarge
	}

	subdir := ""
	if strings.TrimSpace(o.Subdir) != "" {
		cleaned := path.Clean("/" + o.Subdir)
		cleaned = strings.TrimPrefix(cleaned, "/")
		if cleaned == "." || cleaned == "" {
			cleaned = ""
		}
		if strings.Contains(cleaned, "..") {
			return "", ErrInvalidKey
		}
		subdir = cleaned
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("upload: generate key: %w", err)
	}
	name := hex.EncodeToString(buf) + ext
	rel := name
	if subdir != "" {
		rel = path.Join(subdir, name)
	}

	full, err := s.resolve(rel)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("upload: mkdir: %w", err)
	}

	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("upload: open source: %w", err)
	}
	defer src.Close()

	tmp := full + ".tmp-" + hex.EncodeToString(buf[:4])
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("upload: create temp: %w", err)
	}

	limited := io.LimitReader(src, o.MaxBytes+1)
	n, err := io.Copy(dst, limited)
	closeErr := dst.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upload: copy: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upload: close: %w", closeErr)
	}
	if n > o.MaxBytes {
		_ = os.Remove(tmp)
		return "", ErrTooLarge
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upload: rename: %w", err)
	}

	return filepath.ToSlash(rel), nil
}

func (s *Local) Open(key string) (io.ReadCloser, error) {
	full, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("upload: open: %w", err)
	}
	return f, nil
}

func (s *Local) Delete(ctx context.Context, key string) error {
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("upload: delete: %w", err)
	}
	return nil
}

// Serve streams a stored key to w, applying the safety headers (nosniff,
// cache-control, disposition). It returns an error when the key cannot be
// resolved or served; the caller turns that into a 404. Use this from a
// handler that performs authorization before serving a private upload.
func (s *Local) Serve(w http.ResponseWriter, r *http.Request, key string) error {
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		return os.ErrNotExist
	}
	ct := mime.TypeByExtension(filepath.Ext(full))
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	disp := "attachment"
	if strings.HasPrefix(ct, "image/") {
		disp = "inline"
	}
	w.Header().Set("Content-Disposition", disp+`; filename="`+filepath.Base(full)+`"`)

	f, err := os.Open(full)
	if err != nil {
		return err
	}
	defer f.Close()
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	return nil
}

// Handler serves stored files under s.Prefix without directory listing and
// with the safety headers applied. Mounted by app.Config.Uploads.
func (s *Local) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, s.Prefix)
		key = strings.TrimPrefix(key, "/")
		if key == "" || key == "." {
			http.NotFound(w, r)
			return
		}
		if err := s.Serve(w, r, key); err != nil {
			http.NotFound(w, r)
		}
	})
}

func (s *Local) resolve(key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", ErrInvalidKey
	}
	cleaned := path.Clean("/" + key)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." || cleaned == "" {
		return "", ErrInvalidKey
	}
	if strings.Contains(cleaned, "..") {
		return "", ErrInvalidKey
	}
	full := filepath.Join(s.root(), filepath.FromSlash(cleaned))
	root := s.root()
	full = filepath.Clean(full)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", ErrInvalidKey
	}
	return full, nil
}
