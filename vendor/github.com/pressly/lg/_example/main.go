package main

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pressly/lg"
	"github.com/sirupsen/logrus"
)

func main() {

	// Setup the logger
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	lg.RedirectStdlogOutput(logger)
	lg.DefaultLogger = logger

	lg.Infoln("Welcome")

	serverCtx := context.Background()
	serverCtx = lg.WithLoggerContext(serverCtx, logger)
	lg.Log(serverCtx).Infof("Booting up server, %s", "v1.0")

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(lg.RequestLogger(logger))
	// r.Use(lg.PrintPanics)

	r.Use(Counter)

	r.Get("/", Index)
	r.Route("/articles", func(r chi.Router) {
		r.Use(ArticleCtx)
		r.With(PaginateCtx).Get("/", List)
		r.Get("/search", Search)
	})
	r.Get("/stdlog", Stdlog)
	r.Get("/fatal", Fatal)
	r.Get("/panic", Panic)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			lg.Log(serverCtx).Infof("tick")
		}
	}()

	service := chi.ServerBaseContext(serverCtx, r)
	http.ListenAndServe(":3333", service)
}

var counter = uint64(0)

func Counter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&counter, 1)
		lg.SetEntryField(r.Context(), "count", counter)
		next.ServeHTTP(w, r)
	})
}

func ArticleCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lg.RequestLog(r).Warnf("inside ArticleCtx middleware")
		lg.SetRequestEntryField(r, "article", 123)
		next.ServeHTTP(w, r)
	})
}

func PaginateCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lg.RequestLog(r).Warnf("inside PaginateCtx middleware")
		lg.SetEntryField(r.Context(), "paginate", true)
		next.ServeHTTP(w, r)
	})
}

func Index(w http.ResponseWriter, r *http.Request) {
	log := lg.Log(r.Context())
	log.Info("index")
	w.Write([]byte("index"))
}

func List(w http.ResponseWriter, r *http.Request) {
	log := lg.RequestLog(r)
	log.Info("articles list")
	w.Write([]byte("list"))
}

func Search(w http.ResponseWriter, r *http.Request) {
	log := lg.RequestLog(r)
	log.Info("articles search")
	w.Write([]byte("search"))
}

func Stdlog(w http.ResponseWriter, r *http.Request) {
	log.Println("logging from stdlib log to logrus")
	w.Write([]byte("piping from the stdlib log pkg"))
}

func Fatal(w http.ResponseWriter, r *http.Request) {
	lg.RequestLog(r).Fatal("boom")
}

func Panic(w http.ResponseWriter, r *http.Request) {
	panic("oh no")
}
