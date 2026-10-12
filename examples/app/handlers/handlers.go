// Package handlers holds the example app's handlers, loaders, and view models —
// the feature-by-feature shape an app follows (0009).
package handlers

import (
	"errors"
	"net/http"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
)

// Note is an in-memory record; the example has no database yet.
type Note struct {
	ID    string
	Title string
	Body  string
}

var notes = map[string]Note{
	"1": {ID: "1", Title: "First note", Body: "Hello from the cozy-mug example."},
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

// Register wires the example's routes; pass it as app.Config.Routes.
func Register(r *router.Router) {
	notesGroup := r.Group("/notes")
	notesGroup.Use(withExampleHeader)
	notesGroup.Get("/{id}", showNote).As("note")

	r.Get("/", index).As("home")
	r.Get("/boom", boom)
	r.NotFound(notFound)
}

func index(c *core.Context) core.Response {
	list := make([]Note, 0, len(notes))
	for _, n := range notes {
		list = append(list, n)
	}
	return c.Render(http.StatusOK, "home", &homeView{Notes: list})
}

func showNote(c *core.Context) core.Response {
	n, err := loadNote(c.Param("id"))
	if err != nil {
		return c.Error(err)
	}
	return c.Render(http.StatusOK, "notes/show", &noteView{Note: n})
}

func boom(*core.Context) core.Response { panic("example panic") }

func notFound(c *core.Context) core.Response { return c.Error(core.NotFound(nil)) }

// loadNote is a tier-2 loader: plain Go returning a status-aware error; it never
// writes a response and never panics (0007).
func loadNote(id string) (Note, error) {
	n, ok := notes[id]
	if !ok {
		return Note{}, core.NotFound(errors.New("note " + id + " not found"))
	}
	return n, nil
}

// withExampleHeader tags the /notes group's responses; it shows group middleware.
func withExampleHeader(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		c.W.Header().Set("X-Example", "cozy-mug")
		return next(c)
	}
}
