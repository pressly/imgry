package cji

import (
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web"
)

type Inlineware struct {
	middlewares []interface{}
}

func (z *Inlineware) Use(middlewares ...interface{}) *Inlineware {
	iw := &Inlineware{z.middlewares}
	for _, mw := range middlewares {
		switch t := mw.(type) {
		default:
			panic(fmt.Sprintf("unsupported middleware type: %T", t))
		case func(http.Handler) http.Handler:
		case func(*web.C, http.Handler) http.Handler:
		}
		iw.middlewares = append(iw.middlewares, mw)
	}
	return iw
}

// Compose together the middleware chain and wrap the handler with it
func (z *Inlineware) On(handler interface{}) web.Handler {
	var wh web.Handler
	switch t := handler.(type) {
	case web.Handler:
		wh = t
	case func(web.C, http.ResponseWriter, *http.Request):
		wh = web.HandlerFunc(t)
	case func(http.ResponseWriter, *http.Request):
		wh = web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
			t(w, r)
		})
	default:
		panic(fmt.Sprintf("unsupported handler type: %T", t))
	}

	if len(z.middlewares) == 0 {
		return wh
	}

	m := z.wrap(z.middlewares[len(z.middlewares)-1])(wh)
	for i := len(z.middlewares) - 2; i >= 0; i-- {
		f := z.wrap(z.middlewares[i])
		m = f(m)
	}
	return m
}

func (z *Inlineware) wrap(middleware interface{}) func(web.Handler) web.Handler {
	fn := func(wh web.Handler) web.Handler {
		return web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
			newFn := func(ww http.ResponseWriter, rr *http.Request) {
				wh.ServeHTTPC(c, ww, rr)
			}

			var fn http.HandlerFunc

			switch mw := middleware.(type) {
			default:
				panic(fmt.Sprintf("unsupported middleware type: %T", mw))
			case func(http.Handler) http.Handler:
				fn = mw(http.HandlerFunc(newFn)).ServeHTTP
			case func(*web.C, http.Handler) http.Handler:
				fn = mw(&c, http.HandlerFunc(newFn)).ServeHTTP
			}

			fn(w, r)
		})
	}
	return fn
}
