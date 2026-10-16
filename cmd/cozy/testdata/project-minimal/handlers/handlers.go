// Package handlers holds the app's handlers and route registration.
package handlers

import (
	"net/http"

	"github.com/lalithaar/cozy-mug/core"
	"github.com/lalithaar/cozy-mug/router"
)

// Register wires the app's routes.
func Register(r *router.Router) {
	r.Get("/", home).As("home")
}

func home(c *core.Context) core.Response {
	return c.Render(http.StatusOK, "home", nil)
}