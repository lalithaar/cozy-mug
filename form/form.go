// Package form binds an incoming request onto a model, validates it, and carries
// the submitted raw strings back so a re-rendered view can repopulate inputs.
package form

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lalithaar/cozy-mug/core"
)

// Form is the request-bound view of a model value.
type Form[T any] struct {
	Value T
	// RawValues holds the submitted/raw strings by input name. Templates
	// usually go through Raw (below), which is callable as a function.
	RawValues map[string]string

	errs      map[string][]string
	validated bool
}

// Raw returns the raw submitted value for an input name, "" if absent. It is a
// method (not a field) so templates can call it as {{.Form.Raw "title"}}.
func (f *Form[T]) Raw(key string) string {
	if f == nil || f.RawValues == nil {
		return ""
	}
	return f.RawValues[key]
}

type fieldPlan struct {
	key        string
	name       string
	idx        []int
	tag        reflect.StructTag
	omit       bool
	isFileType bool
}

type validateRules struct {
	required bool
	min      string
	max      string
	email    bool
	url      bool
	oneOf    string
}

func New[T any](value T) *Form[T] {
	f := &Form[T]{Value: value, RawValues: map[string]string{}, errs: map[string][]string{}}
	for _, p := range plansFor[T]() {
		if p.omit {
			continue
		}
		fv := reflect.ValueOf(&f.Value).Elem().FieldByIndex(p.idx)
		f.RawValues[p.key] = stringify(fv)
	}
	return f
}

func Bind[T any](c *core.Context) *Form[T] {
	var zero T
	f := New(zero)
	f.bindRequest(c)
	return f
}

func BindOver[T any](c *core.Context, base *T) *Form[T] {
	var v T
	if base != nil {
		v = *base
	}
	f := New(v)
	f.bindRequest(c)
	return f
}

func (f *Form[T]) bindRequest(c *core.Context) {
	r := c.R
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		_ = r.ParseMultipartForm(32 << 20)
	} else {
		_ = r.ParseForm()
	}

	for _, p := range plansFor[T]() {
		if p.omit || p.isFileType {
			continue
		}
		fv := reflect.ValueOf(&f.Value).Elem().FieldByIndex(p.idx)
		values, present := r.Form[p.key]
		if !present {
			if fv.Kind() == reflect.Bool {
				fv.SetBool(false)
				f.RawValues[p.key] = ""
			}
			if fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Bool {
				fv.Set(reflect.Zero(fv.Type()))
				f.RawValues[p.key] = ""
			}
			continue
		}
		f.RawValues[p.key] = firstOrJoined(values)
		_ = coerceInto(fv, values, f.errs, p.key, p.name)
	}
}

func isFileType(t reflect.Type) bool {
	return t.String() == "*multipart.FileHeader"
}

func (f *Form[T]) Valid() bool { return f.Validate() }

func (f *Form[T]) Validate() bool {
	if f.validated {
		return len(f.errs) == 0
	}
	f.validated = true
	if f.errs == nil {
		f.errs = map[string][]string{}
	}

	for _, p := range plansFor[T]() {
		if p.omit || p.isFileType {
			continue
		}
		fv := reflect.ValueOf(&f.Value).Elem().FieldByIndex(p.idx)
		// skip tag rules if coercion already failed for this field
		if len(f.errs[p.key]) > 0 {
			continue
		}
		runRules(f, p.key, p.name, fv, rulesOf(p.tag))
	}
	callModelHooks(f)
	return len(f.errs) == 0
}

func (f *Form[T]) Errors(field string) []string {
	if f == nil || f.errs == nil {
		return nil
	}
	return f.errs[field]
}

func (f *Form[T]) Error(field string) string {
	if msgs := f.Errors(field); len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

func (f *Form[T]) HasError(field string) bool { return len(f.Errors(field)) > 0 }
func (f *Form[T]) HasAnyError() bool          { return len(f.errs) > 0 }

func (f *Form[T]) AddError(field, msg string) {
	if f.errs == nil {
		f.errs = map[string][]string{}
	}
	f.errs[field] = append(f.errs[field], msg)
	f.validated = true
}

// --- plans ---

func plansFor[T any]() []fieldPlan {
	v := reflect.TypeOf((*T)(nil))
	if v == nil {
		return nil
	}
	t := v.Elem()
	if t.Kind() != reflect.Struct {
		return nil
	}
	var out []fieldPlan
	for i := 0; i < t.NumField(); i++ {
		fld := t.Field(i)
		if fld.Anonymous {
			continue
		}
		if relationLike(fld.Type) {
			continue
		}
		p := buildPlan(fld, []int{i})
		if p == nil {
			continue
		}
		p.isFileType = fld.Type.String() == "*multipart.FileHeader"
		out = append(out, *p)
	}
	return out
}

func buildPlan(f reflect.StructField, idx []int) *fieldPlan {
	formTag := f.Tag.Get("form")
	dbTag := f.Tag.Get("db")

	if formTag == "-" {
		return nil
	}
	key := ""
	if formTag != "" {
		parts := strings.Fields(formTag)
		if len(parts) > 0 {
			if parts[0] == "-" {
				return nil
			}
			key = parts[0]
		}
	}
	if key == "" && dbTag != "" && dbTag != "-" {
		parts := strings.Fields(dbTag)
		if len(parts) > 0 && parts[0] != "-" {
			key = parts[0]
		}
	}
	if key == "" {
		key = lowerFirst(f.Name)
	}
	return &fieldPlan{key: key, name: f.Name, idx: idx, tag: f.Tag}
}

func relationLike(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct {
		return true
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Pointer {
		return true
	}
	return false
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// coerceInto stays same (see below) but errs map[string][]string.
// We redefine cleanly:

func coerceInto(fv reflect.Value, values []string, errs map[string][]string, key, lab string) error {
	isPtr := fv.Kind() == reflect.Pointer
	if isPtr {
		if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
			if !fv.IsNil() {
				fv.Set(reflect.Zero(fv.Type()))
			}
			return nil
		}
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		fv = fv.Elem()
	}

	switch fv.Kind() {
	case reflect.String:
		if len(values) > 0 {
			fv.SetString(values[0])
		} else {
			fv.SetString("")
		}
	case reflect.Bool:
		raw := ""
		if len(values) > 0 {
			raw = values[0]
		}
		b, ok := parseBool(raw)
		if !ok && strings.TrimSpace(raw) != "" {
			errs[key] = append(errs[key], lab+" must be true or false")
			return fmt.Errorf("bool")
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		raw := ""
		if len(values) > 0 {
			raw = strings.TrimSpace(values[0])
		}
		if raw == "" {
			fv.SetInt(0)
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			errs[key] = append(errs[key], lab+" must be a whole number")
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		raw := ""
		if len(values) > 0 {
			raw = strings.TrimSpace(values[0])
		}
		if raw == "" {
			fv.SetUint(0)
			return nil
		}
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			errs[key] = append(errs[key], lab+" must be a whole number")
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		raw := ""
		if len(values) > 0 {
			raw = strings.TrimSpace(values[0])
		}
		if raw == "" {
			fv.SetFloat(0)
			return nil
		}
		n, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			errs[key] = append(errs[key], lab+" must be a number")
			return err
		}
		fv.SetFloat(n)
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.String {
			cleaned := make([]string, 0, len(values))
			for _, v := range values {
				cleaned = append(cleaned, v)
			}
			fv.Set(reflect.ValueOf(cleaned))
		} else {
			errs[key] = append(errs[key], lab+" is not supported")
			return fmt.Errorf("unsupported")
		}
	case reflect.Struct:
		if fv.Type() == reflect.TypeOf(time.Time{}) {
			raw := ""
			if len(values) > 0 {
				raw = strings.TrimSpace(values[0])
			}
			if raw == "" {
				fv.Set(reflect.Zero(fv.Type()))
				return nil
			}
			tt, err := parseTime(raw)
			if err != nil {
				errs[key] = append(errs[key], lab+" must be a valid date")
				return err
			}
			fv.Set(reflect.ValueOf(tt))
		} else {
			errs[key] = append(errs[key], lab+" is not supported")
			return fmt.Errorf("unsupported")
		}
	default:
		errs[key] = append(errs[key], lab+" is not supported")
		return fmt.Errorf("unsupported")
	}
	return nil
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "0", "false", "no", "off", "f", "n":
		return false, true
	case "1", "true", "yes", "on", "t", "y":
		return true, true
	}
	return false, false
}

func parseTime(raw string) (time.Time, error) {
	layouts := []string{"2006-01-02", time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04"}
	for _, l := range layouts {
		if t, err := time.Parse(l, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("bad time")
}

func stringify(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		return stringify(v.Elem())
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, v.Type().Bits())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			parts := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				parts[i] = v.Index(i).String()
			}
			return strings.Join(parts, ",")
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			tt := v.Interface().(time.Time)
			if tt.IsZero() {
				return ""
			}
			return tt.Format("2006-01-02")
		}
	}
	return ""
}

func firstOrJoined(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	if len(vals) == 1 {
		return vals[0]
	}
	return strings.Join(vals, ",")
}

func rulesOf(tag reflect.StructTag) validateRules {
	vr := validateRules{}
	vr.required = tagHas(tag, "required")
	vr.min = tag.Get("min")
	vr.max = tag.Get("max")
	vr.email = tagHas(tag, "email")
	vr.url = tagHas(tag, "url")
	vr.oneOf = tag.Get("oneof")
	return vr
}

func tagHas(sf reflect.StructTag, key string) bool {
	_, ok := sf.Lookup(key)
	return ok
}

func runRules[T any](f *Form[T], key, lab string, fv reflect.Value, r validateRules) {
	if r.required && fv.IsZero() {
		f.AddError(key, lab+" is required")
	}
	if r.min != "" {
		switch fv.Kind() {
		case reflect.String:
			n, err := strconv.Atoi(r.min)
			if err == nil && fv.Len() < n {
				f.AddError(key, fmt.Sprintf("%s must be at least %s characters", lab, r.min))
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bound, err := strconv.ParseInt(r.min, 10, 64)
			if err == nil && fv.Int() < bound {
				f.AddError(key, fmt.Sprintf("%s must be at least %s", lab, r.min))
			}
		case reflect.Float32, reflect.Float64:
			bound, err := strconv.ParseFloat(r.min, 64)
			if err == nil && fv.Float() < bound {
				f.AddError(key, fmt.Sprintf("%s must be at least %s", lab, r.min))
			}
		}
	}
	if r.max != "" {
		switch fv.Kind() {
		case reflect.String:
			n, err := strconv.Atoi(r.max)
			if err == nil && fv.Len() > n {
				f.AddError(key, fmt.Sprintf("%s must be at most %s characters", lab, r.max))
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bound, err := strconv.ParseInt(r.max, 10, 64)
			if err == nil && fv.Int() > bound {
				f.AddError(key, fmt.Sprintf("%s must be at most %s", lab, r.max))
			}
		case reflect.Float32, reflect.Float64:
			bound, err := strconv.ParseFloat(r.max, 64)
			if err == nil && fv.Float() > bound {
				f.AddError(key, fmt.Sprintf("%s must be at most %s", lab, r.max))
			}
		}
	}
	if r.email && fv.Kind() == reflect.String {
		v := fv.String()
		if v != "" {
			parsed, err := mail.ParseAddress(v)
			if err != nil || parsed.Address != v {
				f.AddError(key, lab+" must be a valid email address")
			}
		}
	}
	if r.url && fv.Kind() == reflect.String {
		v := fv.String()
		if v != "" {
			u, err := url.Parse(v)
			if err != nil || u.Scheme == "" || u.Host == "" {
				f.AddError(key, lab+" must be a valid URL")
			}
		}
	}
	if r.oneOf != "" {
		v := fmt.Sprint(fv.Interface())
		if v != "" {
			allowed := strings.Fields(strings.ReplaceAll(r.oneOf, ",", " "))
			ok := false
			for _, a := range allowed {
				if v == a {
					ok = true
					break
				}
			}
			if !ok {
				f.AddError(key, lab+" must be one of: "+strings.Join(allowed, ", "))
			}
		}
	}
}

func callModelHooks[T any](f *Form[T]) {
	v := reflect.ValueOf(&f.Value)
	m := v.MethodByName("Validate")
	if !m.IsValid() {
		return
	}
	mt := m.Type()
	// Validate() []core.FieldError
	if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0) == reflect.TypeOf([]core.FieldError(nil)) {
		out := m.Call(nil)
		if errs, ok := out[0].Interface().([]core.FieldError); ok {
			for _, e := range errs {
				if strings.TrimSpace(e.Message) != "" {
					f.AddError(e.Field, e.Message)
				}
			}
		}
		return
	}
	// Validate() error
	if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		out := m.Call(nil)
		if err, ok := out[0].Interface().(error); ok && err != nil {
			f.AddError("", err.Error())
		}
		return
	}
	// Validate(*Form[T])
	if mt.NumIn() == 1 && mt.In(0) == reflect.TypeOf(f) && mt.NumOut() == 0 {
		m.Call([]reflect.Value{reflect.ValueOf(f)})
		return
	}
}

// override plansFor return type usage: Validate() references plansFor[T]().
func (f *Form[T]) plans() []fieldPlan { return plansFor[T]() }
