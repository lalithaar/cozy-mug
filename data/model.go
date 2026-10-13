package data

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Model is embedded by every model struct (0016). It marks the struct as an
// entity: the framework derives schema, queries, CRUD, and relations from the
// struct it is embedded in. The tag carries the table name:
//
//	type Post struct {
//	    data.Model `db:"posts"`
//	    ...
//	}
type Model struct{}

// ---------------------------------------------------------------------------
// Descriptors — the derived, cached shape of a model.
// ---------------------------------------------------------------------------

var descIndex sync.Map // reflect.Type -> *modelDesc, built lazily on first use

// descFor returns the descriptor for a model type, deriving it from the struct
// definition on first use and caching it (0016: the model is the single source
// of truth; there is no registration step).
func descFor(t reflect.Type) (*modelDesc, error) {
	if v, ok := descIndex.Load(t); ok {
		return v.(*modelDesc), nil
	}
	d, err := buildDesc(t)
	if err != nil {
		return nil, err
	}
	descIndex.Store(t, d)
	return d, nil
}

// column is one scalar column derived from a model field.
type column struct {
	name   string // db column name
	field  string // Go field name
	idx    []int  // FieldByIndex path from the struct root
	goType reflect.Type
	auto   bool
	index  bool
	unique bool
	valid  validate // validation rules (F11) — captured now, enforced by forms
}

// relation is an entity association declared as a model-typed field.
type relation struct {
	field  string
	typ    reflect.Type // child model type
	isMany bool         // []*Child (has-many); false = *Child (belongs-to)
	fk     string       // foreign-key column (on this model when isMany==false, on the child when true)
}

type modelDesc struct {
	typ       reflect.Type
	table     string
	cols      []*column
	coln      map[string]*column // lowercase column name -> column
	rels      map[string]*relation
	pk        *column // primary key (the auto column, else the one named "id")
	auto      *column // auto-increment pk, if any
	createdAt *column // maintained automatically when present
	updatedAt *column
	version   *column // optimistic-lock column, maintained when present
}

// tableName returns M's table name (from the embedded data.Model tag).
func tableName(t reflect.Type) (string, error) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type == reflect.TypeOf(Model{}) {
			if n := f.Tag.Get("db"); n != "" {
				return n, nil
			}
			return "", fmt.Errorf("data: %s: tag the embedded data.Model with a table name: data.Model `db:%s<table>%s`", t, `"`, `"`)
		}
	}
	return "", fmt.Errorf("data: %s: must embed data.Model (data.Model `db:%s<table>%s`)", t, `"`, `"`)
}

// buildDesc walks t's fields once, extracting the table, columns, pk,
// timestamps, version, and relations.
func buildDesc(t reflect.Type) (*modelDesc, error) {
	table, err := tableName(t)
	if err != nil {
		return nil, err
	}

	d := &modelDesc{typ: t, table: table, coln: map[string]*column{}, rels: map[string]*relation{}}
	if err := walkFields(t, nil, t, d); err != nil {
		return nil, err
	}

	// pk: the auto column, else the column named "id".
	switch {
	case d.auto != nil:
		d.pk = d.auto
	default:
		for _, c := range d.cols {
			if strings.ToLower(c.name) == "id" {
				d.pk = c
				break
			}
		}
	}

	// Timestamps and version by column-name convention.
	for _, c := range d.cols {
		switch strings.ToLower(c.name) {
		case "created_at":
			d.createdAt = c
		case "updated_at":
			d.updatedAt = c
		case "version":
			d.version = c
		}
	}
	return d, nil
}

// walkFields collects columns and relations. It flattens exported embedded
// structs other than data.Model so their fields are columns of this table.
func walkFields(t reflect.Type, prefix []int, root reflect.Type, d *modelDesc) error {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		idx := appendIdx(prefix, i)

		// The data.Model marker: skip (its tag already gave us the table name).
		if f.Anonymous && f.Type == reflect.TypeOf(Model{}) {
			continue
		}

		// Exported embedded struct (not a model): flatten its columns in.
		if f.Anonymous && f.Tag.Get("db") == "" && f.Type.Kind() == reflect.Struct && !isModelType(f.Type) {
			if err := walkFields(f.Type, idx, root, d); err != nil {
				return err
			}
			continue
		}

		tag := f.Tag.Get("db")
		if tag == "-" {
			continue
		}

		// Relations: model-typed fields (*T / []*T) must carry db:"ref:<fk>".
		if tt, isMany, ok := relationShape(f.Type); ok {
			if isModelType(tt) {
				fk := strings.TrimPrefix(tag, "ref:")
				if fk == "" || fk == tag {
					return fmt.Errorf("data: %s.%s: model-typed field needs db:%sref:<foreign-key-column>%s (or db:%s-%s to ignore)", root, f.Name, "`", "`", "`", "`")
				}
				d.rels[f.Name] = &relation{field: f.Name, typ: tt, isMany: isMany, fk: strings.ToLower(fk)}
				continue
			}
		}

		name, flags := parseDbTag(tag)
		if name == "" {
			return fmt.Errorf("data: %s.%s: missing column name in db tag (db:%s<col>%s)", root, f.Name, "`", "`")
		}
		name = strings.ToLower(name)

		c := &column{
			name:   name,
			field:  f.Name,
			idx:    idx,
			goType: f.Type,
			auto:   flags["auto"],
			index:  flags["index"],
			unique: flags["unique"],
			valid:  validateOf(f.Tag),
		}
		if c.auto {
			d.auto = c
		}
		d.coln[name] = c
		d.cols = append(d.cols, c)
	}
	return nil
}

func appendIdx(prefix []int, i int) []int {
	out := make([]int, 0, len(prefix)+1)
	out = append(out, prefix...)
	return append(out, i)
}

// parseDbTag splits `db:"col auto index unique"`.
func parseDbTag(tag string) (string, map[string]bool) {
	parts := strings.Fields(tag)
	if len(parts) == 0 {
		return "", nil
	}
	flags := map[string]bool{}
	for _, p := range parts[1:] {
		flags[p] = true
	}
	return parts[0], flags
}

// relationShape reports whether t has the shape of a relation and returns the
// child type. *T -> belongs-to/has-one; []*T -> has-many.
func relationShape(t reflect.Type) (child reflect.Type, isMany, ok bool) {
	switch {
	case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct:
		return t.Elem(), false, true
	case t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Ptr && t.Elem().Elem().Kind() == reflect.Struct:
		return t.Elem().Elem(), true, true
	}
	return nil, false, false
}

// isModelType reports whether tt embeds data.Model.
func isModelType(tt reflect.Type) bool {
	for i := 0; i < tt.NumField(); i++ {
		if f := tt.Field(i); f.Anonymous && f.Type == reflect.TypeOf(Model{}) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Validation-rule capture (F11) — stored now, enforced by the forms layer.
// ---------------------------------------------------------------------------

type validate struct {
	required   bool
	min, max   string
	email, url bool
	oneOf      string
}

func validateOf(sf reflect.StructTag) validate {
	return validate{
		required: has(sf, "required"),
		min:      sf.Get("min"),
		max:      sf.Get("max"),
		email:    has(sf, "email"),
		url:      has(sf, "url"),
		oneOf:    sf.Get("oneof"),
	}
}

func has(sf reflect.StructTag, key string) bool { _, ok := sf.Lookup(key); return ok }

// ---------------------------------------------------------------------------
// Scanning a model row.
// ---------------------------------------------------------------------------

// scanModel scans the current row into v (a model ptr). Columns are matched by
// column name against the descriptor; relation fields and the Model marker are
// not columns and are left alone (With populates them later). A NULL in a
// scalar column is an error unless the field is a pointer (then it stays nil).
func scanModel(m *modelDesc, rows rowsScanner, v reflect.Value) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("data: columns: %w", err)
	}

	dest := make([]any, len(cols))
	for i, name := range cols {
		c, ok := m.coln[strings.ToLower(name)]
		if !ok {
			dest[i] = new(any) // unknown column: discard
			continue
		}
		fv := v.FieldByIndex(c.idx)
		if !fv.CanAddr() {
			dest[i] = new(any)
			continue
		}
		// Drivers store time columns differently (modernc -> text, pgx ->
		// time.Time); scan into a buffer and convert after the row is read.
		if isTimeType(c.goType) {
			dest[i] = new(any)
			continue
		}
		dest[i] = fv.Addr().Interface()
	}

	if err := rows.Scan(dest...); err != nil {
		return fmt.Errorf("data: scan: %w", err)
	}

	for i, name := range cols {
		c, ok := m.coln[strings.ToLower(name)]
		if !ok || !isTimeType(c.goType) {
			continue
		}
		raw := *(dest[i].(*any))
		if raw == nil {
			continue // *time.Time stayed nil
		}
		tv, err := asTime(raw)
		if err != nil {
			return fmt.Errorf("data: scan: %s: %w", name, err)
		}
		fv := v.FieldByIndex(c.idx)
		if fv.Kind() == reflect.Ptr {
			fv.Set(reflect.ValueOf(&tv))
		} else {
			fv.Set(reflect.ValueOf(tv))
		}
	}
	return nil
}

// isTimeType reports whether t is time.Time or *time.Time.
func isTimeType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == reflect.TypeOf(time.Time{})
}

// asTime converts whatever a driver handed back for a time column into
// time.Time. Text formats follow the drivers' defaults (modernc writes
// "2006-01-02 15:04:05.999999999-07:00", an RFC3339 without the T).
func asTime(raw any) (time.Time, error) {
	switch x := raw.(type) {
	case time.Time:
		return x, nil
	case string:
		return parseTimeText(x)
	case []byte:
		return parseTimeText(string(x))
	}
	return time.Time{}, fmt.Errorf("unsupported time value %T", raw)
}

func parseTimeText(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999 -0700 MST", // time.Time.String()
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time text %q", s)
}

// rowsScanner is the rows subset scanModel needs; *sql.Rows satisfies it.
type rowsScanner interface {
	Columns() ([]string, error)
	Scan(...any) error
}

// columnValue reads a model value's cell for a column (used by mutations).
func (m *modelDesc) columnValue(v reflect.Value, c *column) reflect.Value {
	return v.FieldByIndex(c.idx)
}

// ---------------------------------------------------------------------------
// DDL (F7) — Go scalar -> dialect type, and a CREATE TABLE. Interim: powers
// EnsureSchema until 0016's generated-migrations runner lands.
// ---------------------------------------------------------------------------

func scalarType(t reflect.Type, names map[reflect.Kind]string) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return "BLOB"
	}
	return names[t.Kind()]
}

var sqliteKinds = map[reflect.Kind]string{
	reflect.String:  "TEXT",
	reflect.Int:     "INTEGER",
	reflect.Int8:    "INTEGER",
	reflect.Int16:   "INTEGER",
	reflect.Int32:   "INTEGER",
	reflect.Int64:   "INTEGER",
	reflect.Bool:    "INTEGER",
	reflect.Float32: "REAL",
	reflect.Float64: "REAL",
}

var pgKinds = map[reflect.Kind]string{
	reflect.String:  "TEXT",
	reflect.Int:     "INTEGER",
	reflect.Int8:    "INTEGER",
	reflect.Int16:   "INTEGER",
	reflect.Int32:   "INTEGER",
	reflect.Int64:   "BIGINT",
	reflect.Bool:    "BOOLEAN",
	reflect.Float32: "REAL",
	reflect.Float64: "DOUBLE PRECISION",
}

// DDL returns the CREATE TABLE statement for a dialect plus its index
// statements.
func (m *modelDesc) DDL(dialect string) (create string, indexes []string, err error) {
	var cols []string
	for _, c := range m.cols {
		if c.auto {
			if dialect == "postgres" {
				cols = append(cols, c.name+" BIGSERIAL PRIMARY KEY")
			} else {
				cols = append(cols, c.name+" INTEGER PRIMARY KEY")
			}
			continue
		}
		kinds := sqliteKinds
		if dialect == "postgres" {
			kinds = pgKinds
		}
		var typ string
		if isTimeType(c.goType) {
			if dialect == "postgres" {
				typ = "TIMESTAMPTZ"
			} else {
				typ = "TEXT" // RFC3339 text; consumed by asTime on scan
			}
		} else {
			typ = scalarType(c.goType, kinds)
		}
		if typ == "" {
			return "", nil, fmt.Errorf("data: %s.%s: unsupported column type %s for %s", m.typ, c.field, c.goType, dialect)
		}
		line := c.name + " " + typ
		if c.goType.Kind() != reflect.Ptr {
			line += " NOT NULL"
		}
		cols = append(cols, line)
	}

	create = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)", m.table, strings.Join(cols, ",\n  "))

	for _, c := range m.cols {
		switch {
		case c.unique:
			indexes = append(indexes, fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS uniq_%s_%s ON %s (%s)", m.table, c.name, m.table, c.name))
		case c.index:
			indexes = append(indexes, fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s)", m.table, c.name, m.table, c.name))
		}
	}
	return create, indexes, nil
}
