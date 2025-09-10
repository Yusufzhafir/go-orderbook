package router

import (
	"log"
	"net/http"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
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

// wrap your mux with cors(mux) when starting the server
// http.ListenAndServe(":8080", cors(logging(mux)))

func Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			// If you need cookies, echo origin & add Allow-Credentials (no "*")
			w.Header().Set("Access-Control-Allow-Origin", origin) // or "*" if no credentials
			w.Header().Set("Vary", "Origin")

			// Reflect requested headers/method for preflight robustness
			reqHdrs := r.Header.Get("Access-Control-Request-Headers")
			if reqHdrs == "" {
				reqHdrs = "Content-Type, Authorization"
			}
			w.Header().Set("Access-Control-Allow-Headers", reqHdrs)

			reqMethod := r.Header.Get("Access-Control-Request-Method")
			if reqMethod == "" {
				reqMethod = "GET, POST, PUT, DELETE, OPTIONS"
			}
			w.Header().Set("Access-Control-Allow-Methods", reqMethod)

			// If you need cookies across origins:
			// w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Cache preflight for a day (optional)
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Short-circuit preflight so it never hits your route table
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GET /api/v1/orderbook?symbol=TICKER-USD&depth=50 →
func bindTicker(serverRouter *http.ServeMux, orderUsecase *order.OrderUseCase) {
	serverRouter.Handle("GET /api/v1/ticker", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-list", logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := *orderUsecase
		data := uc.GetOrderInfos(r.Context())
		writeJSON(w, http.StatusOK, data)
	})))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-queue/{price}", logging(http.HandlerFunc(defaultHandler)))
}

func bindOrder(serverRouter *http.ServeMux, usecase *order.OrderUseCase) {
	newOrderRouter := NewOrderRouter(usecase)
	serverRouter.Handle("POST /api/v1/order/add", logging(http.HandlerFunc(newOrderRouter.Add)))
	serverRouter.Handle("PUT /api/v1/order/modify", logging(http.HandlerFunc(newOrderRouter.Modify)))
	serverRouter.Handle("DELETE /api/v1/order/cancel", logging(http.HandlerFunc(newOrderRouter.Cancel)))
}
func bindUser(serverRouter *http.ServeMux) {
	serverRouter.Handle("GET /api/v1/user/{id}", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/user/{id}/order-list", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/user/{id}/transactions", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/user/{id}/portfolio", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("POST /api/v1/user/{id}/money", logging(http.HandlerFunc(defaultHandler)))
}

type BindRouterOpts struct {
	ServerRouter *http.ServeMux
	OrderUseCase *order.OrderUseCase
}

func BindRouter(opts BindRouterOpts) {
	bindOrder(opts.ServerRouter, opts.OrderUseCase)
	bindUser(opts.ServerRouter)
	bindTicker(opts.ServerRouter, opts.OrderUseCase)
}
