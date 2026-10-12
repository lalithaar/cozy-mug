// Package handlers holds the example app's handlers, loaders, and view models —
// the feature-by-feature shape an app follows (0009).
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/session"
)

// Note is a row in the notes table. db tags map columns to fields; the scanner
// matches by name case-insensitively when a tag is absent.
type Note struct {
	ID    int64  `db:"id"`
	Title string `db:"title"`
	Body  string `db:"body"`
}

// Store holds the app's data access and its routes. Carrying one value and
// handing its Register method to app.Config is how the example does dependency
// injection: Register closes over the DB (0009).
type Store struct {
	db *data.DB
}

func NewStore(db *data.DB) *Store { return &Store{db: db} }

// EnsureSchema creates the table if missing and seeds a couple of notes. An app
// with migrations would run them here instead (migrations are deferred).
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := data.Exec(ctx, s.db, `create table if not exists notes (
		id    integer primary key autoincrement,
		title text not null,
		body  text not null
	)`); err != nil {
		return err
	}

	n, err := data.One[int](ctx, s.db, `select count(*) from notes`)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	_, err = data.Exec(ctx, s.db,
		`insert into notes (title, body) values (?, ?), (?, ?)`,
		"First note", "Hello from the cozy-mug example.",
		"Second note", "Frugal and server-rendered.\nNo build step.",
	)
	return err
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
	title := strings.TrimSpace(c.Form("title"))
	body := strings.TrimSpace(c.Form("body"))
	if title == "" {
		return c.Error(core.BadRequest(errors.New("title is required")))
	}
	if _, err := data.Exec(c.R.Context(), s.db,
		`insert into notes (title, body) values (?, ?)`, title, body); err != nil {
		return c.Error(err)
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
	return data.All[Note](ctx, s.db, `select id, title, body from notes order by id`)
}

func (s *Store) noteByID(ctx context.Context, id int64) (Note, error) {
	// One maps sql.ErrNoRows to core.NotFound, so a missing id renders the 404.
	return data.One[Note](ctx, s.db, `select id, title, body from notes where id = ?`, id)
}

// withExampleHeader tags the /notes group's responses; it shows group middleware.
func withExampleHeader(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		c.W.Header().Set("X-Example", "cozy-mug")
		return next(c)
	}
}
