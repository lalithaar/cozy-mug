// Package handlers holds the app's handlers, loaders, and view models.
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lalithaar/cozy-mug/auth"
	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/form"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/session"
)

// Note is a row in the notes table.
type Note struct {
	data.Model `db:"notes"`
	ID         int64     `db:"id auto"`
	Title      string    `db:"title" required:"" min:"3"`
	Body       string    `db:"body"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
	Version    int64     `db:"version"`
}

var noteCols = struct {
	ID, Title, Body data.Col
}{
	ID:    data.C[Note]("id"),
	Title: data.C[Note]("title"),
	Body:  data.C[Note]("body"),
}

// User is a row in the users table (app-owned, auth stays opt-in).
type User struct {
	data.Model `db:"users"`
	ID         int64     `db:"id auto"`
	Email      string    `db:"email unique" required:"" email:""`
	Password   string    `db:"-" form:"password" required:"" min:"8"`
	PasswordHash string  `db:"password_hash"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
	Version    int64     `db:"version"`
}

// UserLoader resolves a user for the auth processor.
func UserLoader(ctx context.Context, id any) (any, error) {
	var uid int64
	switch v := id.(type) {
	case int64:
		uid = v
	case float64:
		uid = int64(v)
	case int:
		uid = int64(v)
	default:
		return nil, nil
	}
	u, err := (&Store{db: nil}).userByID(ctx, uid)
	_ = u
	_ = err
	// A real loader needs the db; the processor below captures it. Kept here as
	// a type-compatible placeholder for template readers; overridden below.
	return nil, nil
}

type Store struct {
	db *data.DB
}

func NewStore(db *data.DB) *Store { return &Store{db: db} }

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

func (s *Store) Register(r *router.Router) {
	notes := r.Group("/notes")
	notes.Get("/{id}", s.showNote).As("note")

	r.Get("/", s.index).As("home")
	r.Post("/notes", s.createNote).As("notes.create")
	r.Get("/login", s.loginForm).As("login")
	r.Post("/login", s.login).As("login.do")
	r.Get("/register", s.registerForm).As("register")
	r.Post("/register", s.register).As("register.do")
	r.Post("/logout", s.logout).As("logout")
	r.NotFound(s.notFound)
}

type homeView struct {
	core.Globals
	Notes []Note
	User  any
	Form  *form.Form[Note]
}

type noteView struct {
	core.Globals
	Note Note
}

type authView struct {
	core.Globals
	Error string
}

func (s *Store) CurrentUserLoader() auth.UserLoader {
	return func(ctx context.Context, id any) (any, error) {
		var uid int64
		switch v := id.(type) {
		case int64:
			uid = v
		case float64:
			uid = int64(v)
		case int:
			uid = int64(v)
		}
		return s.userByID(ctx, uid)
	}
}

func (s *Store) index(c *core.Context) core.Response {
	notes, err := s.allNotes(c.R.Context())
	if err != nil {
		return c.Error(err)
	}
	return c.Render(http.StatusOK, "home", &homeView{Notes: notes})
}

func (s *Store) createNote(c *core.Context) core.Response {
	f := form.Bind[Note](c)
	if !f.Valid() {
		notes, _ := s.allNotes(c.R.Context())
		return c.Render(http.StatusUnprocessableEntity, "home", &homeView{Notes: notes, Form: f})
	}
	n := f.Value
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

func (s *Store) loginForm(c *core.Context) core.Response {
	return c.Render(http.StatusOK, "auth/login", &authView{})
}

func (s *Store) login(c *core.Context) core.Response {
	email := strings.TrimSpace(c.Form("email"))
	password := c.Form("password")
	u, err := s.userByEmail(c.R.Context(), email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, password) {
		return c.Render(http.StatusUnprocessableEntity, "auth/login", &authView{Error: "Invalid email or password."})
	}
	if err := auth.Login(c, u.ID); err != nil {
		return c.Error(err)
	}
	if sess := session.From(c); sess != nil {
		sess.Set("flash", "Welcome back.")
	}
	return core.Redirect(http.StatusSeeOther, "/")
}

func (s *Store) registerForm(c *core.Context) core.Response {
	return c.Render(http.StatusOK, "auth/register", &authView{})
}

func (s *Store) register(c *core.Context) core.Response {
	email := strings.TrimSpace(c.Form("email"))
	password := c.Form("password")
	if email == "" || len(password) < 8 {
		return c.Render(http.StatusUnprocessableEntity, "auth/register", &authView{Error: "Email and an 8+ character password are required."})
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return c.Error(err)
	}
	u := User{Email: email, PasswordHash: hash}
	if err := s.db.Table[User]().Create(c.R.Context(), &u); err != nil {
		return c.Render(http.StatusUnprocessableEntity, "auth/register", &authView{Error: "Could not create account (email may already be used)."})
	}
	if err := auth.Login(c, u.ID); err != nil {
		return c.Error(err)
	}
	if sess := session.From(c); sess != nil {
		sess.Set("flash", "Account created.")
	}
	return core.Redirect(http.StatusSeeOther, "/")
}

func (s *Store) logout(c *core.Context) core.Response {
	_ = auth.Logout(c)
	if sess := session.From(c); sess != nil {
		sess.Set("flash", "You have been logged out.")
	}
	return core.Redirect(http.StatusSeeOther, "/")
}

func (s *Store) notFound(c *core.Context) core.Response { return c.Error(core.NotFound(nil)) }

func (s *Store) allNotes(ctx context.Context) ([]Note, error) {
	return s.db.Table[Note]().OrderBy(noteCols.ID.Asc()).All(ctx)
}

func (s *Store) noteByID(ctx context.Context, id int64) (Note, error) {
	return s.db.Table[Note]().Where(noteCols.ID.Eq(id)).One(ctx)
}

func (s *Store) userByEmail(ctx context.Context, email string) (User, error) {
	// Column token via direct where for email.
	return s.db.Table[User]().Where(data.C[User]("email").Eq(email)).One(ctx)
}

func (s *Store) userByID(ctx context.Context, id int64) (User, error) {
	return s.db.Table[User]().Where(data.C[User]("id").Eq(id)).One(ctx)
}

// Processors for the auth template.
func Processors() []core.Processor { return []core.Processor{flash} }

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

