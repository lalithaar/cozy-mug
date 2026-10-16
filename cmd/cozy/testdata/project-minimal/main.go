// Command app is a minimal cozy-mug starter: a single server-rendered page, no
// database. Grow it by adding models, handlers, and views under handlers/views.
package main

import (
	"embed"
	"log"
	"os"

	"__MODULE__/handlers"
	"github.com/lalithaar/cozy-mug/app"
)

//go:embed views
var assets embed.FS

func main() {
	a := app.Must(app.Config{
		Assets:  assets,
		Routes:  handlers.Register,
		Dev:     os.Getenv("DEV") != "0",
		UI:      true,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := a.ListenAndServe(":" + port); err != nil {
		log.Fatal(err)
	}
}