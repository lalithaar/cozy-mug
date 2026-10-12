package core_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/views"
)

func serve(h core.Handler, mw ...core.Middleware) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	core.Chain(h, mw...).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/thing", nil))
	return rec
}

func TestFailMapsHTTPError(t *testing.T) {
	f := core.Fail(core.NotFound(errors.New("gone")))
	if f.StatusCode() != http.StatusNotFound {
		t.Fatalf("code = %d", f.StatusCode())
	}
}

func TestFailureFallbackWithoutMiddleware(t *testing.T) {
	rec := serve(func(c *core.Context) core.Response {
		return c.Error(core.NotFound(errors.New("nope")))
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "nope") {
		t.Fatal("fallback leaked the cause")
	}
}

func TestErrorsProductionHidesCause(t *testing.T) {
	rec := serve(func(c *core.Context) core.Response {
		return c.Error(core.Internal(errors.New("secret detail")))
	}, core.Errors(core.Options{Dev: false}))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret detail") {
		t.Fatal("production leaked the cause")
	}
}

func TestErrorsDevShowsCause(t *testing.T) {
	rec := serve(func(c *core.Context) core.Response {
		return c.Error(core.Internal(errors.New("secret detail")))
	}, core.Errors(core.Options{Dev: true}))
	if !strings.Contains(rec.Body.String(), "secret detail") {
		t.Fatalf("dev page missing cause: %q", rec.Body.String())
	}
}

func TestErrorsRecoversPanic(t *testing.T) {
	rec := serve(func(c *core.Context) core.Response {
		panic("kaboom")
	}, core.Errors(core.Options{Dev: true}))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "kaboom") {
		t.Fatalf("dev page missing panic value: %q", rec.Body.String())
	}
}

func TestErrorsUsesAppErrorView(t *testing.T) {
	fsys := fstest.MapFS{
		"layouts/base.html": {Data: []byte(`{{define "base"}}{{block "content" .}}{{end}}{{end}}`)},
		"errors/404.html":   {Data: []byte(`{{define "content"}}custom not found{{end}}`)},
	}
	set, err := views.New(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := serve(func(c *core.Context) core.Response {
		return c.Error(core.NotFound(nil))
	}, core.Errors(core.Options{Dev: false}), views.Middleware(set))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "custom not found") {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestLoggerSetsRequestIDAndLogs(t *testing.T) {
	var buf bytes.Buffer
	lg := slog.New(slog.NewTextHandler(&buf, nil))
	rec := serve(func(c *core.Context) core.Response {
		return core.String(http.StatusTeapot, "hi")
	}, core.Logger(core.Options{Logger: lg}))
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing X-Request-Id header")
	}
	if !strings.Contains(buf.String(), "status=418") {
		t.Fatalf("log = %q", buf.String())
	}
}
