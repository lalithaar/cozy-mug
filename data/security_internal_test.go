package data

import (
	"errors"
	"strings"
	"testing"
)

func TestSafeErrRedactsDSNAndPassword(t *testing.T) {
	dsn := "postgresql://user:supersecret@127.0.0.1:59999/db?sslmode=disable"
	err := safeErr(errors.New("connection failed involving "+dsn+" and supersecret"), dsn)

	if strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("password leaked in error: %s", err.Error())
	}
	if strings.Contains(err.Error(), dsn) {
		t.Fatalf("DSN leaked in error: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "connection failed") {
		t.Fatalf("underlying message lost: %s", err.Error())
	}
}

func TestDetectDialect(t *testing.T) {
	cases := []struct {
		dsn   string
		want  dialect
		driver string
	}{
		{"postgres://u:p@localhost/db", dialectPostgres, "pgx"},
		{"postgresql://u:p@localhost/db", dialectPostgres, "pgx"},
		{"app.db", dialectSQLite, "sqlite"},
		{":memory:", dialectSQLite, "sqlite"},
		{"file:app.db?cache=shared", dialectSQLite, "sqlite"},
	}
	for _, tc := range cases {
		gotD, gotDriver := detectDialect(tc.dsn)
		if gotD != tc.want || gotDriver != tc.driver {
			t.Fatalf("detectDialect(%q) = (%v,%s), want (%v,%s)", tc.dsn, gotD, gotDriver, tc.want, tc.driver)
		}
	}
}