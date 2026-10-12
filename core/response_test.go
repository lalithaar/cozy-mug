package core_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lalithaar/cozy-mug/core"
)

func TestErrorEscapesMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	core.Error(http.StatusBadRequest, "<script>alert(1)</script>").
		WriteTo(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if strings.Contains(rec.Body.String(), "<script>") {
		t.Fatalf("message not escaped: %q", rec.Body.String())
	}
}

func TestHTMLWithSetsHeaderAndContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	core.String(http.StatusOK, "x").With("X-Test", "1").
		WriteTo(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Header().Get("X-Test") != "1" {
		t.Fatal("custom header missing")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestStatusEmptyBody(t *testing.T) {
	rec := httptest.NewRecorder()
	core.Status(http.StatusNoContent).WriteTo(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}
