package form

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
)

type note struct {
	data.Model `db:"notes"`
	ID         int64  `db:"id auto"`
	Title      string `db:"title" required:"" min:"3"`
	Count      int    `db:"count"`
}

func (n *note) Validate() []core.FieldError {
	if strings.Contains(strings.ToLower(n.Title), "spam") {
		return []core.FieldError{{Field: "title", Message: "Title looks like spam"}}
	}
	return nil
}

func TestBindValidation(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("title=ab&count=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := &core.Context{W: httptest.NewRecorder(), R: req}

	f := Bind[note](c)
	if f.Valid() {
		t.Fatal("expected invalid")
	}
	if f.RawValues["title"] != "ab" {
		t.Fatalf("raw title = %q", f.RawValues["title"])
	}
	if f.Error("title") == "" {
		t.Fatal("expected title error")
	}
	if !strings.Contains(f.Error("count"), "whole number") {
		t.Fatalf("count error = %q", f.Error("count"))
	}
	// invalid value is not silently accepted for int
	if f.Value.Count != 0 {
		t.Fatalf("count = %d", f.Value.Count)
	}
}

func TestBindValid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("title=hello&count=5"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := &core.Context{W: httptest.NewRecorder(), R: req}

	f := Bind[note](c)
	if !f.Valid() {
		t.Fatalf("unexpected invalid: %+v", f.Errors(""))
	}
	if f.Value.Title != "hello" || f.Value.Count != 5 {
		t.Fatalf("value = %+v", f.Value)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("title=spammy"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c2 := &core.Context{W: httptest.NewRecorder(), R: req2}
	f2 := Bind[note](c2)
	if f2.Valid() {
		t.Fatal("hook should invalidate")
	}
	if f2.Error("title") != "Title looks like spam" {
		t.Fatalf("hook error = %q", f2.Error("title"))
	}
}

func TestNewSeedsRaw(t *testing.T) {
	var n note
	n.Title = "hello"
	n.Count = 7
	f := New(n)
	if f.RawValues["title"] != "hello" || f.RawValues["count"] != "7" {
		t.Fatalf("raw = %+v", f.RawValues)
	}
	if f.Valid() != true {
		t.Fatal("expected valid")
	}
}

func TestAddError(t *testing.T) {
	f := New(note{})
	f.AddError("title", "boom")
	if !f.HasError("title") {
		t.Fatal("expected error")
	}
	if f.Valid() {
		t.Fatal("should be invalid")
	}
}
