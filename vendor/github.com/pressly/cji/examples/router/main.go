package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pressly/cji"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

var DEBUG = true

func NewRouter() cji.Router {
	r := cji.NewRouter()
	if DEBUG {
		return NewDebugRouter(r)
	} else {
		return r
	}
}

func main() {
	r := NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Get("/", apiIndex)

	r.Mount("/accounts", sup, accountsRouter())

	if d, ok := r.(*DebugMux); ok {
		d.PrintRoutes()
	}

	// http.ListenAndServe(":3333", r)
}

func apiIndex(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("root"))
}

//--

func accountsRouter() cji.Router {
	// r := cji.NewRouter()
	r := NewRouter()
	r.Use(sup1)
	r.Get("/", listAccounts)
	r.Get("/hi", hiAccounts)

	// r.Post("/", createAccount)

	r.Group(func(r cji.Router) {
		r.Use(sup2)

		r.Get("/hi2", func(c web.C, w http.ResponseWriter, r *http.Request) {
			log.Println("hi2..", c)
			w.Write([]byte("woot"))
		})
	})

	// 2nd param is optional..
	r.Route("/:accountID", func(r cji.Router) {
		r.Use(accountCtx)
		r.Get("/", getAccount)
	})

	return r
}

func sup1(c *web.C, h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		c.Env["sup1"] = "sup1"
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(handler)
}

func sup2(c *web.C, h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		c.Env["sup2"] = "sup2"
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(handler)
}

func sup(c *web.C, h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		c.Env["sup"] = "sup"
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(handler)
}

func accountCtx(c *web.C, h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("accountCtx......", c)
		c.Env["account"] = "account 123"
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(handler)
}

func listAccounts(c web.C, w http.ResponseWriter, r *http.Request) {
	log.Println("list accounts", c)
	w.Write([]byte("list accounts"))
}

func hiAccounts(c web.C, w http.ResponseWriter, r *http.Request) {
	log.Println("hi accounts", c)
	w.Write([]byte("hi accounts"))
}

func createAccount(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("create account"))
}

func getAccount(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("get account --> ", c.Env["account"], c.URLParams["accountID"])))
}
