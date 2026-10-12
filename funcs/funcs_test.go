package funcs

import "testing"

func TestArithmetic(t *testing.T) {
	if got := add(2, 3); got != 5 {
		t.Errorf("add(2,3) = %d, want 5", got)
	}
	if got := sub(5, 3); got != 2 {
		t.Errorf("sub(5,3) = %d, want 2", got)
	}
}

func TestLoopHelpers(t *testing.T) {
	if got := counter(0); got != 1 {
		t.Errorf("counter(0) = %d, want 1", got)
	}
	if !first(0) || first(1) {
		t.Errorf("first: got first(0)=%v first(1)=%v", first(0), first(1))
	}
	if !even(0) || even(1) {
		t.Errorf("even: got even(0)=%v even(1)=%v", even(0), even(1))
	}
	if !odd(1) || odd(2) {
		t.Errorf("odd: got odd(1)=%v odd(2)=%v", odd(1), odd(2))
	}
}

func TestLast(t *testing.T) {
	items := []string{"a", "b", "c"}
	cases := []struct {
		i    int
		want bool
	}{{0, false}, {1, false}, {2, true}}
	for _, tc := range cases {
		got, err := last(tc.i, items)
		if err != nil {
			t.Fatalf("last(%d): %v", tc.i, err)
		}
		if got != tc.want {
			t.Errorf("last(%d) = %v, want %v", tc.i, got, tc.want)
		}
	}
	if got, err := last(1, "ab"); err != nil || !got {
		t.Errorf("last(1, \"ab\") = %v, %v; want true, nil", got, err)
	}
	if _, err := last(0, 42); err == nil {
		t.Error("last on an int should error")
	}
}

func TestCycle(t *testing.T) {
	if got, _ := cycle(0, "odd", "even"); got != "odd" {
		t.Errorf("cycle(0) = %q, want odd", got)
	}
	if got, _ := cycle(1, "odd", "even"); got != "even" {
		t.Errorf("cycle(1) = %q, want even", got)
	}
	if got, _ := cycle(2, "odd", "even"); got != "odd" {
		t.Errorf("cycle(2) = %q, want odd", got)
	}
	if _, err := cycle(0); err == nil {
		t.Error("cycle with no items should error")
	}
}

func TestFuncsRegistered(t *testing.T) {
	m := Funcs()
	for _, name := range []string{
		"add", "sub", "counter", "first", "last", "even", "odd", "cycle",
		"upper", "lower", "capitalize", "truncatechars", "truncatewords",
		"slugify", "join", "default", "pluralize", "comma", "date", "linebreaksbr",
	} {
		if _, ok := m[name]; !ok {
			t.Errorf("Funcs() missing %q", name)
		}
	}
}
