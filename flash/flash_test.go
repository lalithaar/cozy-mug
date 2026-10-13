package flash

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/session"
)

func newStore(t *testing.T) *session.Store {
	t.Helper()
	db, err := data.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := session.NewStore(db, session.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestMessagesAcrossRequests(t *testing.T) {
	store := newStore(t)

	handler := store.Middleware(func(c *core.Context) core.Response {
		Success(c, "hello")
		return core.Redirect(http.StatusSeeOther, "/next")
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	c := &core.Context{W: rr, R: req}
	resp := handler(c)
	if resp != nil {
		resp.WriteTo(rr, req)
	}

	follow := store.Middleware(func(c *core.Context) core.Response {
		msgs := Messages(c)
		if len(msgs) != 1 || msgs[0].Text != "hello" || msgs[0].Level != "success" {
			t.Fatalf("messages = %+v", msgs)
		}
		if again := Messages(c); len(again) != 0 {
			t.Fatalf("messages were not consumed: %+v", again)
		}
		return core.String(http.StatusOK, "ok")
	})
	req2 := httptest.NewRequest(http.MethodGet, "/next", nil)
	// Carry the cookie set by the post response back into the follow-up request.
	for _, ck := range rr.Result().Cookies() {
		req2.AddCookie(ck)
	}
	rr2 := httptest.NewRecorder()
	c2 := &core.Context{W: rr2, R: req2}
	resp2 := follow(c2)
	if resp2 != nil {
		resp2.WriteTo(rr2, req2)
	}
}

func TestProcessorExposesMessages(t *testing.T) {
	store := newStore(t)

	handler := store.Middleware(func(c *core.Context) core.Response {
		Info(c, "queued")
		return core.String(http.StatusOK, "ok")
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	c := &core.Context{W: rr, R: req}
	resp := handler(c)
	if resp != nil {
		resp.WriteTo(rr, req)
	}

	processor := Processor()
	followReq := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, ck := range rr.Result().Cookies() {
		followReq.AddCookie(ck)
	}
	followRR := httptest.NewRecorder()
	followC := &core.Context{W: followRR, R: followReq}
	store.Middleware(func(c *core.Context) core.Response {
		values := processor(c)
		msgs, ok := values["flash"].([]Message)
		if !ok || len(msgs) != 1 || msgs[0].Text != "queued" {
			t.Fatalf("processor value = %#v", values)
		}
		return core.String(http.StatusOK, "ok")
	})(followC)
}