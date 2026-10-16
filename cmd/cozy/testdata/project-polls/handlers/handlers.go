// Package handlers holds the polls app's handlers, loaders, view models, and
// route registration. It demonstrates the conventional cozy-mug app layout:
// models with data.Model, typed column tokens, form binding, server-rendered
// views, and route groups.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
	"github.com/lalithaar/cozy-mug/router"
	"github.com/lalithaar/cozy-mug/session"
	"github.com/lalithaar/cozy-mug/upload"
)

// Question is a poll question.
type Question struct {
	data.Model   `db:"questions"`
	ID           int64     `db:"id auto"`
	QuestionText string    `db:"question_text" required:""`
	PubDate      time.Time `db:"pub_date"`
	Choices      []*Choice `db:"ref:question_id"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	Version      int64     `db:"version"`
}

// Choice is one answer choice for a question.
type Choice struct {
	data.Model   `db:"choices"`
	ID           int64     `db:"id auto"`
	QuestionID   int64     `db:"question_id" required:""`
	ChoiceText   string    `db:"choice_text" required:""`
	Votes        int64     `db:"votes"`
	Question     *Question `db:"ref:question_id"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	Version      int64     `db:"version"`
}

// Column tokens for compile-checked queries.

var (
	questionCols = struct {
		ID, QuestionText, PubDate data.Col
	}{
		ID:           data.C[Question]("id"),
		QuestionText: data.C[Question]("question_text"),
		PubDate:      data.C[Question]("pub_date"),
	}

	choiceCols = struct {
		ID, QuestionID, ChoiceText, Votes data.Col
	}{
		ID:          data.C[Choice]("id"),
		QuestionID:  data.C[Choice]("question_id"),
		ChoiceText:  data.C[Choice]("choice_text"),
		Votes:       data.C[Choice]("votes"),
	}
)

// Store holds the app's data access and route registration.
type Store struct {
	db      *data.DB
	Uploads *upload.Local
}

func NewStore(db *data.DB, uploads *upload.Local) *Store {
	return &Store{db: db, Uploads: uploads}
}

// Seed ensures there is at least one question in the database.
func (s *Store) Seed(ctx context.Context) error {
	n, err := s.db.Table[Question]().Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	q := &Question{QuestionText: "What is your favorite programming language?"}
	if err := s.db.Table[Question]().Create(ctx, q); err != nil {
		return err
	}
	for _, text := range []string{"Go", "Python", "JavaScript", "Rust"} {
		c := &Choice{QuestionID: q.ID, ChoiceText: text}
		if err := s.db.Table[Choice]().Create(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// Register wires the polls app's routes.
func (s *Store) Register(r *router.Router) {
	r.Get("/", s.index).As("polls.index")

	polls := r.Group("/polls")
	polls.Get("/{id}", s.detail).As("polls.detail")
	polls.Post("/{id}/vote", s.vote).As("polls.vote")
	polls.Get("/{id}/results", s.results).As("polls.results")

	r.NotFound(s.notFound)
}

// index lists the latest questions.
func (s *Store) index(c *core.Context) core.Response {
	questions, err := s.db.Table[Question]().OrderBy(questionCols.ID.Desc()).All(c.R.Context())
	if err != nil {
		return c.Error(err)
	}
	return c.Render(http.StatusOK, "polls/index", &indexView{
		Questions: questions,
	})
}

// detail shows a question and its choices with a vote form.
func (s *Store) detail(c *core.Context) core.Response {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.Error(core.NotFound(err))
	}
	q, err := s.db.Table[Question]().Where(questionCols.ID.Eq(id)).With(data.R[Question]("Choices")).One(c.R.Context())
	if err != nil {
		return c.Error(err)
	}
	return c.Render(http.StatusOK, "polls/detail", &detailView{
		Question: q,
	})
}

// vote records a vote for a choice.
func (s *Store) vote(c *core.Context) core.Response {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.Error(core.NotFound(err))
	}
	choiceID, err := strconv.ParseInt(c.Form("choice"), 10, 64)
	if err != nil || choiceID == 0 {
		return c.Error(core.BadRequest(fmt.Errorf("no choice selected")))
	}
	ch, err := s.db.Table[Choice]().Where(choiceCols.ID.Eq(choiceID)).One(c.R.Context())
	if err != nil {
		return c.Error(err)
	}
	if ch.QuestionID != id {
		return c.Error(core.BadRequest(fmt.Errorf("choice does not belong to this question")))
	}
	ch.Votes++
	if err := s.db.Table[Choice]().Update(c.R.Context(), &ch); err != nil {
		return c.Error(err)
	}
	sess := session.From(c)
	if sess != nil {
		sess.Set("flash", "Vote recorded.")
	}
	return core.Redirect(http.StatusSeeOther, s.resultsURL(id))
}

// results shows the voting results for a question.
func (s *Store) results(c *core.Context) core.Response {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.Error(core.NotFound(err))
	}
	q, err := s.db.Table[Question]().Where(questionCols.ID.Eq(id)).With(data.R[Question]("Choices")).One(c.R.Context())
	if err != nil {
		return c.Error(err)
	}
	return c.Render(http.StatusOK, "polls/results", &resultsView{
		Question: q,
	})
}

func (s *Store) resultsURL(id int64) string {
	return "/polls/" + strconv.FormatInt(id, 10) + "/results"
}

func (s *Store) notFound(c *core.Context) core.Response {
	return c.Error(core.NotFound(nil))
}

// View models.

type indexView struct {
	core.Globals
	Questions []Question
}

type detailView struct {
	core.Globals
	Question Question
}

type resultsView struct {
	core.Globals
	Question Question
}

// session is the request session seam, provided by the session middleware.
var _ = session.From
