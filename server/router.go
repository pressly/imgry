package server

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pressly/cji"
	"github.com/pressly/consistentrd"
	"github.com/pressly/imgry"
	"github.com/unrolled/render"

	"github.com/pressly/gohttpware/heartbeat"
	"github.com/rcrowley/go-metrics"

	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

func NewRouter() http.Handler {
	conrd, err := consistentrd.New(app.Config.Cluster.LocalNode, app.Config.Cluster.Nodes)
	if err != nil {
		panic(err)
	}

	r := web.New()

	r.Use(middleware.EnvInit)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger)
	r.Use(heartbeat.Route("/ping"))

	r.Get("/", cji.Use(trackRoute("root")).On(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("."))
	}))

	r.Get("/fetch", cji.Use(trackRoute("bucketV0GetItem")).On(BucketV0FetchItem)) // Deprecated: for Pressilla v2 apps

	r.Get("/info", cji.Use(trackRoute("imageInfo")).On(GetImageInfo))

	r.Get("/:bucket", cji.Use(conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem")).On(BucketGetIndex))
	r.Post("/:bucket", BucketImageUpload)

	r.Get("/:bucket/fetch", cji.Use(conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem")).On(BucketFetchItem))

	// DEPRECATED
	r.Get("/:bucket//fetch", cji.Use(conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem")).On(BucketFetchItem))

	r.Get("/:bucket/add", cji.Use(trackRoute("bucketAddItems")).On(BucketAddItems))
	r.Get("/:bucket/:key", cji.Use(conrd.Route()).On(BucketGetItem))
	r.Delete("/:bucket/:key", cji.Use(conrd.Route()).On(BucketDeleteItem))

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
		lg.Info(fmt.Sprintf("Started %s %s", r.Method, u))

		lw := &loggedResponseWriter{w, -1}
		next.ServeHTTP(lw, r)

		lg.Info(fmt.Sprintf(
			"Completed (%s): %v %s in %v\n",
			u,
			lw.Status(),
			http.StatusText(lw.Status()),
			time.Since(start),
		))
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
