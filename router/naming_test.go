package router_test

import (
	"testing"

	"github.com/lalithaar/cozy-mug/router"
)

func TestDuplicateRouteNamePanics(t *testing.T) {
	rt := router.New()
	rt.Get("/a", ok).As("x")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate route name")
		}
	}()
	rt.Get("/b", ok).As("x")
}
