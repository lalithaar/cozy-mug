// Command app is the canonical cozy-mug example: a polls application.
// It demonstrates models, forms, sessions, CSRF, and server-rendered HTML.
// Run `go run .` to start the server.
package main

import (
	"context"
	"embed"
	"io"
	"log"
	"os"

	"__MODULE__/handlers"
	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/auth"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/funcs"
	"github.com/lalithaar/cozy-mug/session"
	"github.com/lalithaar/cozy-mug/upload"
)

//go:embed views static
var assets embed.FS

// publicStoreAdapter adapts *upload.Local to upload.PublicStore.
// The framework's Local.Open does not take a context, while PublicStore
// requires one; the adapter bridges that without touching the framework.
type publicStoreAdapter struct {
	*upload.Local
}

func (a *publicStoreAdapter) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return a.Local.Open(key)
}

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "polls.db"
	}
	db := core.Must(data.Open(ctx, dsn))
	defer db.Close()

	store := handlers.NewStore(db, upload.New("uploads", "/uploads/"))

	// Apply the model-derived schema on boot so `go run .` works immediately.
	// In production, prefer `cozy migrate up` with reviewable migration files
	// under migrations/<dialect>/ instead.
	models := []any{&handlers.Question{}, &handlers.Choice{}}
	for _, m := range models {
		stmts, err := data.SchemaSQLFor(db, m)
		if err != nil {
			log.Fatalf("schema: %v", err)
		}
		for _, s := range stmts {
			if _, err := db.ExecContext(ctx, s); err != nil {
				log.Fatalf("schema: %v", err)
			}
		}
	}
	if err := store.Seed(ctx); err != nil {
		log.Fatal(err)
	}

	sessions := core.Must(session.NewStore(db, session.Options{
		Secure: os.Getenv("DEV") == "0",
	}))
	if err := sessions.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}

	a := app.Must(app.Config{
		Assets:     assets,
		Routes:     store.Register,
		Dev:        os.Getenv("DEV") == "1",
		UI:         true,
		Funcs:      funcs.Funcs(),
		Session:    sessions,
		Flash:      true,
		Processors: []core.Processor{auth.UserProcessor(nil)},
		Uploads:    &publicStoreAdapter{Local: store.Uploads},
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := a.ListenAndServe(":" + port); err != nil {
		log.Fatal(err)
	}
}
