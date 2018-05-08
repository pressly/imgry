package lg

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

// SanitizingRequestLogger is a middleware for the github.com/sirupsen/logrus to log requests.
// It is equipt to handle recovery in case of panics and record the stack trace
// with a panic log-level.
// It's second parameter is a map[string]string of replacements for parameters to be sanitized
// before logging
// Example:
//	map[string]string{
//		"token": "[redacted]",
//		"session": "removed-sesion-id",
//	}
func SanitizingRequestLogger(logger *logrus.Logger, rules map[string]string) func(next http.Handler) http.Handler {
	httpLogger := &SanitizingHTTPLogger{logger, rules}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := httpLogger.NewLogEntry(r)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				// Recover and record stack traces in case of a panic
				if rec := recover(); rec != nil {
					entry.Panic(rec, debug.Stack())
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				// Log the entry, the request is complete.
				entry.Write(ww.Status(), ww.BytesWritten(), t2.Sub(t1))
			}()

			r = r.WithContext(WithLogEntry(r.Context(), entry))
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

type SanitizingHTTPLogger struct {
	Logger *logrus.Logger
	Rules  map[string]string
}

func (l *SanitizingHTTPLogger) NewLogEntry(r *http.Request) *HTTPLoggerEntry {
	entry := &HTTPLoggerEntry{Logger: logrus.NewEntry(l.Logger)}
	logFields := logrus.Fields{}

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields["req_id"] = reqID
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host

	logFields["http_scheme"] = scheme
	logFields["http_proto"] = r.Proto
	logFields["http_method"] = r.Method

	logFields["remote_addr"] = r.RemoteAddr
	logFields["user_agent"] = r.UserAgent()

	if val := r.Header.Get("X-Forwarded-For"); val != "" {
		logFields["X-Forwarded-For"] = val
	}
	if val := r.Header.Get("X-Forwarded-Host"); val != "" {
		logFields["X-Forwarded-Host"] = val
		host = val
	}
	if val := r.Header.Get("X-Forwarded-Scheme"); val != "" {
		logFields["X-Forwarded-Scheme"] = val
		scheme = val
	}

	if u, err := url.ParseRequestURI(r.RequestURI); err == nil {
		q := u.Query()

		// sanitize
		for key, val := range q {
			if rep, ok := l.Rules[key]; ok {
				for i := 0; i < len(val); i++ {
					val[i] = rep
				}
				q[key] = val
			}
		}
		u.RawQuery = q.Encode()

		logFields["uri"] = fmt.Sprintf("%s://%s%s", scheme, host, u.RequestURI())
	}

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Infoln("request started")

	return entry
}
