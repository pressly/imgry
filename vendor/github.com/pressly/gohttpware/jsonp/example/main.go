// JSONP example using Goji framework.. but anything that accepts
// a http.Handler middleware chain will work
package main

import (
	"log"
	"net/http"

	"github.com/pressly/gohttpware/jsonp"
	"github.com/unrolled/render"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

func main() {
	mux := web.New()
	render := render.New(render.Options{})

	mux.Use(middleware.Logger)
	mux.Use(jsonp.Handle)

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := &SomeObj{"superman"}
		render.JSON(w, 200, data)
	})

	err := http.ListenAndServe(":4444", mux)
	if err != nil {
		log.Fatal(err)
	}
}

type SomeObj struct {
	Name string `json:"name"`
}
