package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"cozymug/core"
	"cozymug/router"
	"cozymug/views"
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

func main() {
	viewsFS, err := fs.Sub(viewFS, "views")
	if err != nil {
		log.Fatal(err)
	}
	set, err := views.New(viewsFS, nil)
	if err != nil {
		log.Fatal(err)
	}

	r := router.New()
	r.Use(views.Middleware(set))

	r.Get("/", func(c *core.Context) core.Response {
		return c.Render(http.StatusOK, "home", &homeView{})
	}).As("home")

	r.Get("/users/{id}", func(c *core.Context) core.Response {
		return c.Render(http.StatusOK, "users/show", &userView{ID: c.Param("id")})
	}).As("user")

	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
