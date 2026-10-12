package data_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/data"
)

type note struct {
	ID    int64  `db:"id"`
	Title string `db:"title"`
	Body  string `db:"body"`
}

func newDB(t *testing.T) *data.DB {
	t.Helper()
	db, err := data.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := data.Exec(context.Background(), db, `create table notes (
		id    integer primary key autoincrement,
		title text not null,
		body  text
	)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

func TestOpenDetectsSQLite(t *testing.T) {
	db := newDB(t)
	if got := db.Dialect(); got != "sqlite" {
		t.Fatalf("Dialect() = %q, want sqlite", got)
	}
}

func TestCRUD(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)

	res, err := data.Exec(ctx, db, `insert into notes (title, body) values (?, ?)`, "first", "hello")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if id, _ := res.LastInsertId(); id != 1 {
		t.Fatalf("LastInsertId = %d, want 1", id)
	}
	if _, err := data.Exec(ctx, db, `insert into notes (title, body) values (?, ?)`, "second", "world"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	all, err := data.All[note](ctx, db, `select id, title, body from notes order by id`)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(all) != 2 || all[0].Title != "first" || all[1].Body != "world" {
		t.Fatalf("All = %+v", all)
	}

	one, err := data.One[note](ctx, db, `select id, title, body from notes where id = ?`, 2)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	if one.Title != "second" {
		t.Fatalf("One = %+v", one)
	}

	if _, err := data.Exec(ctx, db, `update notes set title = ? where id = ?`, "renamed", 1); err != nil {
		t.Fatalf("update: %v", err)
	}
	one, _ = data.One[note](ctx, db, `select id, title, body from notes where id = ?`, 1)
	if one.Title != "renamed" {
		t.Fatalf("after update = %+v", one)
	}
}

func TestOneNoRowsIsNotFound(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)

	_, err := data.One[note](ctx, db, `select id, title, body from notes where id = ?`, 999)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var he *core.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("error %v is not *core.HTTPError", err)
	}
	if he.Status != 404 {
		t.Fatalf("status = %d, want 404", he.Status)
	}
}

func TestAllEmptyIsNonNil(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)

	all, err := data.All[note](ctx, db, `select id, title, body from notes`)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if all == nil || len(all) != 0 {
		t.Fatalf("All = %#v, want empty non-nil slice", all)
	}
}

func TestTxCommits(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)

	err := db.Tx(ctx, func(tx *data.Tx) error {
		if _, err := data.Exec(ctx, tx, `insert into notes (title) values (?)`, "a"); err != nil {
			return err
		}
		_, err := data.Exec(ctx, tx, `insert into notes (title) values (?)`, "b")
		return err
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}

	n, err := data.One[int](ctx, db, `select count(*) from notes`)
	if err != nil || n != 2 {
		t.Fatalf("count = %d, err = %v; want 2", n, err)
	}
}

func TestTxRollsBackOnError(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)

	sentinel := errors.New("nope")
	err := db.Tx(ctx, func(tx *data.Tx) error {
		if _, err := data.Exec(ctx, tx, `insert into notes (title) values (?)`, "gone"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("tx err = %v, want sentinel", err)
	}

	n, _ := data.One[int](ctx, db, `select count(*) from notes`)
	if n != 0 {
		t.Fatalf("count = %d after rollback, want 0", n)
	}
}

func TestTxRollsBackOnPanic(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic to propagate")
			}
		}()
		_ = db.Tx(ctx, func(tx *data.Tx) error {
			if _, err := data.Exec(ctx, tx, `insert into notes (title) values (?)`, "boom"); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	n, _ := data.One[int](ctx, db, `select count(*) from notes`)
	if n != 0 {
		t.Fatalf("count = %d after panic rollback, want 0", n)
	}
}

type EmbeddedBase struct {
	ID int64 `db:"id"`
}

type embeddedNote struct {
	EmbeddedBase
	Title string `db:"title"`
}

func TestScanEmbeddedAndTagMapping(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	if _, err := data.Exec(ctx, db, `insert into notes (title) values (?)`, "hi"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := data.One[embeddedNote](ctx, db, `select id, title from notes`)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	if got.ID != 1 || got.Title != "hi" {
		t.Fatalf("got %+v", got)
	}

	type renamed struct {
		Heading string `db:"title"`
	}
	r, err := data.One[renamed](ctx, db, `select title from notes`)
	if err != nil || r.Heading != "hi" {
		t.Fatalf("renamed = %+v, err = %v", r, err)
	}
}

func TestScanUnexportedAndUnknownColumns(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	if _, err := data.Exec(ctx, db, `insert into notes (title) values (?)`, "hi"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	type target struct {
		ID    int64 `db:"id"`
		Title string
		priv  string // unexported: must never be touched by the scanner
		Other string // no matching column: stays zero
	}
	got, err := data.One[target](ctx, db, `select id, title, 1 as ignored_extra from notes`)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	if got.priv != "" || got.Other != "" {
		t.Fatalf("unexpected field values: %+v", got)
	}
	if got.Title != "hi" {
		t.Fatalf("Title = %q, want hi", got.Title)
	}
}

func TestScanNullIntoPointer(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	if _, err := data.Exec(ctx, db, `insert into notes (title, body) values (?, ?)`, "t", nil); err != nil {
		t.Fatalf("insert: %v", err)
	}

	type nullable struct {
		Body *string `db:"body"`
	}
	got, err := data.One[nullable](ctx, db, `select body from notes`)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	if got.Body != nil {
		t.Fatalf("Body = %v, want nil", *got.Body)
	}
}

func TestScanNullIntoStringErrors(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	if _, err := data.Exec(ctx, db, `insert into notes (title, body) values (?, ?)`, "t", nil); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// note.Body is a plain string, so a NULL body is a genuine scan error —
	// returned, not a panic.
	if _, err := data.One[note](ctx, db, `select id, title, body from notes`); err == nil {
		t.Fatal("expected scan error for NULL into string, got nil")
	}
}
