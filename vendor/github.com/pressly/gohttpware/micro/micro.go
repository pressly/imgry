// Microservice Web Handler
package micro

import (
	"net/http"

	"github.com/unrolled/render"
	"github.com/zenazn/goji/web"
)

type WebHandler struct {
	mux    *web.Mux
	render *render.Render
}

var _ web.Handler = (*WebHandler)(nil)

func NewWebHandler() *WebHandler {
	h := &WebHandler{
		mux:    web.New(),
		render: render.New(render.Options{}),
	}
	return h
}

func (h *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *WebHandler) ServeHTTPC(c web.C, w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTPC(c, w, r)
}

func (h *WebHandler) Data(w http.ResponseWriter, status int, v []byte) {
	// h.render.Data(w, status, v)
	w.WriteHeader(status)
	w.Write(v)
}

func (h *WebHandler) HTML(w http.ResponseWriter, status int, name string, binding interface{}, htmlOpt ...render.HTMLOptions) {
	h.render.HTML(w, status, name, binding, htmlOpt...)
}

func (h *WebHandler) JSON(w http.ResponseWriter, status int, v interface{}) {
	h.render.JSON(w, status, v)
}

func (h *WebHandler) XML(w http.ResponseWriter, status int, v interface{}) {
	h.render.XML(w, status, v)
}
