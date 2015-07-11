package jsonp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func Handle(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		callback := r.URL.Query().Get("callback")
		if callback == "" {
			callback = r.URL.Query().Get("jsonp")
		}
		if callback == "" {
			h.ServeHTTP(w, r)
			return
		}

		wb := NewResponseBuffer(w)
		h.ServeHTTP(wb, r)

		if strings.Index(wb.Header().Get("Content-Type"), "/json") >= 0 {
			status := wb.Status
			data := wb.Body.Bytes()
			wb.Body.Reset()

			resp := &jsonpResponse{
				Meta: map[string]interface{}{"status": status},
				Data: data,
			}
			for k, v := range wb.Header() {
				resp.Meta[strings.ToLower(k)] = v[0]
			}
			resp.Meta["content-length"] = len(data)

			body, err := json.Marshal(resp)
			if err != nil {
				panic(err.Error())
			}

			wb.Body.Write([]byte(callback + "("))
			wb.Body.Write(body)
			wb.Body.Write([]byte(")"))

			wb.Header().Set("Content-Type", "application/javascript")
			wb.Header().Set("Content-Length", strconv.Itoa(wb.Body.Len()))
		}

		wb.Flush()
	}
	return http.HandlerFunc(fn)
}

type jsonpResponse struct {
	Meta map[string]interface{}
	Data interface{}
}

func (j *jsonpResponse) MarshalJSON() ([]byte, error) {
	meta, err := json.Marshal(j.Meta)
	if err != nil {
		return nil, err
	}
	b := fmt.Sprintf("{\"meta\":%s,\"data\":%s}", meta, j.Data)
	return []byte(b), nil
}

type responseBuffer struct {
	Response http.ResponseWriter // the actual ResponseWriter to flush to
	Status   int                 // the HTTP response code from WriteHeader
	Body     *bytes.Buffer       // the response content body
	Flushed  bool
}

func NewResponseBuffer(w http.ResponseWriter) *responseBuffer {
	return &responseBuffer{
		Response: w, Status: 200, Body: &bytes.Buffer{},
	}
}

func (w *responseBuffer) Header() http.Header {
	return w.Response.Header() // use the actual response header
}

func (w *responseBuffer) Write(buf []byte) (int, error) {
	w.Body.Write(buf)
	return len(buf), nil
}

func (w *responseBuffer) WriteHeader(status int) {
	w.Status = status
}

func (w *responseBuffer) Flush() {
	if w.Flushed {
		return
	}
	w.Response.WriteHeader(w.Status)
	if w.Body.Len() > 0 {
		_, err := w.Response.Write(w.Body.Bytes())
		if err != nil {
			panic(err)
		}
		w.Body.Reset()
	}
	w.Flushed = true
}
