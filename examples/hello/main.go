package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/views"
)

//go:embed views
var viewFS embed.FS

// homeView embeds core.Globals so the template can generate URLs; the framework
// fills it in during c.Render.
type homeView struct{ core.Globals }

type userView struct {
	core.Globals
	ID string
}

// loadUser is a tier-2 loader: plain Go that returns a structured, status-aware
// error instead of writing a response or panicking.
func loadUser(c *core.Context) (userView, error) {
	id := c.Param("id")
	if id == "0" {
		return userView{}, core.NotFound(fmt.Errorf("user %q not found", id))
	}
	return userView{ID: id}, nil
}

func main() {
	viewsFS, err := fs.Sub(viewFS, "views")
	if err != nil {
		log.Fatal(err)
	}
	set := core.Must(views.New(viewsFS, nil))

	opts := core.Options{Dev: true, Logger: slog.New(slog.NewTextHandler(os.Stdout, nil))}

	r := router.New()
	r.Use(core.Logger(opts), core.Errors(opts), views.Middleware(set))

	r.Get("/", func(c *core.Context) core.Response {
		return c.Render(http.StatusOK, "home", &homeView{})
	}).As("home")

	r.Get("/users/{id}", func(c *core.Context) core.Response {
		u, err := loadUser(c)
		if err != nil {
			return c.Error(err) // → errors/404.html (dev shows detail)
		}
		return c.Render(http.StatusOK, "users/show", &u)
	}).As("user")

	r.Get("/boom", func(c *core.Context) core.Response {
		panic("demo panic")
	})

	r.NotFound(func(c *core.Context) core.Response {
		return c.Error(core.NotFound(nil))
	})

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
