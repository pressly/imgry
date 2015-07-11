package auth

import (
	"fmt"
	"net/http"
)

type AuthFunc func(*http.Request, []string) bool

func Wrap(f AuthFunc, realm string, secrets ...string) func(http.Handler) http.Handler {
	h := func(h http.Handler) http.Handler {
		hn := func(w http.ResponseWriter, r *http.Request) {
			if !f(r, secrets) {
				Unauthorized(realm, w)
				return
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hn)
	}
	return h
}

func Unauthorized(realm string, w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Please authenticate with the proper details\n"))
}
