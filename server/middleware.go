package server

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/pressly/chi"
	"github.com/pressly/go-metrics"
	"github.com/tobi/airbrake-go"
)

// func CtxInit(ctx context.Context) func(c *web.C, next http.Handler) http.Handler {
// 	return func(c *web.C, next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			c.Env["ctx"] = ctx
// 			next.ServeHTTP(w, r)
// 		})
// 	}
// }

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
	r := chi.NewRouter()
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

type wrappedResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *wrappedResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (l *wrappedResponseWriter) Status() int {
	return l.status
}

func trackRoute(metricID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		route := fmt.Sprintf("route.%s", metricID)
		routeTimer := metrics.GetOrRegisterTimer(route, nil)
		errCounter := metrics.GetOrRegisterCounter(fmt.Sprintf("%s-err", route), nil)

		handler := func(w http.ResponseWriter, r *http.Request) {
			reqStart := time.Now()

			lw := &wrappedResponseWriter{w, -1}
			next.ServeHTTP(lw, r)

			routeTimer.UpdateSince(reqStart)
			if lw.Status() >= 400 {
				errCounter.Inc(1)
			}
		}
		return http.HandlerFunc(handler)
	}
}
