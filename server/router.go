package server

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/pressly/cji"
	"github.com/pressly/consistentrd"
	"github.com/pressly/gohttpware/heartbeat"
	"github.com/pressly/imgry"
	"github.com/tobi/airbrake-go"
	"github.com/unrolled/render"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"golang.org/x/net/context"
)

func NewRouter() http.Handler {
	conrd, err := consistentrd.New(app.Config.Cluster.LocalNode, app.Config.Cluster.Nodes)
	if err != nil {
		panic(err)
	}

	r := cji.NewRouter()

	r.Use(middleware.EnvInit)
	r.Use(CtxInit(app.Config.Limits.RequestTimeout))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// r.Use(RequestLogger)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	if app.Config.Airbrake.ApiKey != "" {
		r.Use(AirbrakeRecoverer(app.Config.Airbrake.ApiKey))
	}
	r.Use(heartbeat.Route("/ping"))

	// if app.Config. // .Profiler { ... }
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

func CtxInit(timeout time.Duration) func(c *web.C, next http.Handler) http.Handler {
	return func(c *web.C, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var ctx context.Context
			var cancel context.CancelFunc

			if timeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), timeout)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			go func() {
				select {
				case <-ctx.Done():
					return
				case <-w.(http.CloseNotifier).CloseNotify():
					cancel()
					return
				}
			}()

			c.Env["ctx"] = ctx
			next.ServeHTTP(w, r)
		})
	}
}

// Airbrake recoverer middleware to capture and report any panics to
// airbrake.io.
func AirbrakeRecoverer(apiKey string) func(http.Handler) http.Handler {
	airbrake.ApiKey = apiKey
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if apiKey != "" {
				defer airbrake.CapturePanic(r)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
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
