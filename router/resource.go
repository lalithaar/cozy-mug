// Package router (P2): resourceful routing.
//
// Router.Resource registers the conventional Rails-style routes for a
// controller: index, new, create, show, edit, update, destroy. The controller
// is a struct whose methods are handlers; missing methods simply skip that
// route. Route names are <resource>.index, <resource>.new, etc. The member
// param is {id}.
//
// Example:
//
//	type PeopleController struct{}
//	func (c PeopleController) Index(ctx *core.Context) core.Response {...}
//	func (c PeopleController) Show(ctx *core.Context) core.Response {...}
//	r := router.New()
//	r.Resource("people", PeopleController{})
//
// Registers:
//
//	GET    /people           people.index
//	GET    /people/new       people.new
//	POST   /people           people.create
//	GET    /people/{id}      people.show
//	GET    /people/{id}/edit people.edit
//	PUT    /people/{id}      people.update
//	PATCH  /people/{id}      people.update
//	DELETE /people/{id}      people.destroy
package router

import (
	"reflect"

	"github.com/lalithaar/cozy-mug/core"
)

// handlerOf extracts a core.Handler from a controller method named name. It
// returns nil when the method is absent or has the wrong signature. A valid
// handler method is func(*core.Context) core.Response.
func handlerOf(controller reflect.Value, name string) core.Handler {
	m := controller.MethodByName(name)
	if !m.IsValid() || m.IsNil() {
		return nil
	}
	mt := m.Type()
	if mt.NumIn() != 1 || mt.NumOut() != 1 {
		return nil
	}
	ctxType := mt.In(0)
	if ctxType != reflect.TypeOf((*core.Context)(nil)) {
		return nil
	}
	respType := mt.Out(0)
	if respType != reflect.TypeOf((*core.Response)(nil)).Elem() {
		return nil
	}
	return func(c *core.Context) core.Response {
		out := m.Call([]reflect.Value{reflect.ValueOf(c)})
		return out[0].Interface().(core.Response)
	}
}

// Resource registers conventional resource routes for controller. name is the
// plural resource name used in paths and route names (e.g. "people"). The
// member route param is {id}. Methods on controller that match
// func(*core.Context) core.Response are wired; absent methods are skipped.
//
// Routes:
//
//	GET    /{name}           name.index
//	GET    /{name}/new       name.new
//	POST   /{name}           name.create
//	GET    /{name}/{id}      name.show
//	GET    /{name}/{id}/edit name.edit
//	PUT    /{name}/{id}      name.update
//	PATCH  /{name}/{id}      name.update
//	DELETE /{name}/{id}      name.destroy
func (rt *Router) Resource(name string, controller any) {
	v := reflect.ValueOf(controller)
	if v.Kind() != reflect.Struct {
		panic("router: Resource controller must be a struct")
	}

	index := handlerOf(v, "Index")
	newH := handlerOf(v, "New")
	create := handlerOf(v, "Create")
	show := handlerOf(v, "Show")
	edit := handlerOf(v, "Edit")
	update := handlerOf(v, "Update")
	destroy := handlerOf(v, "Destroy")

	base := "/" + name
	member := base + "/{id}"

	if index != nil {
		rt.Get(base, index).As(name + ".index")
	}
	if newH != nil {
		rt.Get(base+"/new", newH).As(name + ".new")
	}
	if create != nil {
		rt.Post(base, create).As(name + ".create")
	}
	if show != nil {
		rt.Get(member, show).As(name + ".show")
	}
	if edit != nil {
		rt.Get(member+"/edit", edit).As(name + ".edit")
	}
	if update != nil {
		rt.Put(member, update).As(name + ".update")
		rt.Patch(member, update)
	}
	if destroy != nil {
		rt.Delete(member, destroy).As(name + ".destroy")
	}
}
