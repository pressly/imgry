package main

import (
	"log"

	"github.com/pressly/cji"
)

/*

/

TokenAuth, SessionCtx - /auth
TokenAuth, SessionCtx, HubCtx - /hubs


... would be cool if it generated an HTML page from this........
.. and we could mount /debug/routes ... etc........


... and finally.. run a thing that connects to roles, and verifies route permissions.. are all sane..?

GET   /hubs
POST  /hubs
GET   /hubs/:id
        - SessionCtx, A, B, C


SessionCtx, B, C
  -> GET /hubs

X, Y, Z
  -> POST /hubs

... group the middlewares...?


ughhhhh..

*** use rails rake routes for inspiration...

*/

// TODO: put this in a separate pacakge.. not inside cji ..  cjiroutes  ...?
type DebugMux struct {
	*cji.Mux // or cji.Router .......?

	RootMux *DebugMux
	Routes  []Route
}

type Route struct {
	Method     string
	Path       interface{}
	Middleware []interface{}

	// ... so... how do we want output.. we need to show middleware.. and inline middleware..
	// perhaps groups...? with
}

func NewDebugRouter(r *cji.Mux) *DebugMux {
	return &DebugMux{Mux: r, RootMux: &DebugMux{Mux: r}}
}

var _ cji.Router = &DebugMux{}

func (r *DebugMux) PrintRoutes() {
	log.Println("wooot!")

	for _, rt := range r.RootMux.Routes {
		log.Println(rt.Method, rt.Path)
	}
}

func (r *DebugMux) Use(middlewares ...interface{}) {
	r.Mux.Use(middlewares...)
}

func (r *DebugMux) Handle(pattern interface{}, handlers ...interface{}) {
	r.Mux.Handle(pattern, handlers...)
}

func (r *DebugMux) Connect(pattern interface{}, handlers ...interface{}) {
	r.Mux.Connect(pattern, handlers...)
}

func (r *DebugMux) Head(pattern interface{}, handlers ...interface{}) {
	r.Mux.Head(pattern, handlers...)
}

func (r *DebugMux) Get(pattern interface{}, handlers ...interface{}) {
	log.Println("Get call for..", pattern)
	rt := Route{Method: "GET", Path: pattern}
	r.RootMux.Routes = append(r.RootMux.Routes, rt)

	r.Mux.Get(pattern, handlers...)
}

func (r *DebugMux) Post(pattern interface{}, handlers ...interface{}) {
	r.Mux.Post(pattern, handlers...)
}

func (r *DebugMux) Put(pattern interface{}, handlers ...interface{}) {
	r.Mux.Put(pattern, handlers...)
}

func (r *DebugMux) Patch(pattern interface{}, handlers ...interface{}) {
	r.Mux.Patch(pattern, handlers...)
}

func (r *DebugMux) Delete(pattern interface{}, handlers ...interface{}) {
	r.Mux.Delete(pattern, handlers...)
}

func (r *DebugMux) Trace(pattern interface{}, handlers ...interface{}) {
	r.Mux.Trace(pattern, handlers...)
}

func (r *DebugMux) Options(pattern interface{}, handlers ...interface{}) {
	r.Mux.Options(pattern, handlers...)
}

func (r *DebugMux) Group(fn func(r cji.Router)) cji.Router {
	return r.Mux.Group(fn)
}

func (r *DebugMux) Route(pattern string, fn func(r cji.Router)) cji.Router {
	return r.Mux.Route(pattern, fn)
}

func (r *DebugMux) Mount(path string, handlers ...interface{}) {

	r.Mux.Mount(path, handlers...)
}
