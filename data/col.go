package data

import (
	"reflect"
	"strings"
)

// Col is a typed column token bound to a model (0016). Generated tokens — or a
// hand-written token set — give every column a compiler-checked name, so a typo
// is impossible and identifiers stay a fixed, model-derived set:
//
//	var P = struct {
//		ID   data.Col
//		Name data.Col
//	}{ID: data.C[Note]("id"), Name: data.C[Note]("name")}
//
// A Col binds at construction; whether its name exists on the model is checked
// when a query is built (a bad token is a loud error, never a string typo).
type Col struct {
	model reflect.Type
	name  string
}

// C binds a column token to model M by its db column name.
func C[M any](name string) Col {
	return Col{model: reflect.TypeOf((*M)(nil)).Elem(), name: strings.ToLower(name)}
}

// Rel is a relation token naming a relation field on a model (used with With).
func R[M any](field string) Rel {
	return Rel{model: reflect.TypeOf((*M)(nil)).Elem(), field: field}
}

// Rel references a relation field on a model for eager loading.
type Rel struct {
	model reflect.Type
	field string
}

// Predicate is one WHERE condition built from a Col. Ops are the bounded set
// (0016): Eq, Ne, Gt, Gte, Lt, Lte, Like, In, NotIn, Nil, NotNil.
type Predicate struct {
	col    Col
	op     string // =, !=, >, >=, <, <=, LIKE, IN, NOT IN, IS NULL, IS NOT NULL
	values []any
}

func p(col Col, op string, vals ...any) Predicate {
	return Predicate{col: col, op: op, values: vals}
}

func (c Col) Eq(v any) Predicate          { return p(c, "=", v) }
func (c Col) Ne(v any) Predicate          { return p(c, "!=", v) }
func (c Col) Gt(v any) Predicate          { return p(c, ">", v) }
func (c Col) Gte(v any) Predicate         { return p(c, ">=", v) }
func (c Col) Lt(v any) Predicate          { return p(c, "<", v) }
func (c Col) Lte(v any) Predicate         { return p(c, "<=", v) }
func (c Col) Like(v any) Predicate        { return p(c, "LIKE", v) }
func (c Col) In(vals ...any) Predicate    { return p(c, "IN", vals...) }
func (c Col) NotIn(vals ...any) Predicate { return p(c, "NOT IN", vals...) }
func (c Col) Nil() Predicate              { return p(c, "IS NULL") }
func (c Col) NotNil() Predicate           { return p(c, "IS NOT NULL") }

// Sort is an ORDER BY term (Asc/Desc).
type Sort struct {
	col Col
	dir string // ASC | DESC
}

func (c Col) Asc() Sort  { return Sort{col: c, dir: "ASC"} }
func (c Col) Desc() Sort { return Sort{col: c, dir: "DESC"} }
