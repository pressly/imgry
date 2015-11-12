HTTP Coala
==========

Just a little bit of performance enhancing middleware -- HTTP Coala (aka Coalescer).

HTTP Coala is a middleware handler that routes multiple requests for the same URI
(and routed methods) to be processed as a single request. I don't recommend it
for every web service or handler chain, but for the computationally expensive
handlers that always yield the same response, HTTP Coala will give you a speed boost.

It's common among HTTP reverse proxy cache servers such as nginx,
Squid or Varnish - they all call it something else but works similarly.

* https://www.varnish-cache.org/docs/3.0/tutorial/handling_misbehaving_servers.html
* http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_cache_lock
* http://wiki.squid-cache.org/Features/CollapsedForwarding


# Usage

Example with goji

```go
// from _example/main.go ....
func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(httpcoala.Route("HEAD", "GET")) // or, Route("*")
	// r.Use(otherMiddleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // expensive op
		w.WriteHeader(200)
		w.Write([]byte("hi"))
	})

	http.ListenAndServe(":1111", r)
}
```

as well, look at httpcoala_test.go


# Benchmarks

For a naive benchmark on my local Macbook pro, I used wrk with -c 100 and -t 100
on a web service that downloads an image and returns a resized version.

```
Without httpcoala middleware, Requests/sec:   7081.09
   With httpcoala middleware, Requests/sec:  18373.87
```

# TODO

* Allow a request key to be passed that determines when to coalesce requests.
  It would allow for more control such as grouping by a query param or header.

# License

Brought to you by the Pressly Go team - www.pressly.com / https://github.com/pressly

MIT License

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
