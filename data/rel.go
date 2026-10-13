package data

import (
	"context"
	"fmt"
	"reflect"
)

// loadRelations eagerly batch-loads the requested relations for a set of rows
// (0016): one extra query per relation, never a query per row. belongs-to
// relations are loaded by collecting the rows' foreign keys and fetching all
// parents in one IN; has-many relations fetch all children in one IN and group
// them by foreign key. rows must be a pointer to a []M slice so relation
// assignments write back into the caller's slice.
func loadRelations(ctx context.Context, db *DB, src *modelDesc, rows any, rels ...Rel) error {
	v := reflect.ValueOf(rows)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("data: relation load: pass a pointer to the row slice, got %s", v.Kind())
	}
	sv := v.Elem()
	if sv.Kind() != reflect.Slice {
		return fmt.Errorf("data: relation load: pass a row slice, got %s", sv.Kind())
	}
	if len(rels) == 0 || sv.Len() == 0 {
		return nil
	}

	for _, tok := range rels {
		if tok.model != src.typ {
			return fmt.Errorf("data: %s: With() relation token was built for %s, not %s", src.table, tok.model, src.typ)
		}
		rel, ok := src.rels[tok.field]
		if !ok {
			return fmt.Errorf("data: %s: unknown relation %q (declared fields: %v)", src.table, tok.field, relationNames(src))
		}
		if rel.isMany {
			if err := loadHasMany(ctx, db, src, sv, rel); err != nil {
				return err
			}
		} else {
			if err := loadBelongsTo(ctx, db, src, sv, rel); err != nil {
				return err
			}
		}
	}
	return nil
}

func relationNames(d *modelDesc) []string {
	out := make([]string, 0, len(d.rels))
	for name := range d.rels {
		out = append(out, name)
	}
	return out
}

// loadBelongsTo fills a *Child relation by collecting this model's foreign keys
// and fetching every parent in one query.
func loadBelongsTo(ctx context.Context, db *DB, src *modelDesc, rows reflect.Value, rel *relation) error {
	fkCol, ok := src.coln[rel.fk]
	if !ok {
		return fmt.Errorf("data: %s.%s: ref column %q is not a column of %s", src.table, rel.field, rel.fk, src.table)
	}
	child, err := descFor(rel.typ)
	if err != nil {
		return err
	}
	if child.pk == nil {
		return fmt.Errorf("data: %s.%s: child %s has no primary key for relation lookup", src.table, rel.field, child.table)
	}

	// Distinct foreign keys across the rows.
	need := map[any]struct{}{}
	for i := 0; i < rows.Len(); i++ {
		fv := rows.Index(i).FieldByIndex(fkCol.idx)
		if isZero(fv) {
			continue
		}
		need[keyOf(fv.Interface())] = struct{}{}
	}
	if len(need) == 0 {
		return nil
	}
	keys := make([]any, 0, len(need))
	for k := range need {
		keys = append(keys, k)
	}

	kids, err := queryWhereIN(ctx, db, child, child.coln[child.pk.name], keys)
	if err != nil {
		return err
	}
	byPk := map[any]reflect.Value{}
	for _, kid := range kids {
		fv := kid.FieldByIndex(child.pk.idx)
		byPk[fv.Interface()] = kid
	}

	for i := 0; i < rows.Len(); i++ {
		row := rows.Index(i)
		fv := row.FieldByIndex(fkCol.idx)
		if isZero(fv) {
			continue
		}
		kid, ok := byPk[fv.Interface()]
		if !ok {
			continue // parent missing: leave nil
		}
		row.FieldByName(rel.field).Set(kid.Addr())
	}
	return nil
}

// loadHasMany fills a []*Child relation by fetching every child whose foreign
// key is one of this model's primary keys.
func loadHasMany(ctx context.Context, db *DB, src *modelDesc, rows reflect.Value, rel *relation) error {
	if src.pk == nil {
		return fmt.Errorf("data: %s.%s: %s has no primary key for the has-many lookup", src.table, rel.field, src.table)
	}
	child, err := descFor(rel.typ)
	if err != nil {
		return err
	}
	fkCol, ok := child.coln[rel.fk]
	if !ok {
		return fmt.Errorf("data: %s.%s: ref column %q is not a column of child %s", src.table, rel.field, rel.fk, child.table)
	}

	need := map[any]struct{}{}
	for i := 0; i < rows.Len(); i++ {
		pv := rows.Index(i).FieldByIndex(src.pk.idx)
		if isZero(pv) {
			continue
		}
		need[pv.Interface()] = struct{}{}
	}
	if len(need) == 0 {
		return nil
	}
	keys := make([]any, 0, len(need))
	for k := range need {
		keys = append(keys, k)
	}

	kids, err := queryWhereIN(ctx, db, child, fkCol, keys)
	if err != nil {
		return err
	}
	byFk := map[any][]reflect.Value{}
	for _, kid := range kids {
		fv := kid.FieldByIndex(fkCol.idx)
		byFk[fv.Interface()] = append(byFk[fv.Interface()], kid)
	}

	sliceT := reflect.SliceOf(reflect.PtrTo(child.typ))
	for i := 0; i < rows.Len(); i++ {
		row := rows.Index(i)
		pv := row.FieldByIndex(src.pk.idx)
		if isZero(pv) {
			continue
		}
		list := byFk[pv.Interface()]
		out := reflect.MakeSlice(sliceT, 0, len(list))
		for _, kid := range list {
			out = reflect.Append(out, kid.Addr())
		}
		row.FieldByName(rel.field).Set(out)
	}
	return nil
}

// queryWhereIN runs SELECT <all cols> FROM <table> WHERE <col> IN (...) and
// scans every row into an addressable model value.
func queryWhereIN(ctx context.Context, db *DB, d *modelDesc, col *column, values []any) ([]reflect.Value, error) {
	w := newSQLW(db.Dialect())
	w.write("SELECT ")
	w.write(colList(d))
	w.write(" FROM ")
	w.write(d.table)
	w.write(" WHERE ")
	w.write(col.name)
	w.write(" IN (")
	for i, v := range values {
		if i > 0 {
			w.write(", ")
		}
		w.arg(v)
	}
	w.write(")")

	rows, err := db.sql.QueryContext(ctx, w.b.String(), w.args...)
	if err != nil {
		return nil, fmt.Errorf("data: relation load: %w", err)
	}
	defer rows.Close()

	var out []reflect.Value
	for rows.Next() {
		mv := reflect.New(d.typ).Elem()
		if err := scanModel(d, rows, mv); err != nil {
			return nil, err
		}
		out = append(out, mv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("data: relation rows: %w", err)
	}
	return out, nil
}

// keyOf normalizes a value into a comparable map key (int/uint/floats/strings;
// anything else is kept as itself).
func keyOf(v any) any {
	switch k := v.(type) {
	case int:
		return int64(k)
	case int8:
		return int64(k)
	case int16:
		return int64(k)
	case int32:
		return int64(k)
	case uint:
		return int64(k)
	case uint8:
		return int64(k)
	case uint16:
		return int64(k)
	case uint32:
		return int64(k)
	case uint64:
		return int64(k)
	case float32:
		return float64(k)
	default:
		return v
	}
}
