package data

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/lalithaar/cozy-mug/core"
)

// Create inserts the whole row (all non-auto columns), then back-fills the auto
// primary key (LastInsertId on SQLite, RETURNING on Postgres) and the
// maintained columns (created_at, updated_at, version) onto the struct so the
// object reflects the database.
func (t *Table[M]) Create(ctx context.Context, m *M) error {
	v := reflect.ValueOf(m).Elem()
	d := t.desc

	now := time.Now().UTC()
	if d.createdAt != nil && v.FieldByIndex(d.createdAt.idx).Type() == reflect.TypeOf(time.Time{}) {
		v.FieldByIndex(d.createdAt.idx).Set(reflect.ValueOf(now))
	}
	if d.updatedAt != nil && v.FieldByIndex(d.updatedAt.idx).Type() == reflect.TypeOf(time.Time{}) {
		v.FieldByIndex(d.updatedAt.idx).Set(reflect.ValueOf(now))
	}
	if d.version != nil && intKind(d.version.goType) {
		v.FieldByIndex(d.version.idx).SetInt(1)
	}

	w := newSQLW(t.db.Dialect())
	w.write("INSERT INTO ")
	w.write(d.table)
	w.write(" (")
	cols := make([]*column, 0, len(d.cols))
	for _, c := range d.cols {
		if c.auto {
			continue
		}
		cols = append(cols, c)
	}
	for i, c := range cols {
		if i > 0 {
			w.write(", ")
		}
		w.write(c.name)
	}
	w.write(") VALUES (")
	for i := range cols {
		if i > 0 {
			w.write(", ")
		}
		w.arg(d.columnValue(v, cols[i]).Interface())
	}
	w.write(")")

	if t.db.Dialect() == "postgres" && d.auto != nil {
		// Postgres has no LastInsertId; fetch the generated pk back.
		w.write(" RETURNING ")
		w.write(d.auto.name)
		row := t.db.sql.QueryRowContext(ctx, w.b.String(), w.args...)
		if err := row.Scan(v.FieldByIndex(d.auto.idx).Addr().Interface()); err != nil {
			return fmt.Errorf("data: create: %w", err)
		}
		return nil
	}

	res, err := t.db.sql.ExecContext(ctx, w.b.String(), w.args...)
	if err != nil {
		return fmt.Errorf("data: create: %w", err)
	}
	if d.auto != nil {
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("data: create: last insert id: %w", err)
		}
		setInt(v.FieldByIndex(d.auto.idx), id)
	}
	return nil
}

// Update writes the whole row (all non-auto columns) under the primary key,
// guarded by the optimistic-lock version column when the model has one: the
// UPDATE carries WHERE pk = ? AND version = <stored>, so a concurrent write
// that already bumped the version makes RowsAffected 0 and Update returns
// core.Conflict — the handler re-renders "edited elsewhere, reload" (0016 F1).
func (t *Table[M]) Update(ctx context.Context, m *M) error {
	v := reflect.ValueOf(m).Elem()
	d := t.desc

	if d.pk == nil {
		return fmt.Errorf("data: update: %s has no primary key column", d.table)
	}
	pkVal := d.columnValue(v, d.pk)
	if isZero(pkVal) {
		return fmt.Errorf("data: update: %s: primary key is zero — load or create the row first", d.table)
	}

	var whereVersion int64
	if d.version != nil {
		cur := v.FieldByIndex(d.version.idx).Int()
		if cur < 1 {
			return fmt.Errorf("data: update: %s: version is %d — this is a fresh or unloaded row", d.table, cur)
		}
		whereVersion = cur
		v.FieldByIndex(d.version.idx).SetInt(cur + 1)
	}
	if d.updatedAt != nil && v.FieldByIndex(d.updatedAt.idx).Type() == reflect.TypeOf(time.Time{}) {
		v.FieldByIndex(d.updatedAt.idx).Set(reflect.ValueOf(time.Now().UTC()))
	}

	w := newSQLW(t.db.Dialect())
	w.write("UPDATE ")
	w.write(d.table)
	w.write(" SET ")
	setCols := make([]*column, 0, len(d.cols))
	for _, c := range d.cols {
		if c.auto || c == d.pk {
			continue
		}
		setCols = append(setCols, c)
	}
	for i, c := range setCols {
		if i > 0 {
			w.write(", ")
		}
		w.write(c.name)
		w.write(" = ")
		w.arg(d.columnValue(v, c).Interface())
	}
	w.write(" WHERE ")
	w.write(d.pk.name)
	w.write(" = ")
	w.arg(pkVal.Interface())
	if d.version != nil {
		w.write(" AND ")
		w.write(d.version.name)
		w.write(" = ")
		w.arg(whereVersion)
	}

	res, err := t.db.sql.ExecContext(ctx, w.b.String(), w.args...)
	if err != nil {
		return fmt.Errorf("data: update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("data: update: rows affected: %w", err)
	}
	if n == 0 {
		exists, err := t.exists(ctx, pkVal.Interface())
		if err != nil {
			return err
		}
		if !exists {
			return core.NotFound(nil)
		}
		return core.Conflict(nil)
	}
	return nil
}

// Delete removes the row by primary key. A missing row is core.NotFound.
func (t *Table[M]) Delete(ctx context.Context, m *M) error {
	v := reflect.ValueOf(m).Elem()
	d := t.desc

	if d.pk == nil {
		return fmt.Errorf("data: delete: %s has no primary key column", d.table)
	}
	pkVal := d.columnValue(v, d.pk)
	if isZero(pkVal) {
		return fmt.Errorf("data: delete: %s: primary key is zero", d.table)
	}

	w := newSQLW(t.db.Dialect())
	w.write("DELETE FROM ")
	w.write(d.table)
	w.write(" WHERE ")
	w.write(d.pk.name)
	w.write(" = ")
	w.arg(pkVal.Interface())

	res, err := t.db.sql.ExecContext(ctx, w.b.String(), w.args...)
	if err != nil {
		return fmt.Errorf("data: delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("data: delete: rows affected: %w", err)
	}
	if n == 0 {
		return core.NotFound(nil)
	}
	return nil
}

// exists reports whether a row with the given pk value exists.
func (t *Table[M]) exists(ctx context.Context, pk any) (bool, error) {
	w := newSQLW(t.db.Dialect())
	w.write("SELECT 1 FROM ")
	w.write(t.desc.table)
	w.write(" WHERE ")
	w.write(t.desc.pk.name)
	w.write(" = ")
	w.arg(pk)

	var one int
	err := t.db.sql.QueryRowContext(ctx, w.b.String(), w.args...).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("data: exists: %w", err)
	}
	return true, nil
}

// EnsureSchema creates the table (and its index/unique columns) if missing.
// Interim: 0016's generated, versioned migrations replace this on the next
// slice; apps keep calling store logic at boot either way.
func (t *Table[M]) EnsureSchema(ctx context.Context) error {
	create, indexes, err := t.desc.DDL(t.db.Dialect())
	if err != nil {
		return err
	}
	if _, err := t.db.sql.ExecContext(ctx, create); err != nil {
		return fmt.Errorf("data: ensure schema: %w", err)
	}
	for _, idx := range indexes {
		if _, err := t.db.sql.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("data: ensure schema: %w", err)
		}
	}
	return nil
}

func intKind(t reflect.Type) bool {
	k := t.Kind()
	return k >= reflect.Int && k <= reflect.Int64
}

func setInt(f reflect.Value, n int64) {
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f.SetUint(uint64(n))
	}
}

// isZero reports whether f holds the zero value (for nullable pointers, a nil).
func isZero(f reflect.Value) bool {
	if f.Kind() == reflect.Ptr {
		return f.IsNil()
	}
	return f.IsZero()
}
