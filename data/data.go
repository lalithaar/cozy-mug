// Package data is cozy-mug's database layer: a thin, dialect-aware wrapper over
// database/sql. It carries the drivers (frugality 0001: the DB driver is the one
// sanctioned dependency) so an app never picks one, and it stays out of the way
// of the SQL you write — raw pass-through, no query rewriting (0012).
package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lalithaar/cozy-mug/core"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
	_ "modernc.org/sqlite"             // registers the "sqlite" driver (pure Go)
)

// dialect is inferred from the DSN, never configured (0009 spirit: no knobs the
// app must remember). It selects the driver and, only where it matters, defaults.
type dialect uint8

const (
	dialectSQLite dialect = iota
	dialectPostgres
)

func (d dialect) String() string {
	if d == dialectPostgres {
		return "postgres"
	}
	return "sqlite"
}

// sqlitePragmas run once on the (single) SQLite connection. With MaxOpenConns(1)
// below they reliably apply to every statement: foreign keys are enforced and a
// busy write waits instead of failing instantly.
var sqlitePragmas = []string{
	"PRAGMA foreign_keys = ON",
	"PRAGMA busy_timeout = 5000",
}

// DB is the handle an app opens once at boot and passes to its loaders. It is a
// deliberate wrapper, not an embedded *sql.DB: the only query entry points are
// the package helpers (All/One/Exec), so there is no raw db.Exec to reach for by
// mistake.
type DB struct {
	sql     *sql.DB
	dialect dialect
}

// Dialect reports the detected backend ("sqlite" or "postgres").
func (db *DB) Dialect() string { return db.dialect.String() }

// Close releases the connection pool. Call it once at shutdown.
func (db *DB) Close() error { return db.sql.Close() }

// QueryContext, QueryRowContext, and ExecContext implement Conn on *DB. They
// exist so *DB satisfies Conn; prefer the All/One/Exec helpers in app code.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.sql.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.sql.QueryRowContext(ctx, query, args...)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.sql.ExecContext(ctx, query, args...)
}

// Open connects to the database named by dsn and returns a ready handle. The
// dialect is inferred: a postgres:// or postgresql:// URL is Postgres; anything
// else is a SQLite path or DSN (":memory:", "file:...", "app.db").
//
// The dsn is treated as a secret: it is never logged and never echoed into an
// error (secrets heuristic).
func Open(ctx context.Context, dsn string) (*DB, error) {
	d, driver := detectDialect(dsn)

	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("data: open: %w", safeErr(err, dsn))
	}

	configurePool(sqlDB, d)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("data: ping: %w", safeErr(err, dsn))
	}

	if d == dialectSQLite {
		for _, p := range sqlitePragmas {
			if _, err := sqlDB.ExecContext(ctx, p); err != nil {
				_ = sqlDB.Close()
				return nil, fmt.Errorf("data: pragma: %w", safeErr(err, dsn))
			}
		}
	}

	return &DB{sql: sqlDB, dialect: d}, nil
}

// detectDialect infers the backend and the database/sql driver name from the DSN.
func detectDialect(dsn string) (dialect, string) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return dialectPostgres, "pgx"
	}
	return dialectSQLite, "sqlite"
}

// configurePool sets sane resource limits so a small app cannot exhaust
// connections or hold a Postgres backend open forever. SQLite runs a single
// connection: it serializes writers anyway, and it is what makes a :memory:
// database (and the per-connection pragmas above) behave predictably.
func configurePool(db *sql.DB, d dialect) {
	if d == dialectSQLite {
		db.SetMaxOpenConns(1)
		db.SetConnMaxLifetime(0)
		return
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
}

// Conn is the subset of the database handle the query helpers need. Both *DB and
// *Tx satisfy it, so the same query works inside or outside a transaction.
//
// Queries are plain SQL, passed through unchanged. cozy-mug does not rewrite
// placeholders: use ? for SQLite and $1 for Postgres, because the two dialects
// differ and are not translated (0001 frugality: no query builder). All arguments
// are sent as bound parameters by the driver — never interpolated into the SQL.
type Conn interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// All runs query and scans every row into a T. It returns an empty (non-nil)
// slice when there are no rows.
func All[T any](ctx context.Context, c Conn, query string, args ...any) ([]T, error) {
	rows, err := c.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("data: query: %w", err)
	}
	defer rows.Close()

	out := []T{}
	for rows.Next() {
		var v T
		if err := scanInto(rows, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("data: rows: %w", err)
	}
	return out, nil
}

// One runs query and scans the first row into a T. When the query returns no
// rows it returns a core 404 error, so a loader can hand the result straight to
// c.Error and the app gets the standard not-found page (errors 0007).
func One[T any](ctx context.Context, c Conn, query string, args ...any) (T, error) {
	var zero T
	rows, err := c.QueryContext(ctx, query, args...)
	if err != nil {
		return zero, fmt.Errorf("data: query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zero, fmt.Errorf("data: rows: %w", err)
		}
		return zero, core.NotFound(nil)
	}

	var v T
	if err := scanInto(rows, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// Exec runs a statement that returns no rows (INSERT/UPDATE/DELETE). Use the
// returned result for LastInsertId / RowsAffected.
func Exec(ctx context.Context, c Conn, query string, args ...any) (sql.Result, error) {
	res, err := c.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("data: exec: %w", err)
	}
	return res, nil
}

// Tx runs fn inside a transaction, committing if fn returns nil and rolling back
// on any returned error. A panic inside fn also rolls back and then re-panics,
// so the Errors middleware still sees it and the connection is never left in an
// open transaction.
func (db *DB) Tx(ctx context.Context, fn func(*Tx) error) error {
	sqlTx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("data: begin: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = sqlTx.Rollback()
			panic(p)
		}
	}()

	if err := fn(&Tx{Tx: sqlTx}); err != nil {
		_ = sqlTx.Rollback()
		return err
	}

	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("data: commit: %w", err)
	}
	return nil
}

// Tx is a transaction handle. It satisfies Conn, so All/One/Exec accept it
// directly and the same query code runs inside a transaction unchanged.
type Tx struct {
	*sql.Tx
}

// safeErr returns an error whose message never contains the DSN or its password.
// Drivers sometimes include the connection string; we strip it before it can
// reach a log (secrets heuristic).
func safeErr(err error, dsn string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if dsn != "" {
		msg = strings.ReplaceAll(msg, dsn, "<dsn>")
	}
	if u, e := url.Parse(dsn); e == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok && pw != "" {
			msg = strings.ReplaceAll(msg, pw, "<redacted>")
		}
	}
	return errors.New(msg)
}
