package lg

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

var (
	LoggerCtxKey   = &contextKey{"Logger"}
	LogEntryCtxKey = &contextKey{"LogEntry"}
)

func WithLoggerContext(parent context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(parent, LoggerCtxKey, logger)
}

func WithLogEntry(parent context.Context, logEntry *HTTPLoggerEntry) context.Context {
	return context.WithValue(parent, LogEntryCtxKey, logEntry)
}

func Log(ctx context.Context) logrus.FieldLogger {
	if entry, ok := ctx.Value(LogEntryCtxKey).(*HTTPLoggerEntry); ok {
		return entry.Logger
	}
	lgr, ok := ctx.Value(LoggerCtxKey).(*logrus.Logger)
	if !ok {
		panic("lg: logger backend has not been set on the context.")
	}
	return lgr
}

func RequestLog(r *http.Request) logrus.FieldLogger {
	return Log(r.Context())
}

func SetEntryField(ctx context.Context, key string, value interface{}) {
	if entry, ok := ctx.Value(LogEntryCtxKey).(*HTTPLoggerEntry); ok {
		entry.Logger = entry.Logger.WithField(key, value)
	}
}

func SetEntryFields(ctx context.Context, fields map[string]interface{}) {
	if entry, ok := ctx.Value(LogEntryCtxKey).(*HTTPLoggerEntry); ok {
		entry.Logger = entry.Logger.WithFields(fields)
	}
}

func SetRequestEntryField(r *http.Request, key string, value interface{}) {
	SetEntryField(r.Context(), key, value)
}

func SetRequestEntryFields(r *http.Request, fields map[string]interface{}) {
	SetEntryFields(r.Context(), fields)
}

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "lg context value " + k.name
}
