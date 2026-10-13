package data_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
)

// ---------------------------------------------------------------------------
// Models — the declaration is the whole schema story (0016).
// ---------------------------------------------------------------------------

type Author struct {
	data.Model `db:"authors"`
	ID         int64  `db:"id auto"`
	Name       string `db:"name"`
}

type Comment struct {
	data.Model `db:"comments"`
	ID         int64  `db:"id auto"`
	PostID     int64  `db:"post_id index"`
	Post       *Post  `db:"ref:post_id"`
	Text       string `db:"text"`
	Version    int64  `db:"version"`
}

type Post struct {
	data.Model `db:"posts"`
	ID         int64      `db:"id auto"`
	Title      string     `db:"title"`
	Body       *string    `db:"body"`
	AuthorID   int64      `db:"author_id index"`
	Author     *Author    `db:"ref:author_id"`
	Comments   []*Comment `db:"ref:post_id"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	Version    int64      `db:"version"`
}

// Tokens — what the (future) cozy gen would emit for Post; hand-written here.
var toc = struct {
	ID, Title, Body, AuthorID, CreatedAt, UpdatedAt, Version data.Col
	Author, Comments                                         data.Rel
}{
	ID:        data.C[Post]("id"),
	Title:     data.C[Post]("title"),
	Body:      data.C[Post]("body"),
	AuthorID:  data.C[Post]("author_id"),
	CreatedAt: data.C[Post]("created_at"),
	UpdatedAt: data.C[Post]("updated_at"),
	Version:   data.C[Post]("version"),
	Author:    data.R[Post]("Author"),
	Comments:  data.R[Post]("Comments"),
}

var tcc = struct {
	ID, PostID, Text, Version data.Col
	Post                      data.Rel
}{
	ID:      data.C[Comment]("id"),
	PostID:  data.C[Comment]("post_id"),
	Text:    data.C[Comment]("text"),
	Version: data.C[Comment]("version"),
	Post:    data.R[Comment]("Post"),
}

func testDB(t *testing.T) *data.DB {
	t.Helper()
	db, err := data.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, fn := range []func(context.Context) error{
		db.Table[Author]().EnsureSchema,
		db.Table[Post]().EnsureSchema,
		db.Table[Comment]().EnsureSchema,
	} {
		if err := fn(context.Background()); err != nil {
			t.Fatalf("ensure schema: %v", err)
		}
	}
	return db
}

func createPost(t *testing.T, db *data.DB, author *Author, title string) *Post {
	t.Helper()
	p := &Post{Title: title, Body: nil, AuthorID: author.ID}
	if err := db.Table[Post]().Create(context.Background(), p); err != nil {
		t.Fatalf("create post: %v", err)
	}
	return p
}

// ---------------------------------------------------------------------------

func TestCreateAndLoad(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatalf("create author: %v", err)
	}
	if ada.ID == 0 {
		t.Fatal("auto pk not back-filled")
	}

	p := createPost(t, db, ada, "First")
	if p.ID == 0 || p.Version != 1 {
		t.Fatalf("create back-fill wrong: %+v", p)
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Fatal("timestamps not set on create")
	}

	got, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	if got.Title != "First" || got.Version != 1 || got.CreatedAt.IsZero() {
		t.Fatalf("loaded row wrong: %+v", got)
	}

	createPost(t, db, ada, "Second")
	n, err := db.Table[Post]().Count(ctx)
	if err != nil || n != 2 {
		t.Fatalf("count = %d, %v", n, err)
	}

	rows, total, err := db.Table[Post]().OrderBy(toc.ID.Asc()).Page(ctx, 2, 1)
	if err != nil || total != 2 || len(rows) != 1 || rows[0].Title != "Second" {
		t.Fatalf("page: rows=%+v total=%d err=%v", rows, total, err)
	}
}

func TestNullableColumn(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatal(err)
	}
	txt := "hello"
	p := &Post{Title: "t", Body: &txt, AuthorID: ada.ID}
	if err := db.Table[Post]().Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body == nil || *got.Body != "hello" {
		t.Fatalf("body not scanned: %+v", got.Body)
	}
}

func TestUpdateOptimisticLock(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatal(err)
	}
	p := createPost(t, db, ada, "orig")

	// Two copies of the same row: whoever updates second is stale.
	a, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	if err != nil {
		t.Fatal(err)
	}

	a.Title = "from A"
	if err := db.Table[Post]().Update(ctx, &a); err != nil {
		t.Fatalf("update A: %v", err)
	}
	if a.Version != 2 {
		t.Fatalf("version not bumped: %d", a.Version)
	}

	b.Title = "from B"
	err = db.Table[Post]().Update(ctx, &b)
	var he *core.HTTPError
	if !errors.As(err, &he) || he.Status != 409 {
		t.Fatalf("stale update should be Conflict(409), got %v", err)
	}

	// The winner's value persists; reloading shows version 2.
	got, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	if err != nil || got.Title != "from A" || got.Version != 2 {
		t.Fatalf("after conflict: %+v err=%v", got, err)
	}
}

func TestUpdateMissingRowIsNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatal(err)
	}
	p := createPost(t, db, ada, "x")
	if err := db.Table[Post]().Delete(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.Title = "ghost writes"
	err := db.Table[Post]().Update(ctx, p)
	var he *core.HTTPError
	if !errors.As(err, &he) || he.Status != 404 {
		t.Fatalf("update of a deleted row should be NotFound, got %v", err)
	}
}

func TestDeleteAndOneNotFound(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatal(err)
	}
	p := createPost(t, db, ada, "x")
	if err := db.Table[Post]().Delete(ctx, p); err != nil {
		t.Fatal(err)
	}
	_, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	var he *core.HTTPError
	if !errors.As(err, &he) || he.Status != 404 {
		t.Fatalf("one on missing row should be NotFound, got %v", err)
	}
}

func TestPredicateVocabulary(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		createPost(t, db, ada, "post")
	}

	cases := []struct {
		query *data.Query[Post]
		want  int
	}{
		{db.Table[Post]().Where(toc.ID.Lte(2)), 2},
		{db.Table[Post]().Where(toc.ID.Gt(4)), 0},
		{db.Table[Post]().Where(toc.ID.In(int64(1), int64(3))), 2},
		{db.Table[Post]().Where(toc.Title.Like("pos%")), 3},
		{db.Table[Post]().Where(toc.Body.Nil()), 3},
		{db.Table[Post]().Where(toc.Body.NotNil()), 0},
	}
	for _, c := range cases {
		rows, err := c.query.All(ctx)
		if err != nil || len(rows) != c.want {
			t.Errorf("query got %d rows err=%v", len(rows), err)
		}
	}

	// A predicate on a column that does not exist is a loud error.
	_, err := db.Table[Post]().Where(data.C[Post]("nope").Eq(1)).All(ctx)
	if err == nil {
		t.Fatal("unknown column should error")
	}
}

func TestRelationsEagerLoading(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ada := &Author{Name: "Ada"}
	bob := &Author{Name: "Bob"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatal(err)
	}
	if err := db.Table[Author]().Create(ctx, bob); err != nil {
		t.Fatal(err)
	}
	p1 := createPost(t, db, ada, "by ada")
	p2 := createPost(t, db, ada, "also ada")
	createPost(t, db, bob, "by bob")

	for _, c := range []*Comment{
		{PostID: p1.ID, Text: "nice"},
		{PostID: p1.ID, Text: "again"},
	} {
		if err := db.Table[Comment]().Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	posts, err := db.Table[Post]().
		Where(toc.AuthorID.Eq(ada.ID)).
		With(toc.Author, toc.Comments).
		OrderBy(toc.ID.Asc()).
		All(ctx)
	if err != nil {
		t.Fatalf("all with relations: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("want 2 posts, got %d", len(posts))
	}
	if posts[0].Author == nil || posts[0].Author.Name != "Ada" {
		t.Fatalf("belongs-to not loaded: %+v", posts[0].Author)
	}
	// No rows ever share a parent pointer in a batch (each has its own struct).
	if len(posts[0].Comments) != 2 {
		t.Fatalf("has-many not loaded: %+v", posts[0].Comments)
	}
	if len(posts[1].Comments) != 0 {
		t.Fatalf("empty has-many should be empty, got %d", len(posts[1].Comments))
	}
	if posts[1].ID != p2.ID {
		t.Fatalf("second post mismatch: %d != %d", posts[1].ID, p2.ID)
	}

	// Single row + has-many via One.
	one, err := db.Table[Post]().Where(toc.ID.Eq(p1.ID)).With(toc.Comments).One(ctx)
	if err != nil || len(one.Comments) != 2 || one.Comments[0].PostID != p1.ID {
		t.Fatalf("One with relation: %+v err=%v", one, err)
	}
}

func TestBadModelPanics(t *testing.T) {
	db, err := data.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer func() {
		if recover() == nil {
			t.Fatal("unbroken model should panic at Table")
		}
	}()
	db.Table[badModel]()
}

type badModel struct {
	data.Model       // no table tag
	ID         int64 `db:"id auto"`
}
