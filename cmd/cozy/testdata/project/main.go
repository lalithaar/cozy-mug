// Command app is the canonical cozy-mug example: a small server-rendered notes
// app. It is the living reference for the framework's opinions — it follows the
// app layout (0009) and grows as the framework does.
package main

import (
	"context"
	"embed"
	"log"
	"os"

	"__MODULE__/handlers"
	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/funcs"
	"github.com/lalithaar/cozy-mug/session"
)

//go:embed views
var assets embed.FS

func main() {
	ctx := context.Background()

	// SQLite by default; set DATABASE_URL=postgres://... to use Postgres. The DSN
	// is read from the environment and is never logged (secrets heuristic).
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "app.db"
	}
	db := core.Must(data.Open(ctx, dsn))
	defer db.Close()

	store := handlers.NewStore(db)
	// Schema is managed by migrations (cozy migrate), never at boot (0016).
	// Seed only fills demo rows; it expects the tables to already exist.
	if err := store.Seed(ctx); err != nil {
		log.Fatal(err)
	}

	// Sessions + CSRF, kept server-side in the database (ADR 0013): the cookie
	// holds only an opaque id, so sessions are revocable and there is no global
	// key to leak. The framework owns this one table (not an app migration).
	sessions := core.Must(session.NewStore(db, session.Options{
		Secure: os.Getenv("DEV") == "0",
	}))
	if err := sessions.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}

	a := app.Must(app.Config{
		Assets:     assets,                  // views/ (+ static/ at /static/)
		Routes:     store.Register,          // func(*router.Router), closes over the DB
		Dev:        os.Getenv("DEV") != "0", // non-production unless DEV=0
		UI:         true,                    // opt into the (scaffold) ui library
		Funcs:      funcs.Funcs(),           // opt into the loop/arithmetic template funcs
		Session:    sessions,                // sessions + CSRF, on by default
		Processors: handlers.Processors(),   // flash (and future current-user) in every view
	})
	if err := a.ListenAndServe(":8080"); err != nil {
		log.Fatal(err)
	}
}
