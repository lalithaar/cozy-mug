// Package test provides helpers for building a cozy-mug app in tests, so
// integration tests don't have to repeat in-memory DB setup, schema
// application, session wiring, and app assembly (P8).
//
// NewApp opens an in-memory SQLite database, applies the schema returned by
// schemaFunc, creates a session store, and assembles an app with the given
// routes. It returns the app, DB, and session store so the caller can drive
// requests with httptest and inspect state directly.
//
// Example:
//
//	a, db, sess := test.NewApp(t, handlers.Register, func(db *data.DB) []string {
//	    stmts, _ := db.Table[handlers.Person]().SchemaSQL()
//	    return stmts
//	})
//	rec := httptest.NewRecorder()
//	a.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
package test

import (
	"context"
	"embed"
	"io/fs"
	"testing"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/session"
)

//go:embed views/*
var viewFS embed.FS

// NewApp builds a test app with an in-memory database, the schema returned by
// schemaFunc applied, sessions enabled, and the given routes registered. t
// must not be nil. Flash is enabled so {{.Values.flash}} is available in
// every view.
//
// The returned *app.App is an http.Handler, so tests drive it with httptest.
func NewApp(t *testing.T, routes func(*router.Router), schemaFunc func(*data.DB) []string) (*app.App, *data.DB, *session.Store) {
	t.Helper()
	ctx := context.Background()

	db, err := data.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Apply the model-derived schema.
	for _, s := range schemaFunc(db) {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("schema exec: %v", err)
		}
	}

	sessions, err := session.NewStore(db, session.Options{})
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if err := sessions.EnsureSchema(ctx); err != nil {
		t.Fatalf("session schema: %v", err)
	}

	assetFS, err := fs.Sub(viewFS, "views")
	if err != nil {
		t.Fatalf("assets: %v", err)
	}

	a := app.Must(app.Config{
		Assets:     assetFS,
		Routes:     routes,
		Dev:        true,
		Session:    sessions,
		Flash:      true,
		Processors: []core.Processor{},
	})

	return a, db, sessions
}
