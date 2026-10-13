// Package handlers holds the example app's handlers, loaders, and view models —
// the feature-by-feature shape an app follows (0009).
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/form"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/session"
)

// Note is a row in the notes table. The model declaration is the single source
// of truth (0016): the framework derives the schema, the seed/CRUD SQL, and the
// optimistic-lock version guard from it. The embedded data.Model tag names the
// table; created_at/updated_at/version are maintained automatically.
type Note struct {
	data.Model `db:"notes"`
	ID         int64     `db:"id auto"`
	Title      string    `db:"title" required:"" min:"3"`
	Body       string    `db:"body"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
	Version    int64     `db:"version"`
}

// noteCols are typed column tokens — what cozy gen will emit so query builders
// stay catch-at-compile-time (0016).
var noteCols = struct {
	ID, Title, Body data.Col
}{
	ID:    data.C[Note]("id"),
	Title: data.C[Note]("title"),
	Body:  data.C[Note]("body"),
}

// Store holds the app's data access and its routes. Carrying one value and
// handing its Register method to app.Config is how the example does dependency
// injection: Register closes over the DB (0009).
type Store struct {
	db *data.DB
}

func NewStore(db *data.DB) *Store { return &Store{db: db} }

// Seed inserts the example's demo notes when the notes table is empty. It
// assumes the table already exists — schema comes from migrations.
func (s *Store) Seed(ctx context.Context) error {
	n, err := s.db.Table[Note]().Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	for _, m := range []Note{
		{Title: "First note", Body: "Hello from the cozy-mug example."},
		{Title: "Second note", Body: "Frugal and server-rendered.\nNo build step."},
	} {
		if err := s.db.Table[Note]().Create(ctx, &m); err != nil {
			return err
		}
	}
	return nil
}

// Register wires the example's routes; pass store.Register as app.Config.Routes.
func (s *Store) Register(r *router.Router) {
	notesGroup := r.Group("/notes")
	notesGroup.Use(withExampleHeader)
	notesGroup.Get("/{id}", s.showNote).As("note")

	r.Get("/", s.index).As("home")
	r.Post("/notes", s.createNote).As("notes.create")
	r.Get("/boom", boom)
	r.NotFound(notFound)
}

// View models embed core.Globals so templates can build reverse URLs.

type homeView struct {
	core.Globals
	Notes []Note
	Form  *form.Form[Note]
}

type noteView struct {
	core.Globals
	Note Note
}

func (s *Store) index(c *core.Context) core.Response {
	notes, err := s.allNotes(c.R.Context())
	if err != nil {
		return c.Error(err)
	}
	// The flash is in .Values, put there by the flash processor (gap 7).
	return c.Render(http.StatusOK, "home", &homeView{Notes: notes})
}

// createNote handles the form POST. It is CSRF-protected by the session
// middleware, which rejects the request before this runs if the token is wrong.
func (s *Store) createNote(c *core.Context) core.Response {
	f := form.Bind[Note](c)
	if !f.Valid() {
		notes, err := s.allNotes(c.R.Context())
		if err != nil {
			return c.Error(err)
		}
		return c.Render(http.StatusUnprocessableEntity, "home", &homeView{Notes: notes, Form: f})
	}
	n := f.Value
	if strings.TrimSpace(n.Body) != n.Body {
		n.Body = strings.TrimSpace(n.Body)
	}
	if err := s.db.Table[Note]().Create(c.R.Context(), &n); err != nil {
		f.AddError("", "Could not save the note.")
		notes, _ := s.allNotes(c.R.Context())
		return c.Render(http.StatusUnprocessableEntity, "home", &homeView{Notes: notes, Form: f})
	}
	if sess := session.From(c); sess != nil {
		sess.Set("flash", "Note added.")
	}
	return core.Redirect(http.StatusSeeOther, "/")
}

func (s *Store) showNote(c *core.Context) core.Response {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.Error(core.NotFound(err))
	}
	n, err := s.noteByID(c.R.Context(), id)
	if err != nil {
		return c.Error(err)
	}
	return c.Render(http.StatusOK, "notes/show", &noteView{Note: n})
}

func boom(*core.Context) core.Response { panic("example panic") }

func notFound(c *core.Context) core.Response { return c.Error(core.NotFound(nil)) }

// Processors are the app's global context processors (gap 7): values every view
// sees as {{.Values.name}}. Hand them to app.Config.
func Processors() []core.Processor { return []core.Processor{flash} }

// flash moves a one-shot flash message out of the session and into the view
// values, deleting it so it renders once. It runs after sessions, so the delete
// is persisted by the session middleware.
func flash(c *core.Context) map[string]any {
	s := session.From(c)
	if s == nil {
		return nil
	}
	msg := s.GetString("flash")
	if msg == "" {
		return nil
	}
	s.Delete("flash")
	return map[string]any{"flash": msg}
}

// allNotes and noteByID are loaders: plain Go returning status-aware errors,
// never writing a response and never panicking (0007).
func (s *Store) allNotes(ctx context.Context) ([]Note, error) {
	return s.db.Table[Note]().OrderBy(noteCols.ID.Asc()).All(ctx)
}

func (s *Store) noteByID(ctx context.Context, id int64) (Note, error) {
	// One maps a missing row to core.NotFound, so a bad id renders the 404.
	return s.db.Table[Note]().Where(noteCols.ID.Eq(id)).One(ctx)
}

// withExampleHeader tags the /notes group's responses; it shows group middleware.
func withExampleHeader(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		c.W.Header().Set("X-Example", "cozy-mug")
		return next(c)
	}
}
