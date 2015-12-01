package server

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/goware/go-metrics"
	"github.com/pressly/chi"
	"github.com/tobi/airbrake-go"
	"golang.org/x/net/context"
)

// Set the parent context in the middleware chain to something else. Useful
// in the instance of having a global server context to signal all requests.
func ParentContext(parent context.Context) func(next chi.Handler) chi.Handler {
	return func(next chi.Handler) chi.Handler {
		fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			pctx := context.WithValue(parent, chi.URLParamsCtxKey, chi.URLParams(ctx))
			next.ServeHTTPC(pctx, w, r)
		}
		return chi.HandlerFunc(fn)
	}
}

// Airbrake recoverer middleware to capture and report any panics to
// airbrake.io.
func AirbrakeRecoverer(apiKey string) func(chi.Handler) chi.Handler {
	airbrake.ApiKey = apiKey

	return func(next chi.Handler) chi.Handler {
		return chi.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if apiKey != "" {
				defer airbrake.CapturePanic(r)
			}
			next.ServeHTTPC(ctx, w, r)
		})
	}
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
		errRoute := fmt.Sprintf("%s-err", route)

		handler := func(w http.ResponseWriter, r *http.Request) {
			defer metrics.MeasureSince([]string{route}, time.Now())

			lw := &wrappedResponseWriter{w, -1}
			next.ServeHTTP(lw, r)

			if lw.Status() >= 400 {
				metrics.IncrCounter([]string{errRoute}, 1)
			}
		}
		return http.HandlerFunc(handler)
	}
}
