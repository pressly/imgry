package server

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	raven "github.com/getsentry/raven-go"
	"github.com/sirupsen/logrus"
)

func Logger(level logrus.Level, msg string) {
	if level <= logrus.WarnLevel {
		packet := raven.NewPacket(
			msg,
			raven.NewException(
				fmt.Errorf("API alert: %s", msg),
				raven.NewStacktrace(2, 3, nil),
			),
		)
		switch level {
		case logrus.FatalLevel:
			packet.Level = raven.FATAL

		case logrus.ErrorLevel:
			packet.Level = raven.ERROR

		case logrus.WarnLevel:
			packet.Level = raven.WARNING
		}

		raven.Capture(packet, nil)
	}
}

// CapturePanic middleware reports panics to sentry.
func CapturePanic() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rval := recover(); rval != nil {
					debug.PrintStack()
					rvalStr := fmt.Sprint(rval)
					packet := raven.NewPacket(rvalStr, raven.NewException(errors.New(rvalStr), raven.NewStacktrace(2, 3, nil)), raven.NewHttp(r))
					raven.Capture(packet, nil)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
