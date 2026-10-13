// Command app is the canonical cozy-mug example: a small server-rendered notes
// app. It is the living reference for the framework's opinions — it follows the
// app layout (0009) and grows as the framework does.
package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"__MODULE__/handlers"
	"github.com/lalithaar/cozy-mug/funcs"
	"github.com/lalithaar/cozy-mug/session"
)

//go:embed views static
var assets embed.FS

func main() {
	ctx := context.Background()

	// SQLite by default; set DATABASE_URL=postgres://... to use Postgres. The DSN
	// is read from the environment and is never logged (secrets heuristic).
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "notes.db"
	}
	db := core.Must(data.Open(ctx, dsn))
	defer db.Close()

	// Schema comes from the model (0016): EnsureSchema creates tables + indexes,
// idempotently (IF NOT EXISTS). It is the interim boot-time creator — real
// deployments replace it with the generated-migrations runner (cozy migrate:
// reviewable per-dialect artifacts, explicit step, never at boot).
store := handlers.NewStore(db)
if err := store.EnsureSchema(ctx); err != nil {
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
	log.Fatal(a.ListenAndServe(":8080"))
}
