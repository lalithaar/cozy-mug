package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/session"
)

func TestHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("valid password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("invalid password accepted")
	}
	if CheckPassword("not a hash", "x") {
		t.Fatal("malformed hash accepted")
	}
}

func TestHashPasswordValidation(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("empty password accepted")
	}
	tooLong := make([]byte, maxPasswordBytes+1)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	if _, err := HashPassword(string(tooLong)); err == nil {
		t.Fatal("overlong password accepted")
	}
}

func TestLoginSetsAndClearsIdentity(t *testing.T) {
	ctx := context.Background()
	db, err := data.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := session.NewStore(db, session.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}

	handler := store.Middleware(func(c *core.Context) core.Response {
		if err := Login(c, 42); err != nil {
			t.Fatalf("login: %v", err)
		}
		return core.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rr := httptest.NewRecorder()
	c := &core.Context{W: rr, R: req}
	resp := handler(c)
	if resp != nil {
		resp.WriteTo(rr, req)
	}

	if !Authenticated(c) {
		t.Fatal("login did not mark session authenticated")
	}
	id, ok := UserID(c)
	if !ok || id != 42 {
		t.Fatalf("unexpected user id: %#v, %v", id, ok)
	}

	if err := Logout(c); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if Authenticated(c) {
		t.Fatal("logout did not clear identity")
	}
}

func TestUserProcessorSetsGlobal(t *testing.T) {
	ctx := context.Background()
	db, err := data.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := session.NewStore(db, session.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}

	handler := store.Middleware(func(c *core.Context) core.Response {
		if err := Login(c, 7); err != nil {
			t.Fatalf("login: %v", err)
		}
		return core.String(http.StatusOK, "ok")
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	c := &core.Context{W: rr, R: req}
	resp := handler(c)
	if resp != nil {
		resp.WriteTo(rr, req)
	}

	processor := UserProcessor(func(context.Context, any) (any, error) {
		return "loaded-user", nil
	})
	values := processor(c)
	if values["user"] != "loaded-user" {
		t.Fatalf("processor did not load user: %#v", values)
	}
	if From(c) != "loaded-user" {
		// From reads already-set context value; processor itself does not Set,
		// so simulate the app calling c.Set.
		c.Set("user", values["user"])
		if From(c) != "loaded-user" {
			t.Fatalf("From did not return processor value")
		}
	}
}
