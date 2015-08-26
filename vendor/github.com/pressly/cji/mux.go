package cji

import (
	"net/http"

	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

type Mux struct {
	*web.Mux

	inline      bool
	middlewares []interface{}
}

var _ Router = &Mux{}

func (r *Mux) Use(middlewares ...interface{}) {
	if r.inline {
		iw := Use(middlewares...)
		r.middlewares = append(r.middlewares, iw.middlewares...)
	} else {
		for _, mw := range middlewares {
			r.Mux.Use(mw)
		}
	}
}

func (r *Mux) Handle(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Handle(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Handle(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Connect(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Connect(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Connect(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Head(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Head(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Head(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Get(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Get(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Get(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Post(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Post(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Post(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Put(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Put(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Put(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Patch(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Patch(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Patch(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Delete(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Delete(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Delete(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Trace(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Trace(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Trace(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Options(pattern interface{}, handlers ...interface{}) {
	if r.inline {
		r.Mux.Options(pattern, r.chain(append(r.middlewares, handlers...)...))
	} else {
		r.Mux.Options(pattern, r.chain(handlers...))
	}
}

func (r *Mux) Group(fn func(r Router)) Router {
	mw := make([]interface{}, len(r.middlewares))
	copy(mw, r.middlewares)

	g := &Mux{Mux: r.Mux, inline: true, middlewares: mw}
	if fn != nil {
		fn(g)
	}
	return g
}

func (r *Mux) Route(pattern string, fn func(r Router)) Router {
	sr := NewRouter()
	r.Mount(pattern, append(r.middlewares, sr)...)
	if fn != nil {
		fn(sr)
	}
	return sr
}

func (r *Mux) Mount(path string, handlers ...interface{}) {
	h := append(r.middlewares, handlers...)
	subRouter := Use(middleware.SubRouter).On(r.chain(h...))

	subRouterIndex := web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
		if c.URLParams == nil {
			c.URLParams = make(map[string]string)
		}
		c.URLParams["*"] = "/"
		subRouter.ServeHTTPC(c, w, r)
	})

	if path == "/" {
		path = ""
	}

	r.Mux.Get(path, subRouterIndex)
	r.Mux.Handle(path, subRouterIndex)
	if path != "" {
		r.Mux.Handle(path+"/", http.NotFound)
	}
	r.Mux.Handle(path+"/*", subRouter)
}

func (r *Mux) chain(handlers ...interface{}) interface{} {
	var h interface{}
	if len(handlers) > 1 {
		mw := handlers[0 : len(handlers)-1]
		handler := handlers[len(handlers)-1]
		h = Use(mw...).On(handler)
	} else {
		h = handlers[0]
	}
	return h
}
