package server

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"time"

	"github.com/goware/lg"
	"github.com/pressly/cji"
	"github.com/pressly/consistentrd"
	"github.com/pressly/gohttpware/heartbeat"
	"github.com/pressly/imgry"
	"github.com/rcrowley/go-metrics"
	"github.com/unrolled/render"
	"github.com/zenazn/goji/web/middleware"
)

func NewRouter() http.Handler {
	conrd, err := consistentrd.New(app.Config.Cluster.LocalNode, app.Config.Cluster.Nodes)
	if err != nil {
		panic(err)
	}

	r := cji.NewRouter()

	r.Use(middleware.EnvInit)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger)
	r.Use(heartbeat.Route("/ping"))

	r.Mount("/debug", middleware.NoCache, Profiler())

	r.Get("/", trackRoute("root"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("."))
	})

	r.Get("/fetch", trackRoute("bucketV0GetItem"), BucketV0FetchItem) // Deprecated: for Pressilla v2 apps

	r.Get("/info", trackRoute("imageInfo"), GetImageInfo)

	r.Get("/:bucket", conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem"), BucketGetIndex)
	r.Post("/:bucket", BucketImageUpload)

	r.Get("/:bucket/fetch", conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem"), BucketFetchItem)

	// DEPRECATED
	r.Get("/:bucket//fetch", conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem"), BucketFetchItem)

	r.Get("/:bucket/add", trackRoute("bucketAddItems"), BucketAddItems)
	r.Get("/:bucket/:key", conrd.Route(), BucketGetItem)
	r.Delete("/:bucket/:key", conrd.Route(), BucketDeleteItem)

	return r
}

func RequestLogger(next http.Handler) http.Handler {
	reqCounter := metrics.GetOrRegisterCounter("route.TotalNumRequests", nil)

	h := func(w http.ResponseWriter, r *http.Request) {
		reqCounter.Inc(1)

		u, err := url.QueryUnescape(r.URL.RequestURI())
		if err != nil {
			lg.Error(err.Error())
		}

		start := time.Now()
		lg.Infof("Started %s %s", r.Method, u)

		lw := &loggedResponseWriter{w, -1}
		next.ServeHTTP(lw, r)

		lg.Infof("Completed (%s): %v %s in %v",
			u,
			lw.Status(),
			http.StatusText(lw.Status()),
			time.Since(start),
		)
	}
	return http.HandlerFunc(h)
}

type Responder struct {
	*render.Render
}

func NewResponder() *Responder {
	return &Responder{render.New(render.Options{})}
}

func (r *Responder) ImageError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		r.Data(w, status, []byte{})
		return
	}

	r.cacheErrors(w, err)
	w.Header().Set("X-Err", err.Error())
	r.Data(w, status, []byte{})
}

func (r *Responder) ApiError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		r.JSON(w, status, []byte{})
		return
	}

	r.cacheErrors(w, err)
	r.JSON(w, status, map[string]interface{}{"error": err.Error()})
}

func (r *Responder) cacheErrors(w http.ResponseWriter, err error) {
	switch err {
	case imgry.ErrInvalidImageData, ErrInvalidURL:
		// For invalid inputs, we tell the surrogate to cache the
		// error for a small amount of time.
		w.Header().Set("Surrogate-Control", "max-age=300") // 5 minutes
	default:
	}
}

func Profiler() http.Handler {
	r := cji.NewRouter()
	r.Handle("/vars", expVars)
	r.Handle("/pprof/", pprof.Index)
	r.Handle("/pprof/cmdline", pprof.Cmdline)
	r.Handle("/pprof/profile", pprof.Profile)
	r.Handle("/pprof/symbol", pprof.Symbol)
	r.Handle("/pprof/block", pprof.Handler("block").ServeHTTP)
	r.Handle("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	r.Handle("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	r.Handle("/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	return r
}

// Replicated from expvar.go as not public.
func expVars(w http.ResponseWriter, r *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
