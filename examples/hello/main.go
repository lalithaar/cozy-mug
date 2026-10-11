package main

import (
	"log"
	"net/http"

	"cozymug/core"
	"cozymug/router"
)

func main() {
	r := router.New()
	r.Use(logger)

	r.Get("/", func(c *core.Context) core.Response {
		link, _ := c.URL("user", map[string]string{"id": "1"})
		return core.String(http.StatusOK, `<h1>Home</h1><a href="`+link+`">User 1</a>`)
	}).As("home")

	r.Get("/users/{id}", func(c *core.Context) core.Response {
		return core.String(http.StatusOK, "<h1>User "+c.Param("id")+"</h1>")
	}).As("user")

	r.Post("/users", func(c *core.Context) core.Response {
		name := c.Form("name")
		if name == "" {
			return core.Error(http.StatusBadRequest, "name required")
		}
		return core.Redirect(http.StatusSeeOther, "/")
	})

	admin := r.Group("/admin", requireAdmin)
	admin.Get("/users", func(*core.Context) core.Response {
		return core.String(http.StatusOK, "<h1>Admin users</h1>")
	})

	r.Static("/static/", "./static")
	r.NotFound(func(*core.Context) core.Response {
		return core.Error(http.StatusNotFound, "Page not found")
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

func logger(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		log.Printf("%s %s", c.Method(), c.Path())
		return next(c)
	}
}

func requireAdmin(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		if c.Query("admin") != "1" {
			return core.Error(http.StatusForbidden, "Forbidden")
		}
		return next(c)
	}
}
