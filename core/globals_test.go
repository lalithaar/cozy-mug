package core_test

import (
	"testing"

	"github.com/lalithaar/cozy-mug/core"
)

func TestContextSetGlobal(t *testing.T) {
	c := &core.Context{}
	c.Set("a", 1)
	c.Set("a", 2)
	if got := c.Global("a"); got != 2 {
		t.Fatalf("Global(a) = %v, want 2 (later write wins)", got)
	}
	if got := c.Global("missing"); got != nil {
		t.Fatalf("Global(missing) = %v, want nil", got)
	}
}

func TestGlobalsValues(t *testing.T) {
	c := &core.Context{}
	c.Set("flash", "saved")
	if got := c.Globals().Values()["flash"]; got != "saved" {
		t.Fatalf("Values[flash] = %v, want saved", got)
	}
	var zero core.Globals
	if zero.Values() != nil {
		t.Fatal("zero Globals.Values should be nil")
	}
}

func TestProcessors(t *testing.T) {
	var order []string
	h := core.Processors(
		func(*core.Context) map[string]any { order = append(order, "one"); return map[string]any{"x": 1} },
		nil, // nil processors are skipped
		func(*core.Context) map[string]any {
			order = append(order, "two")
			return map[string]any{"x": 2, "y": "hi"}
		},
	)(func(c *core.Context) core.Response {
		if got := c.Global("x"); got != 2 {
			t.Errorf("x = %v, want 2", got)
		}
		if got := c.Global("y"); got != "hi" {
			t.Errorf("y = %v, want hi", got)
		}
		return nil
	})

	h(&core.Context{})
	if len(order) != 2 || order[0] != "one" || order[1] != "two" {
		t.Fatalf("processors ran in wrong order: %v", order)
	}
}
