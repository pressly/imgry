package cji

import (
	"net/http"

	"github.com/zenazn/goji/web"
)

type Router interface {
	http.Handler
	web.Handler

	Use(middlewares ...interface{})
	Group(fn func(r Router)) Router
	Route(pattern string, fn func(r Router)) Router
	Mount(path string, handlers ...interface{})

	Handle(pattern interface{}, handlers ...interface{})
	Connect(pattern interface{}, handlers ...interface{})
	Head(pattern interface{}, handlers ...interface{})
	Get(pattern interface{}, handlers ...interface{})
	Post(pattern interface{}, handlers ...interface{})
	Put(pattern interface{}, handlers ...interface{})
	Patch(pattern interface{}, handlers ...interface{})
	Delete(pattern interface{}, handlers ...interface{})
	Trace(pattern interface{}, handlers ...interface{})
	Options(pattern interface{}, handlers ...interface{})
}

func NewRouter() *Mux {
	return &Mux{Mux: web.New()}
}

func Use(middlewares ...interface{}) *Inlineware {
	return (&Inlineware{}).Use(middlewares...)
}
