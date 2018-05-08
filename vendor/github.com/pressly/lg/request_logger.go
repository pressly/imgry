package lg

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

// RequestLogger is a middleware for the github.com/sirupsen/logrus to log requests.
// It is equipt to handle recovery in case of panics and record the stack trace
// with a panic log-level.
func RequestLogger(logger *logrus.Logger) func(next http.Handler) http.Handler {
	httpLogger := &HTTPLogger{logger}

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

type HTTPLogger struct {
	Logger *logrus.Logger
}

func (l *HTTPLogger) NewLogEntry(r *http.Request) *HTTPLoggerEntry {
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

	logFields["uri"] = fmt.Sprintf("%s://%s%s", scheme, host, r.RequestURI)

	entry.Logger = entry.Logger.WithFields(logFields)

	entry.Logger.Infoln("request started")

	return entry
}

type HTTPLoggerEntry struct {
	Logger logrus.FieldLogger // field logger interface, created by RequestLogger
	Level  *logrus.Level      // intended log level to write when request finishes
}

func (l *HTTPLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"resp_status": status, "resp_bytes_length": bytes,
		"resp_elasped_ms": float64(elapsed.Nanoseconds()) / 1000000.0,
	})

	if l.Level == nil {
		l.Logger.Infoln("request complete")
	} else {
		switch *l.Level {
		case logrus.DebugLevel:
			l.Logger.Debugln("request complete")
		case logrus.InfoLevel:
			l.Logger.Infoln("request complete")
		case logrus.WarnLevel:
			l.Logger.Warnln("request complete")
		case logrus.ErrorLevel:
			l.Logger.Errorln("request complete")
		case logrus.FatalLevel:
			l.Logger.Fatalln("request complete")
		case logrus.PanicLevel:
			l.Logger.Errorln("request complete")
		}
	}
}

func (l *HTTPLoggerEntry) Panic(rec interface{}, stack []byte) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", rec),
	})
	panicLevel := logrus.PanicLevel
	l.Level = &panicLevel
}

// PrintPanics is a development middleware that preempts the request logger
// and prints a panic message and stack trace to stdout.
func PrintPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("\nPANIC: %+v\n", rec)
				fmt.Printf("%s", debug.Stack())
				fmt.Printf("\nPANIC: %+v\n", rec)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
