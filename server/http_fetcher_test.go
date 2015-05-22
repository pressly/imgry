package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpFetcher(t *testing.T) {
	// Testing server that responds with request URI.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.RequestURI))
	}))
	defer srv.Close()

	hf := NewHttpFetcher()

	// Fetch hundred different responses.
	for i := 0; i < 100; i++ {
		go func(i int) {
			uri := fmt.Sprintf("/%d", i)

			resp, err := hf.client().Get(srv.URL + uri)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != uri {
				t.Errorf(`expected "%s", got "%s"`, uri, got)
			}
		}(i)
	}
}
