package data_test

import (
	"context"
	"testing"

	"github.com/lalithaar/cozy-mug/data"
)

func TestOpenEnforcesForeignKeys(t *testing.T) {
	db, err := data.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`create table parents (id integer primary key)`,
		`create table children (id integer primary key, parent_id integer references parents(id))`,
	} {
		if _, err := data.Exec(context.Background(), db, stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	if _, err := data.Exec(context.Background(), db, `insert into children (parent_id) values (?)`, 999); err == nil {
		t.Fatal("foreign key violation was allowed")
	}
}