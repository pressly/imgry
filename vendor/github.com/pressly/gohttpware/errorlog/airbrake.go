package errorlog

import (
	"errors"
	"net/http"

	"github.com/op/go-logging"
	"github.com/tobi/airbrake-go"
)

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

type AirbrakeBackend struct {
	ApiKey string
}

func NewAirbrakeBackend(apiKey string) *AirbrakeBackend {
	backend := &AirbrakeBackend{
		ApiKey: apiKey,
	}
	airbrake.ApiKey = backend.ApiKey
	return backend
}

func (b *AirbrakeBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	e := errors.New(rec.Formatted(calldepth + 1))
	return airbrake.Notify(e)
}
