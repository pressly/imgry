package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

func BasicAuth(user, pass string) func(http.Handler) http.Handler {
	return Wrap(basicAuth, "Restricted", user, pass)
}

func basicAuth(r *http.Request, secrets []string) bool {
	user, pass := secrets[0], secrets[1]
	authString := fmt.Sprintf("%s:%s", user, pass)
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	pass, err := decodePlainAuth(auth[6:])
	if err != nil || pass != authString {
		return false
	}
	return true
}

// This function decodes the given string.
// Here is where we would put any decryption if required.
func decodePlainAuth(auth string) (string, error) {
	pass, err := base64.StdEncoding.DecodeString(auth)
	return string(pass), err
}
