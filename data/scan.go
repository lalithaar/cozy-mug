package data

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// scanInto scans the current row of rows into *T. Columns are matched to struct
// fields by name (case-insensitively): the db:"col" tag wins, otherwise the
// field name. Unknown columns are ignored; missing fields stay zero.
func scanInto(rows *sql.Rows, dst any) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("data: columns: %w", err)
	}

	rv := reflect.ValueOf(dst).Elem()

	// A non-struct target (int, string, time.Time, ...) scans a single column
	// directly — enough for count(*) and similar scalar queries.
	if rv.Kind() != reflect.Struct {
		if len(cols) != 1 {
			return fmt.Errorf("data: cannot scan %d columns into %s", len(cols), rv.Type())
		}
		if err := rows.Scan(dst); err != nil {
			return fmt.Errorf("data: scan: %w", err)
		}
		return nil
	}

	fields := fieldsOf(rv.Type())

	dest := make([]any, len(cols))
	for i, name := range cols {
		fi, ok := fields[strings.ToLower(name)]
		if !ok {
			dest[i] = new(any) // no matching field: discard
			continue
		}
		fv := rv.FieldByIndex(fi.index)
		if !fv.CanAddr() || !fv.CanSet() {
			dest[i] = new(any)
			continue
		}
		dest[i] = fv.Addr().Interface()
	}

	if err := rows.Scan(dest...); err != nil {
		return fmt.Errorf("data: scan: %w", err)
	}
	return nil
}

type fieldInfo struct {
	index []int
}

var fieldCache sync.Map // reflect.Type -> map[string]fieldInfo

// fieldsOf returns the exported, scannable fields of t keyed by lower-cased
// match name. Results are cached per type.
func fieldsOf(t reflect.Type) map[string]fieldInfo {
	if v, ok := fieldCache.Load(t); ok {
		return v.(map[string]fieldInfo)
	}
	m := map[string]fieldInfo{}
	collectFields(t, nil, m)
	fieldCache.Store(t, m)
	return m
}

// collectFields walks t depth-first, flattening exported embedded structs (no db
// tag) so their fields are matched at the top level. Unexported fields are
// skipped, so reflection can only ever read/write what the author exported.
func collectFields(t reflect.Type, prefix []int, m map[string]fieldInfo) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// Skip unexported, non-embedded fields. Embedded fields are walked even
		// when their type name is unexported, so their exported fields flatten in
		// (assignment falls back to a discard when reflection cannot set them).
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}
		tag := f.Tag.Get("db")
		if tag == "-" {
			continue
		}

		index := make([]int, 0, len(prefix)+1)
		index = append(index, prefix...)
		index = append(index, i)

		if f.Anonymous && tag == "" && f.Type.Kind() == reflect.Struct {
			collectFields(f.Type, index, m)
			continue
		}

		key := tag
		if key == "" {
			key = f.Name
		}
		m[strings.ToLower(key)] = fieldInfo{index: index}
	}
}
