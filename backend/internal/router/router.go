package router

import (
	"log"
	"net/http"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/router/middleware"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/user"
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
func bindTicker(serverRouter *http.ServeMux, orderUsecase *order.OrderUseCase, tokenMaker *middleware.JWTMaker) {
	authmiddleware := middleware.AuthMiddleware(tokenMaker)
	serverRouter.Handle("GET /api/v1/ticker", authmiddleware(logging(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}", authmiddleware(logging(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-queue/{price}", authmiddleware(logging(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-list", authmiddleware(logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := *orderUsecase
		data := uc.GetOrderInfos(r.Context())
		writeJSON(w, http.StatusOK, data)
	}))))
}

func bindOrder(serverRouter *http.ServeMux, usecase *order.OrderUseCase, tokenMaker *middleware.JWTMaker) {
	authmiddleware := middleware.AuthMiddleware(tokenMaker)
	newOrderRouter := NewOrderRouter(usecase)
	serverRouter.Handle("POST /api/v1/order/add", logging(authmiddleware(http.HandlerFunc(newOrderRouter.Add))))
	serverRouter.Handle("PUT /api/v1/order/modify", logging(authmiddleware(http.HandlerFunc(newOrderRouter.Modify))))
	serverRouter.Handle("DELETE /api/v1/order/cancel", logging(authmiddleware(http.HandlerFunc(newOrderRouter.Cancel))))
}
func bindUser(serverRouter *http.ServeMux, tokenMaker *middleware.JWTMaker, userUseCase *user.UserUseCase) {
	authmiddleware := middleware.AuthMiddleware(tokenMaker)
	userRouter := NewUserRouter(userUseCase, tokenMaker)
	serverRouter.Handle("GET /api/v1/user/", logging(authmiddleware(http.HandlerFunc(userRouter.GetUser))))
	serverRouter.Handle("GET /api/v1/user/order-list", logging(authmiddleware(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("GET /api/v1/user/transactions", logging(authmiddleware(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("GET /api/v1/user/portfolio", logging(authmiddleware(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("POST /api/v1/user/money", logging(authmiddleware(http.HandlerFunc(defaultHandler))))
	serverRouter.Handle("POST /api/v1/user/register", logging(http.HandlerFunc(userRouter.RegisterUser)))
	serverRouter.Handle("POST /api/v1/user/login", logging(http.HandlerFunc(userRouter.LoginUser)))
}

type BindRouterOpts struct {
	ServerRouter *http.ServeMux
	OrderUseCase *order.OrderUseCase
	TokenMaker   *middleware.JWTMaker
	UserUseCase  *user.UserUseCase
}

func BindRouter(opts BindRouterOpts) {
	bindOrder(opts.ServerRouter, opts.OrderUseCase, opts.TokenMaker)
	bindUser(opts.ServerRouter, opts.TokenMaker, opts.UserUseCase)
	bindTicker(opts.ServerRouter, opts.OrderUseCase, opts.TokenMaker)

	//healthcheck
	opts.ServerRouter.Handle("GET /healthz", logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": 200,
			"health": "healthy",
		})
	})))
}
