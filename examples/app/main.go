// Command app is the canonical cozy-mug example: a small server-rendered notes
// app. It is the living reference for the framework's opinions — it follows the
// app layout (0009) and grows as the framework does.
package main

import (
	"embed"
	"log"
	"os"

	"github.com/lalithaar/cozy-mug/app"
	"github.com/lalithaar/cozy-mug/examples/app/handlers"
)

//go:embed views static
var assets embed.FS

func main() {
	a := app.Must(app.Config{
		Assets: assets,                  // views/ (+ static/ at /static/)
		Routes: handlers.Register,       // func(*router.Router)
		Dev:    os.Getenv("DEV") != "0", // non-production unless DEV=0
		UI:     true,                    // opt into the (scaffold) ui library
	})
	log.Fatal(a.ListenAndServe(":8080"))
}
