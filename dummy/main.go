package main

import (
	"embed"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/views"
)

//go:embed views static
var assets embed.FS

// Note is a dummy record.
type Note struct {
	ID    string
	Title string
	Body  string
}

var notes = map[string]Note{
	"1": {ID: "1", Title: "First note", Body: "Hello from the dummy app."},
	"2": {ID: "2", Title: "Second note", Body: "Frugal and server-rendered."},
}

// View models embed core.Globals so templates can build reverse URLs.

type homeView struct {
	core.Globals
	Notes []Note
}

type noteView struct {
	core.Globals
	Note Note
}

// loadNote is a tier-2 loader: plain Go returning a status-aware error, never
// writing a response and never panicking.
func loadNote(id string) (Note, error) {
	n, ok := notes[id]
	if !ok {
		return Note{}, core.NotFound(errors.New("note " + id + " not found"))
	}
	return n, nil
}

func main() {
	viewFS := core.Must(fs.Sub(assets, "views"))
	staticFS := core.Must(fs.Sub(assets, "static"))
	set := core.Must(views.New(viewFS, nil))

	dev := os.Getenv("DEV") != "0"
	opts := core.Options{Dev: dev, Logger: slog.New(slog.NewTextHandler(os.Stdout, nil))}

	r := router.New()
	r.Use(core.Logger(opts), core.Errors(opts), views.Middleware(set))

	// A group under /notes with its own middleware.
	notesGroup := r.Group("/notes")
	notesGroup.Use(func(next core.Handler) core.Handler {
		return func(c *core.Context) core.Response {
			c.W.Header().Set("X-App", "dummy")
			return next(c)
		}
	})
	notesGroup.Get("/{id}", func(c *core.Context) core.Response {
		n, err := loadNote(c.Param("id"))
		if err != nil {
			return c.Error(err)
		}
		return c.Render(http.StatusOK, "notes/show", &noteView{Note: n})
	}).As("note")

	r.Get("/", func(c *core.Context) core.Response {
		list := make([]Note, 0, len(notes))
		for _, n := range notes {
			list = append(list, n)
		}
		return c.Render(http.StatusOK, "home", &homeView{Notes: list})
	}).As("home")

	r.StaticFS("/static/", staticFS)

	r.Get("/boom", func(*core.Context) core.Response { panic("dummy panic") })

	r.NotFound(func(c *core.Context) core.Response {
		return c.Error(core.NotFound(nil))
	})

	log.Printf("dummy app (dev=%v) on http://localhost:8080", dev)
	log.Fatal(http.ListenAndServe(":8080", r))
}
