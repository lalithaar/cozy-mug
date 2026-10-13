package data_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
)

// TestPostgresDataLayer is an env-gated integration test: it exercises the
// model-first data layer against a real Postgres server. Set DATABASE_URL (or
// POSTGRES_DSN) to a postgres://... DSN to run it; otherwise it skips. CI runs
// it against a service container (see .github/workflows/ci.yml).
func TestPostgresDataLayer(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		t.Skip("set DATABASE_URL or POSTGRES_DSN to run the live Postgres data test")
	}

	db, err := data.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	for _, stmt := range []string{
		"DROP TABLE IF EXISTS comments CASCADE",
		"DROP TABLE IF EXISTS posts CASCADE",
		"DROP TABLE IF EXISTS authors CASCADE",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("drop: %v", err)
		}
	}

	for _, model := range []any{Author{}, Post{}, Comment{}} {
		_ = model
	}
	for _, fn := range []func() error{
		func() error {
			stmts, err := db.Table[Author]().SchemaSQL()
			if err != nil {
				return err
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return err
				}
			}
			return nil
		},
		func() error {
			stmts, err := db.Table[Post]().SchemaSQL()
			if err != nil {
				return err
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return err
				}
			}
			return nil
		},
		func() error {
			stmts, err := db.Table[Comment]().SchemaSQL()
			if err != nil {
				return err
			}
			for _, s := range stmts {
				if _, err := db.ExecContext(ctx, s); err != nil {
					return err
				}
			}
			return nil
		},
	} {
		if err := fn(); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	// Create + back-fill + timestamps
	ada := &Author{Name: "Ada"}
	if err := db.Table[Author]().Create(ctx, ada); err != nil {
		t.Fatalf("create author: %v", err)
	}
	p := &Post{Title: "First", Body: nil, AuthorID: ada.ID}
	if err := db.Table[Post]().Create(ctx, p); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if p.ID == 0 || p.Version != 1 || p.CreatedAt.IsZero() {
		t.Fatalf("create back-fill wrong: %+v", p)
	}

	got, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	if err != nil || got.Title != "First" {
		t.Fatalf("one: %+v %v", got, err)
	}

	// Nullable column
	txt := "hello"
	pp := &Post{Title: "t", Body: &txt, AuthorID: ada.ID}
	if err := db.Table[Post]().Create(ctx, pp); err != nil {
		t.Fatal(err)
	}
	loaded, err := db.Table[Post]().Where(toc.ID.Eq(pp.ID)).One(ctx)
	if err != nil || loaded.Body == nil || *loaded.Body != "hello" {
		t.Fatalf("nullable: %+v %v", loaded.Body, err)
	}

	// Predicate vocabulary + count
	db.Table[Post]().Create(ctx, &Post{Title: "post", Body: nil, AuthorID: ada.ID})
	n, err := db.Table[Post]().Count(ctx)
	if err != nil || n < 3 {
		t.Fatalf("count = %d, %v", n, err)
	}

	// Optimistic lock conflict
	a, _ := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	b, _ := db.Table[Post]().Where(toc.ID.Eq(p.ID)).One(ctx)
	a.Title = "from A"
	if err := db.Table[Post]().Update(ctx, &a); err != nil {
		t.Fatalf("update A: %v", err)
	}
	b.Title = "from B"
	err = db.Table[Post]().Update(ctx, &b)
	var he *core.HTTPError
	if !errors.As(err, &he) || he.Status != 409 {
		t.Fatalf("conflict: want 409, got %v", err)
	}

	// Relation eager loading
	cmt := &Comment{PostID: p.ID, Text: "nice"}
	if err := db.Table[Comment]().Create(ctx, cmt); err != nil {
		t.Fatal(err)
	}
	posts, err := db.Table[Post]().Where(toc.ID.Eq(p.ID)).With(toc.Comments).One(ctx)
	if err != nil || len(posts.Comments) != 1 {
		t.Fatalf("relation: %+v %v", posts.Comments, err)
	}

	// Delete + not found
	if err := db.Table[Post]().Delete(ctx, &a); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = db.Table[Post]().Where(toc.ID.Eq(a.ID)).One(ctx)
	if !errors.As(err, &he) || he.Status != 404 {
		t.Fatalf("not found: %v", err)
	}
}
