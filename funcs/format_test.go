package funcs

import (
	"html/template"
	"strings"
	"testing"
	"time"
)

func TestCase(t *testing.T) {
	if got := upper("Go"); got != "GO" {
		t.Errorf("upper = %q", got)
	}
	if got := lower("Go"); got != "go" {
		t.Errorf("lower = %q", got)
	}
	if got := capitalize("hello world"); got != "Hello world" {
		t.Errorf("capitalize = %q", got)
	}
	if got := capitalize(""); got != "" {
		t.Errorf("capitalize empty = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncatechars(5, "hello world"); got != "hell…" {
		t.Errorf("truncatechars = %q", got)
	}
	if got := truncatechars(20, "short"); got != "short" {
		t.Errorf("truncatechars under n = %q", got)
	}
	if got := truncatewords(2, "the quick brown fox"); got != "the quick …" {
		t.Errorf("truncatewords = %q", got)
	}
	if got := truncatewords(9, "too short"); got != "too short" {
		t.Errorf("truncatewords under n = %q", got)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello, World!":  "hello-world",
		"  Go   Rocks  ": "go-rocks",
		"C++ & You":      "c-you",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoin(t *testing.T) {
	got, err := join(", ", []string{"a", "b", "c"})
	if err != nil || got != "a, b, c" {
		t.Errorf("join = %q, %v", got, err)
	}
	got, err = join("-", []int{1, 2, 3})
	if err != nil || got != "1-2-3" {
		t.Errorf("join ints = %q, %v", got, err)
	}
	if _, err := join(",", "nope"); err == nil {
		t.Error("join on a string should error")
	}
}

func TestDefault(t *testing.T) {
	if got := defaultVal("anon", ""); got != "anon" {
		t.Errorf("default empty string = %v", got)
	}
	if got := defaultVal("anon", "Ada"); got != "Ada" {
		t.Errorf("default value = %v", got)
	}
	if got := defaultVal(0, []int{}); got != 0 {
		t.Errorf("default empty slice = %v", got)
	}
	var p *int
	if got := defaultVal("nil", p); got != "nil" {
		t.Errorf("default nil ptr = %v", got)
	}
}

func TestPluralize(t *testing.T) {
	if got := pluralize(1); got != "" {
		t.Errorf("pluralize(1) = %q", got)
	}
	if got := pluralize(2); got != "s" {
		t.Errorf("pluralize(2) = %q", got)
	}
	if got := pluralize(2, "items"); got != "items" {
		t.Errorf("pluralize(2, suffix) = %q", got)
	}
	if got := pluralize(1, "item", "items"); got != "item" {
		t.Errorf("pluralize(1, sing, plur) = %q", got)
	}
}

func TestComma(t *testing.T) {
	cases := map[int]string{5: "5", 1234567: "1,234,567", -1000: "-1,000", 999: "999"}
	for n, want := range cases {
		if got := comma(n); got != want {
			t.Errorf("comma(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestDate(t *testing.T) {
	tm := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	if got := date("2006-01-02", tm); got != "2026-01-02" {
		t.Errorf("date = %q", got)
	}
}

func TestLinebreaksbr(t *testing.T) {
	got := linebreaksbr("a\nb <c>")
	if got != "a<br>\nb &lt;c&gt;" {
		t.Errorf("linebreaksbr = %q", got)
	}
	if _, ok := any(got).(template.HTML); !ok {
		t.Error("linebreaksbr should return template.HTML")
	}
	if !strings.Contains(string(got), "<br>") {
		t.Errorf("linebreaksbr missing <br>: %q", got)
	}
}
