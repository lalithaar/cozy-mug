package data

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/lalithaar/cozy-mug/core"
)

// Table is the query/CUD entry point for a model: db.Table[Post](). It derives
// the model's schema from its declaration at first use and caches it. There is
// no per-model generated method (Go has no external receivers), so the table is
// reached via one generic call — the generated tokens keep the columns checked.
type Table[M any] struct {
	db   *DB
	desc *modelDesc
}

// Table binds model M to a DB handle. Descriptor errors (missing table tag, a
// model-typed field without db:"ref:...") panic here — they are static model
// mistakes and fail loud (0007; recreate at boot, so also exercised by any
// EnsureSchema call at startup).
func (db *DB) Table[M any]() *Table[M] {
	desc, err := descFor(reflect.TypeOf((*M)(nil)).Elem())
	if err != nil {
		panic(err)
	}
	return &Table[M]{db: db, desc: desc}
}

// Query is an immutable, partially-built select over a model. Every method
// returns a new Query, so a base query can be reused with different limits.
type Query[M any] struct {
	t      *Table[M]
	preds  []Predicate
	sorts  []Sort
	limit  int // -1 = no limit
	offset int
	withs  []Rel
}

func (t *Table[M]) Where(preds ...Predicate) *Query[M] {
	return (&Query[M]{t: t, limit: -1}).Where(preds...)
}

// OrderBy and Page/Count conveniences: the first step of a chain is a Table;
// the rest of the pipe is a Query.
func (t *Table[M]) OrderBy(sorts ...Sort) *Query[M] {
	return (&Query[M]{t: t, limit: -1}).OrderBy(sorts...)
}

// Count returns the number of rows in the table.
func (t *Table[M]) Count(ctx context.Context) (int64, error) {
	return (&Query[M]{t: t, limit: -1}).Count(ctx)
}

func (q *Query[M]) Where(preds ...Predicate) *Query[M] {
	c := q.clone()
	c.preds = append(append([]Predicate{}, q.preds...), preds...)
	return c
}

func (q *Query[M]) OrderBy(sorts ...Sort) *Query[M] {
	c := q.clone()
	c.sorts = append(append([]Sort{}, q.sorts...), sorts...)
	return c
}

func (q *Query[M]) Limit(n int) *Query[M]  { c := q.clone(); c.limit = n; return c }
func (q *Query[M]) Offset(n int) *Query[M] { c := q.clone(); c.offset = n; return c }

// With eagerly batch-loads the named relations (0016): one extra query per
// relation, no lazy N+1. Only top-level relations are loaded in this pass.
func (q *Query[M]) With(rels ...Rel) *Query[M] {
	c := q.clone()
	c.withs = append(append([]Rel{}, q.withs...), rels...)
	return c
}

func (q *Query[M]) clone() *Query[M] {
	return &Query[M]{t: q.t, preds: q.preds, sorts: q.sorts, limit: q.limit, offset: q.offset, withs: q.withs}
}

// All returns every matching row (empty, non-nil slice when none) with any
// requested relations loaded.
func (q *Query[M]) All(ctx context.Context) ([]M, error) {
	sql, args, err := q.build(false)
	if err != nil {
		return nil, err
	}
	rows, err := q.t.db.sql.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("data: query: %w", err)
	}

	out := make([]M, 0, 8)
	for rows.Next() {
		var m M
		if err := scanModel(q.t.desc, rows, reflect.ValueOf(&m).Elem()); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("data: rows: %w", err)
	}
	// Release the query connection before relation loading runs its own queries
	// (the pool is capped at 1 for SQLite).
	rows.Close()
	if err := loadRelations(ctx, q.t.db, q.t.desc, &out, q.withs...); err != nil {
		return nil, err
	}
	return out, nil
}

// One returns the first matching row, or core.NotFound when there are none so a
// loader can hand it straight to c.Error (errors 0007).
func (q *Query[M]) One(ctx context.Context) (M, error) {
	var zero M
	sql, args, err := q.build(true)
	if err != nil {
		return zero, err
	}
	rows, err := q.t.db.sql.QueryContext(ctx, sql, args...)
	if err != nil {
		return zero, fmt.Errorf("data: query: %w", err)
	}

	if !rows.Next() {
		rows.Close()
		if err := rows.Err(); err != nil {
			return zero, fmt.Errorf("data: rows: %w", err)
		}
		return zero, core.NotFound(nil)
	}
	var m M
	if err := scanModel(q.t.desc, rows, reflect.ValueOf(&m).Elem()); err != nil {
		rows.Close()
		return zero, err
	}
	// Release the query connection before relation loading (pool cap is 1).
	if err := rows.Close(); err != nil {
		return zero, err
	}
	one := []M{m}
	if err := loadRelations(ctx, q.t.db, q.t.desc, &one, q.withs...); err != nil {
		return zero, err
	}
	return one[0], nil
}

// Count returns the number of matching rows (no ORDER BY / paging).
func (q *Query[M]) Count(ctx context.Context) (int64, error) {
	sql, args, err := q.countSQL()
	if err != nil {
		return 0, err
	}
	var n int64
	if err := q.t.db.sql.QueryRowContext(ctx, sql, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("data: count: %w", err)
	}
	return n, nil
}

// Page returns one page of rows plus the total count of matching rows — the
// two numbers a listing page + pagination links need (0016 F8). per <= 0 is
// treated as no limit.
func (q *Query[M]) Page(ctx context.Context, page, per int) ([]M, int64, error) {
	if page < 1 {
		page = 1
	}
	if per > 0 {
		p := q.clone()
		p.offset = (page - 1) * per
		p.limit = per
		items, err := p.All(ctx)
		if err != nil {
			return nil, 0, err
		}
		total, err := q.Count(ctx)
		if err != nil {
			return nil, 0, err
		}
		return items, total, nil
	}
	items, err := q.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	total, err := q.Count(ctx)
	return items, total, err
}

// sqlw accumulates a SQL statement and its bound args, emitting the dialect's
// placeholder for each value (SQLite ?, Postgres $n). The builder composes this
// SQL; raw user SQL (the escape hatch) is never touched.
type sqlw struct {
	b    strings.Builder
	args []any
	ph   func(int) string
}

func newSQLW(dialect string) *sqlw {
	if dialect == "postgres" {
		return &sqlw{ph: func(n int) string { return fmt.Sprintf("$%d", n) }}
	}
	return &sqlw{ph: func(int) string { return "?" }}
}

// arg records a bound value and writes its placeholder.
func (w *sqlw) arg(v any) {
	w.args = append(w.args, v)
	w.b.WriteString(w.ph(len(w.args)))
}

func (w *sqlw) write(s string) { w.b.WriteString(s) }

func colList(desc *modelDesc) string {
	names := make([]string, len(desc.cols))
	for i, c := range desc.cols {
		names[i] = c.name
	}
	return strings.Join(names, ", ")
}

// build compiles the select. one adds LIMIT 1.
func (q *Query[M]) build(one bool) (string, []any, error) {
	desc := q.t.desc
	w := newSQLW(q.t.db.Dialect())
	w.write("SELECT ")
	w.write(colList(desc))
	w.write(" FROM ")
	w.write(desc.table)

	if len(q.preds) > 0 {
		w.write(" WHERE ")
		if err := writePredicates(w, desc, q.preds); err != nil {
			return "", nil, err
		}
	}
	if len(q.sorts) > 0 {
		w.write(" ORDER BY ")
		for i, s := range q.sorts {
			if i > 0 {
				w.write(", ")
			}
			col, ok := desc.coln[s.col.name]
			if !ok {
				return "", nil, fmt.Errorf("data: %s: OrderBy on unknown column %q", desc.table, s.col.name)
			}
			w.write(col.name)
			w.write(" ")
			w.write(s.dir)
		}
	}
	if one {
		w.write(" LIMIT 1")
	} else if q.limit >= 0 {
		w.write(" LIMIT ")
		w.arg(int64(q.limit))
	}
	if q.offset > 0 {
		w.write(" OFFSET ")
		w.arg(int64(q.offset))
	}
	return w.b.String(), w.args, nil
}

// countSQL compiles the COUNT(*) for the same predicates (no ordering/paging).
func (q *Query[M]) countSQL() (string, []any, error) {
	desc := q.t.desc
	w := newSQLW(q.t.db.Dialect())
	w.write("SELECT COUNT(*) FROM ")
	w.write(desc.table)
	if len(q.preds) > 0 {
		w.write(" WHERE ")
		if err := writePredicates(w, desc, q.preds); err != nil {
			return "", nil, err
		}
	}
	return w.b.String(), w.args, nil
}

func writePredicates(w *sqlw, desc *modelDesc, preds []Predicate) error {
	for i, p := range preds {
		if i > 0 {
			w.write(" AND ")
		}
		col, ok := desc.coln[p.col.name]
		if !ok {
			return fmt.Errorf("data: %s: predicate on unknown column %q", desc.table, p.col.name)
		}
		w.write(col.name)
		w.write(" ")
		switch p.op {
		case "IS NULL", "IS NOT NULL":
			w.write(p.op)
		case "IN", "NOT IN":
			w.write(p.op)
			w.write(" (")
			for j, v := range p.values {
				if j > 0 {
					w.write(", ")
				}
				w.arg(v)
			}
			w.write(")")
		default:
			w.write(p.op)
			w.write(" ")
			w.arg(p.values[0])
		}
	}
	return nil
}

// mustTable panics only on programmer error; used by EnsureSchema/migrate paths
// that have no other error channel.
func mustDesc(t reflect.Type) *modelDesc {
	d, err := descFor(t)
	if err != nil {
		panic(err)
	}
	return d
}
