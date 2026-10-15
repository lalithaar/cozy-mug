package upload

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newFileRequest(t *testing.T, field, name, ctype, body string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if ctype != "" {
		// CreateFormFile sets its own content type; override not needed for tests.
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

func TestResolveRejectsTraversal(t *testing.T) {
	s := New(t.TempDir(), "/u/")
	if _, err := s.resolve("../"); err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
	if _, err := s.resolve(""); err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey for empty key, got %v", err)
	}
}

func TestURL(t *testing.T) {
	s := New("uploads", "/uploads/")
	if got := s.URL("a/b.png"); got != "/uploads/a/b.png" {
		t.Fatalf("got %s", got)
	}
}

func TestSaveOpenDelete(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, "/u/")
	r := newFileRequest(t, "file", "hello.txt", "text/plain", "hi there")
	key, err := s.Save(context.Background(), r, "file")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if filepath.Ext(key) != ".txt" {
		t.Fatalf("ext lost: %s", key)
	}
	rc, err := s.Open(key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	buf := make([]byte, 100)
	n, _ := rc.Read(buf)
	rc.Close()
	if string(buf[:n]) != "hi there" {
		t.Fatalf("body: %q", buf[:n])
	}
	if err := s.Delete(context.Background(), key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(key))); !os.IsNotExist(err) {
		t.Fatalf("file still present")
	}
}

func TestSizeLimit(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, "/u/")
	r := newFileRequest(t, "file", "big.txt", "text/plain", strings.Repeat("a", 100))
	_, err := s.Save(context.Background(), r, "file", Options{MaxBytes: 10})
	if err != ErrTooLarge {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func TestAllowedExtension(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, "/u/")
	r := newFileRequest(t, "file", "x.png", "image/png", "bytes")
	_, err := s.Save(context.Background(), r, "file", Options{AllowedExtensions: []string{".jpg"}})
	if err != ErrInvalidExtension {
		t.Fatalf("expected ErrInvalidExtension, got %v", err)
	}
}

func TestMissingFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, "/u/")
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.Close()
	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	if _, err := s.Save(context.Background(), r, "file"); err != ErrMissingFile {
		t.Fatalf("expected ErrMissingFile, got %v", err)
	}
}
