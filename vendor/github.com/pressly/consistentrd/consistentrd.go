package consistentrd

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stathat.com/c/consistent"
)

type consistentRoad struct {
	localNode  *node
	allNodes   map[string]*node
	hashRing   *consistent.Consistent
	httpClient *http.Client
}

type node struct {
	hashId   string
	url      *url.URL
	seqFails uint32 // sequential failures
}

type RouteFn func(*url.URL) string

const (
	XConrdHeader      = "X-Conrd"
	NumRingReplicas   = 20
	NodeConnTimeout   = time.Duration(time.Millisecond * 50)
	NodeConnKeepAlive = time.Duration(time.Second * 300)
	NodeReqTimeout    = time.Duration(time.Second * 30)
	NodeFailLimit     = 3
	NodeWithdrawTime  = time.Duration(time.Second * 60)
)

func New(localNode string, allNodes []string) (conrd *consistentRoad, err error) {
	conrd = &consistentRoad{allNodes: make(map[string]*node)}

	conrd.hashRing = consistent.New()
	conrd.hashRing.NumberOfReplicas = NumRingReplicas

	conrd.localNode, err = newNode(localNode)
	if err != nil {
		return nil, err
	}
	conrd.hashRing.Add(conrd.localNode.hashId)

	for _, s := range allNodes {
		n, err := newNode(s)
		if err != nil {
			return nil, err
		}
		conrd.hashRing.Add(n.hashId)
		conrd.allNodes[n.hashId] = n
	}

	conrd.httpClient = &http.Client{
		Timeout: NodeReqTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   NodeConnTimeout,
				KeepAlive: NodeConnKeepAlive,
			}).Dial,
			TLSHandshakeTimeout: NodeReqTimeout,
			MaxIdleConnsPerHost: 30,
		},
	}

	return conrd, nil
}

func newNode(urlStr string) (*node, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	u.Path = ""
	return &node{url: u, hashId: urlStr}, nil
}

func (c *consistentRoad) Route() func(http.Handler) http.Handler {
	mw := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.proxyHandler(w, r, r.URL.Path, h)
		}
		return http.HandlerFunc(fn)
	}
	return mw
}

func (c *consistentRoad) RouteWithParams(keys ...string) func(http.Handler) http.Handler {
	mw := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			routeKey := r.URL.Path
			routeQuery := url.Values{}
			for _, key := range keys {
				param := r.URL.Query().Get(key)
				if param != "" {
					routeQuery.Add(key, param)
				}
			}
			routeKey = routeKey + "?" + routeQuery.Encode()
			c.proxyHandler(w, r, routeKey, h)
		}
		return http.HandlerFunc(fn)
	}
	return mw
}

func (c *consistentRoad) RoutePathFn(routeFn RouteFn) func(http.Handler) http.Handler {
	mw := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.proxyHandler(w, r, routeFn(r.URL), h)
		}
		return http.HandlerFunc(fn)
	}
	return mw
}

func (c *consistentRoad) proxyHandler(w http.ResponseWriter, r *http.Request, routeKey string, next http.Handler) {

	if r.Header.Get(XConrdHeader) != "" || routeKey == "" {
		next.ServeHTTP(w, r)
		return
	}

	// Find server on ring
	routeHashId, err := c.hashRing.Get(routeKey)
	if err != nil {
		log.Println("Conrd hashRing err:", err)
		next.ServeHTTP(w, r)
		return
	}
	routeNode, ok := c.allNodes[routeHashId]
	if !ok {
		next.ServeHTTP(w, r)
		return
	}

	if routeNode.seqFails >= NodeFailLimit {
		log.Printf("Conrd withdrawing node %s for failing %d sequential times", routeNode.hashId, routeNode.seqFails)
		c.hashRing.Remove(routeNode.hashId)
		routeNode.seqFails = 0

		// Add the node back to the ring at some point in the future
		go func() {
			time.Sleep(NodeWithdrawTime)
			log.Printf("Conrd added node %s to ring after %v", routeNode.hashId, NodeWithdrawTime)
			c.hashRing.Add(routeNode.hashId)
		}()

		// withdraw the faulty node from the ring for a period of time
		// and get the job done ourselves
		next.ServeHTTP(w, r)
		return
	}

	// Stop here if the request is intended for us
	if c.localNode.hashId == routeHashId {
		next.ServeHTTP(w, r)
		return
	}

	// Prepare request
	proxyUrl := routeNode.url.String() + r.URL.Path + "?" + r.URL.RawQuery
	proxyReq, _ := http.NewRequest(r.Method, proxyUrl, r.Body)
	proxyReq.Header.Set(XConrdHeader, "Yes")

	// Do request
	log.Println("Conrd http proxy to", proxyUrl)
	proxyRes, err := c.httpClient.Do(proxyReq)
	if err != nil {
		routeNode.seqFails += 1
		log.Println("Conrd proxy err:", err)
		next.ServeHTTP(w, r)
		return
	}

	// Read the data
	data, err := ioutil.ReadAll(proxyRes.Body)
	if err != nil {
		routeNode.seqFails += 1
		log.Println("Conrd proxy read err", err)
		next.ServeHTTP(w, r)
		return
	}

	routeNode.seqFails = 0 // clear

	// Proxy all of the headers
	for k, v := range proxyRes.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}
	w.WriteHeader(proxyRes.StatusCode)
	w.Write(data)
}
