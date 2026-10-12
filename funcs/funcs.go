// Package funcs is an optional library of small, useful template functions.
//
// It is deliberately not built into views: html/template ships almost no
// arithmetic or loop helpers, and the framework's answer is a library you opt
// into (see docs/decisions/0011), matching the ui package. Add it with
// app.Config.Funcs:
//
//	app.Must(app.Config{Assets: assets, Funcs: funcs.Funcs(), ...})
//
// Scope so far: loop metadata (counter, first, last, even, odd, cycle),
// arithmetic (add, sub), and formatting (upper, lower, capitalize, truncatechars,
// truncatewords, slugify, join, default, pluralize, comma, date, linebreaksbr).
package funcs

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
)

// Funcs returns the library's template functions.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"add":           add,
		"sub":           sub,
		"counter":       counter,
		"first":         first,
		"last":          last,
		"even":          even,
		"odd":           odd,
		"cycle":         cycle,
		"upper":         upper,
		"lower":         lower,
		"capitalize":    capitalize,
		"truncatechars": truncatechars,
		"truncatewords": truncatewords,
		"slugify":       slugify,
		"join":          join,
		"default":       defaultVal,
		"pluralize":     pluralize,
		"comma":         comma,
		"date":          date,
		"linebreaksbr":  linebreaksbr,
	}
}

// add is a + b. html/template has no arithmetic operators.
func add(a, b int) int { return a + b }

// sub is a - b.
func sub(a, b int) int { return a - b }

// counter is the 1-based loop number for a {{range}} index (Django's
// forloop.counter).
func counter(i int) int { return i + 1 }

// first reports whether i is the first {{range}} index.
func first(i int) bool { return i == 0 }

// last reports whether i is the last index of s (a string, slice, or array).
func last(i int, s any) (bool, error) {
	n, err := count(s)
	if err != nil {
		return false, err
	}
	return i == n-1, nil
}

// even reports whether i is even.
func even(i int) bool { return i%2 == 0 }

// odd reports whether i is odd.
func odd(i int) bool { return i%2 != 0 }

// cycle returns items[i mod len(items)]; use it for zebra striping, e.g.
// class="{{cycle $i "odd" "even"}}".
func cycle(i int, items ...string) (string, error) {
	if len(items) == 0 {
		return "", errors.New("funcs: cycle needs at least one item")
	}
	return items[i%len(items)], nil
}

// count returns the length of a string, slice, or array.
func count(s any) (int, error) {
	if str, ok := s.(string); ok {
		return len(str), nil
	}
	switch rv := reflect.ValueOf(s); rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len(), nil
	}
	return 0, fmt.Errorf("funcs: cannot take the length of %T", s)
}
