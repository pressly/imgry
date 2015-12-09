package httpcoala

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	var numRequests = 30

	var hits uint32
	var expectedStatus int = 201
	var expectedBody = []byte("hi")

	app := func(w http.ResponseWriter, r *http.Request) {
		// log.Println("app handler..")

		atomic.AddUint32(&hits, 1)

		hitsNow := atomic.LoadUint32(&hits)
		if hitsNow > 1 {
			// panic("uh oh")
		}

		// time.Sleep(100 * time.Millisecond) // slow handler
		w.Header().Set("X-Httpjoin", "test")
		w.WriteHeader(expectedStatus)
		w.Write(expectedBody)
	}

	var count uint32
	counter := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddUint32(&count, 1)
			next.ServeHTTP(w, r)
			atomic.AddUint32(&count, ^uint32(0))
			// log.Println("COUNT:", atomic.LoadUint32(&count))
		})
	}

	recoverer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Println("recovered panicing request:", r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}

	ts := httptest.NewServer(counter(recoverer(Route("GET")(http.HandlerFunc(app)))))
	defer ts.Close()

	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL)
			if err != nil {
				t.Fatal(err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// log.Println("got resp:", resp, "len:", len(body), "body:", string(body))

			if string(body) != string(expectedBody) {
				t.Error("expecting response body:", string(expectedBody))
			}

			if resp.StatusCode != expectedStatus {
				t.Error("expecting response status:", expectedStatus)
			}

			if resp.Header.Get("X-Httpjoin") != "test" {
				t.Error("expecting x-httpjoin test header")
			}

		}()
	}

	wg.Wait()

	totalHits := atomic.LoadUint32(&hits)
	// if totalHits > 1 {
	// 	t.Error("handler was hit more than once. hits:", totalHits)
	// }
	log.Println("total hits:", totalHits)

	finalCount := atomic.LoadUint32(&count)
	if finalCount > 0 {
		t.Error("queue count was expected to be empty, but count:", finalCount)
	}
	log.Println("final count:", finalCount)
}

func TestStress(t *testing.T) {

	mockData := make([]byte, 1024*1024*20)
	rand.Read(mockData)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(Route("HEAD", "GET")) // or, Route("*")

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 2) // expensive op
		w.WriteHeader(200)
		w.Write(mockData)
	})

	go http.ListenAndServe(":1111", r)

	time.Sleep(time.Second * 1)

	test := func(i int) error {
		res, err := http.Get(fmt.Sprintf("http://127.0.0.1:1111/?_=%d", i))
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			return errors.New("Expecting 200 OK.")
		}

		return err
	}

	var wg sync.WaitGroup

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(i int) {
			if err := test(i); err != nil {
				t.Fatal("test: ", err)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}
