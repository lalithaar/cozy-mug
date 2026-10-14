// Package ui is the optional default component library (ADR 0010).
//
// It provides small, accessible components as template functions plus a base
// stylesheet, so an app looks decent without hand-writing CSS. It is opt-in:
// core never depends on it, and an app is fully functional without it
// (heuristic "just enough styling to be functional").
//
// Components are called like Vue components:
//
//	{{button "Save" (props "variant" "primary" "type" "submit")}}
//
// Each component executes an internal template partial and returns template.HTML,
// so html/template performs the escaping (never hand-build markup with user data).
// Markup uses internal, namespaced "cm-*" classes; the public theming seam is the
// "--cm-*" CSS variables.
//
// Base styling is provided by a vendored copy of inherent.css (ui/static/base.css),
// served at /static/ui/base.css. The component set beyond `button` is still open.
package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
)

// Props are optional attributes passed to a component, built with the "props"
// template helper: (props "variant" "primary" "type" "submit").
type Props map[string]string

// Funcs returns the template funcs exposing ui's components and helpers. The app
// glue merges them into the app's FuncMap when Config.UI is set.
//
// The component set is not decided yet (ADR 0010); only the shared "props"
// builder exists so far.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"props":  props,
		"button": button,
	}
}

// props builds a Props from alternating key/value arguments.
func props(pairs ...string) (Props, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("ui: props needs an even number of arguments, got %d", len(pairs))
	}
	p := make(Props, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		p[pairs[i]] = pairs[i+1]
	}
	return p, nil
}

//go:embed static
var staticFS embed.FS

//go:embed templates
var templateFS embed.FS

var components = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// FS returns the embedded stylesheet filesystem. The app glue mounts it at
// /static/ui/, so the base stylesheet is served at /static/ui/base.css.
func FS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("ui: embedded static FS: " + err.Error())
	}
	return sub
}
