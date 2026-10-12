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
	"github.com/lalithaar/cozy-mug/examples/app/handlers"
	"github.com/lalithaar/cozy-mug/funcs"
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

	// Schema/migrations are the app's job; data.Open only wires the connection.
	store := handlers.NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}

	a := app.Must(app.Config{
		Assets: assets,                  // views/ (+ static/ at /static/)
		Routes: store.Register,          // func(*router.Router), closes over the DB
		Dev:    os.Getenv("DEV") != "0", // non-production unless DEV=0
		UI:     true,                    // opt into the (scaffold) ui library
		Funcs:  funcs.Funcs(),           // opt into the loop/arithmetic template funcs
	})
	log.Fatal(a.ListenAndServe(":8080"))
}
