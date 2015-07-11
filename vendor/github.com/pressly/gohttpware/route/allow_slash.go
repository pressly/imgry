package route

import (
	"net/http"
	"strings"
)

// Removes all trailing slashes if it exists.
// So /scripts/ == /scripts but /scripts// == /scripts
func AllowSlash(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > 1 {
			r.URL.Path = strings.TrimRight(r.URL.Path, "/")
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
