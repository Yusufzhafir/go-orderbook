package router

import (
	"log"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	n      int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.n += n
	return n, err
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %dB %s", r.Method, r.URL.Path, sw.status, sw.n, time.Since(start))
	})
}

func bindTicker(serverRouter *http.ServeMux) {
	serverRouter.Handle("GET /api/v1/ticker", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-list", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-queue/{price}", logging(http.HandlerFunc(defaultHandler)))
}
func bindOrder(serverRouter *http.ServeMux) {
	serverRouter.Handle("POST /api/v1/order/add", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("PUT /api/v1/order/modify", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("DELETE /api/v1/order/cancel", logging(http.HandlerFunc(defaultHandler)))
}
func bindUser(serverRouter *http.ServeMux) {
	serverRouter.Handle("GET /api/v1/user/{id}", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/user/{id}/transactions", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/user/{id}/portfolio", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("POST /api/v1/user/{id}/money", logging(http.HandlerFunc(defaultHandler)))
}

type BindRouterOpts struct {
	ServerRouter *http.ServeMux
}

func BindRouter(opts BindRouterOpts) {
	bindOrder(opts.ServerRouter)
	bindUser(opts.ServerRouter)
	bindTicker(opts.ServerRouter)
}
