package server

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/goware/urlx"
	"github.com/rcrowley/go-metrics"
)

var (
	DefaultHttpFetcherReqTimeout     = time.Second * 20
	DefaultHttpFetcherReqNumAttempts = 2
)

type HttpFetcher struct {
	Client *http.Client
	Transport *http.Transport

	ReqTimeout     time.Duration
	ReqNumAttempts int
	HostKeepAlive  time.Duration

	// TODO: lru cache of responses.. like a reverse cache.. including bad urls.. 404s, etc...
}

type HttpFetcherResponse struct {
	URL    *url.URL
	Status int
	Data   []byte
	Err    error
}

func NewHttpFetcher() *HttpFetcher {
	hf := &HttpFetcher{}
	hf.ReqTimeout = DefaultHttpFetcherReqTimeout
	hf.ReqNumAttempts = DefaultHttpFetcherReqNumAttempts
	hf.HostKeepAlive = 60 * time.Second
	return hf
}

func (hf HttpFetcher) client() *http.Client {
	if hf.Client != nil {
		return hf.Client
	}

	hf.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   hf.ReqTimeout,
			KeepAlive: hf.HostKeepAlive,
		}).Dial,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConnsPerHost: 2,
		DisableCompression:  true,
		DisableKeepAlives:   true,
		ResponseHeaderTimeout: hf.ReqTimeout,
	}

	hf.Client = &http.Client{
		Timeout: hf.ReqTimeout,
		Transport: hf.Transport,
	}

	// transport.CloseIdleConnections()

	return hf.Client
}

func (hf HttpFetcher) Get(url string) (*HttpFetcherResponse, error) {
	resps, err := hf.GetAll([]string{url})
	if err != nil {
		return nil, err
	}
	if len(resps) == 0 {
		return nil, errors.New("httpfetcher: no response")
	}
	resp := resps[0]
	if resp.Err != nil {
		return resp, resp.Err
	}
	return resp, nil
}

func (hf HttpFetcher) GetAll(urls []string) ([]*HttpFetcherResponse, error) {
	m := metrics.GetOrRegisterTimer("fn.FetchRemoteData", nil) // TODO: update metric name
	defer m.UpdateSince(time.Now())

	resps := make([]*HttpFetcherResponse, len(urls))

	var wg sync.WaitGroup
	wg.Add(len(urls))

	// TODO: add thruput here..

	for i, urlStr := range urls {
		resps[i] = &HttpFetcherResponse{}

		go func(resp *HttpFetcherResponse) {
			defer wg.Done()

			url, err := urlx.Parse(urlStr)
			if err != nil {
				resp.Err = err
				return
			}
			resp.URL = url

			lg.Info("Fetching %s", url.String())

			fetch, err := hf.client().Get(url.String())
			if err != nil {
				lg.Warning("Error fetching %s because %s", url.String(), err)
				resp.Err = err
				return
			}
			defer fetch.Body.Close()

			resp.Status = fetch.StatusCode

			body, err := ioutil.ReadAll(fetch.Body)
			if err != nil {
				resp.Err = err
				return
			}
			resp.Data = body
			resp.Err = nil

		}(resps[i])
	}

	wg.Wait()
	return resps, nil
}
