package ui

import (
	"bytes"
	"fmt"
	"html/template"
	"unicode"
)

// button renders an accessible <button> backed by an internal partial.
//
//	{{button "Save"}}
//	{{button "Save" (props "variant" "primary" "type" "submit")}}
func button(label string, ps ...Props) (template.HTML, error) {
	p := Props{}
	for _, x := range ps {
		for k, v := range x {
			p[k] = v
		}
	}

	variant := p["variant"]
	delete(p, "variant")
	class := p["class"]
	delete(p, "class")

	attrs := map[string]string{}
	for k, v := range p {
		if !safeAttrName(k) {
			continue
		}
		attrs[k] = v
	}

	var buf bytes.Buffer
	if err := components.ExecuteTemplate(&buf, "cm-button", buttonData{
		Label:   label,
		Variant: variant,
		Class:   class,
		Attrs:   attrs,
	}); err != nil {
		return "", fmt.Errorf("ui: button: %w", err)
	}
	return template.HTML(buf.String()), nil
}

type buttonData struct {
	Label   string
	Variant string
	Class   string
	Attrs   map[string]string
}

// safeAttrName reports whether s is usable as an HTML attribute name without
// any escaping surprises.
func safeAttrName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != ':' && r != '.' {
			return false
		}
	}
	return true
}

