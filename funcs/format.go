package funcs

import (
	"fmt"
	"html"
	"html/template"
	"reflect"
	"strings"
	"time"
	"unicode"
)

// upper returns s in upper case.
func upper(s string) string { return strings.ToUpper(s) }

// lower returns s in lower case.
func lower(s string) string { return strings.ToLower(s) }

// capitalize upper-cases the first rune of s.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// truncatechars shortens s to at most n runes, ellipsis included.
func truncatechars(n int, s string) string {
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	if n == 1 {
		return "\u2026"
	}
	return string(r[:n-1]) + "\u2026"
}

// truncatewords keeps the first n words of s, appending an ellipsis.
func truncatewords(n int, s string) string {
	fields := strings.Fields(s)
	if n <= 0 || len(fields) <= n {
		return s
	}
	return strings.Join(fields[:n], " ") + " …"
}

// slugify lowercases s and reduces runs of non-alphanumerics to single hyphens
// ("Hello, World!" -> "hello-world").
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// join concatenates a slice or array with sep, stringifying each element.
func join(sep string, s any) (string, error) {
	rv := reflect.ValueOf(s)
	if k := rv.Kind(); k != reflect.Slice && k != reflect.Array {
		return "", fmt.Errorf("funcs: join needs a slice, got %T", s)
	}
	parts := make([]string, rv.Len())
	for i := range parts {
		parts[i] = fmt.Sprint(rv.Index(i).Interface())
	}
	return strings.Join(parts, sep), nil
}

// defaultVal returns value unless it is empty (nil, "", 0, false, or an empty
// collection), in which case it returns fallback. Usage: {{.Name | default "anon"}}.
func defaultVal(fallback, value any) any {
	if isEmpty(value) {
		return fallback
	}
	return value
}

// isEmpty reports whether v is a nil, zero, or empty value.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

// pluralize returns the singular form when count == 1 and the plural otherwise.
// With no args it yields ""/"s"; with one arg that arg is the plural suffix
// (""/"items"); with two args they are the singular and plural forms.
func pluralize(count int, forms ...string) string {
	singular, plural := "", "s"
	switch len(forms) {
	case 0:
	case 1:
		plural = forms[0]
	default:
		singular, plural = forms[0], forms[1]
	}
	if count == 1 {
		return singular
	}
	return plural
}

// comma formats an integer with thousands separators (1234567 -> "1,234,567").
func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// date formats t with a Go reference layout: {{.Created | date "Jan 2, 2006"}}.
func date(layout string, t time.Time) string { return t.Format(layout) }

// linebreaksbr escapes s and turns newlines into <br>. The input is escaped
// first, so the returned markup is safe.
func linebreaksbr(s string) template.HTML {
	s = strings.ReplaceAll(html.EscapeString(s), "\r\n", "\n")
	return template.HTML(strings.ReplaceAll(s, "\n", "<br>\n"))
}
