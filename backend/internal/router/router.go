package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
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
	serverRouter.Handle("POST /api/v1/order/add", logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type AddOrderRequest struct {
			Side     model.Side      `json:"side"` // "BID" or "ASK" (or BUY/SELL depending on your model)
			Price    model.Price     `json:"price"`
			Quantity model.Quantity  `json:"quantity"`
			Type     model.OrderType `json:"type"` // e.g., LIMIT, MARKET, FOK, IOC, etc.
			// Optional: ClientOrderID string `json:"clientOrderId,omitempty"`
		}
		type AddOrderResponse struct {
			OrderID model.OrderId  `json:"orderId"`
			Trades  []*model.Trade `json:"trades,omitempty"`
			Status  string         `json:"status"` // "accepted", "rejected"
			Message string         `json:"message,omitempty"`
		}
		req, err := decodeJSON[AddOrderRequest](w, r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}

		// Basic validation (adjust as needed)
		if req.Quantity <= 0 {
			writeJSONError(w, http.StatusBadRequest, errors.New("quantity must be > 0"))
			return
		}

		uc := *usecase
		trades, orderID, err := uc.AddOrder(r.Context(), req.Side, req.Price, req.Quantity, req.Type)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, AddOrderResponse{
				Status:  "rejected",
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, AddOrderResponse{
			OrderID: orderID,
			Trades:  trades,
			Status:  "accepted",
		})
	})))
	serverRouter.Handle("PUT /api/v1/order/modify", logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type ModifyOrderRequest struct {
			ID       model.OrderId   `json:"id"`
			Price    model.Price     `json:"price,omitempty"`
			Quantity model.Quantity  `json:"quantity,omitempty"`
			Type     model.OrderType `json:"type,omitempty"`
		}
		type ModifyOrderResponse struct {
			OrderID model.OrderId  `json:"orderId"`
			Trades  []*model.Trade `json:"trades,omitempty"`
			Status  string         `json:"status"`
			Message string         `json:"message,omitempty"`
		}

		req, err := decodeJSON[ModifyOrderRequest](w, r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		if req.ID == 0 {
			writeJSONError(w, http.StatusBadRequest, errors.New("id is required"))
			return
		}

		// Build a modify command; adapt to your real usecase signature
		modify := model.OrderModify{
			ID:       req.ID,
			Price:    req.Price,
			Quantity: req.Quantity,
		}
		newType := req.Type
		uc := *usecase
		trades, err := uc.ModifyOrder(r.Context(), modify, newType)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, ModifyOrderResponse{
				OrderID: req.ID,
				Status:  "rejected",
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, ModifyOrderResponse{
			OrderID: req.ID,
			Trades:  trades,
			Status:  "accepted",
		})
	})))
	serverRouter.Handle("DELETE /api/v1/order/cancel", logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type CancelOrderRequest struct {
			ID model.OrderId `json:"id"`
		}
		type CancelOrderResponse struct {
			OrderID model.OrderId `json:"orderId"`
			Status  string        `json:"status"`
			Message string        `json:"message,omitempty"`
		}

		req, err := decodeJSON[CancelOrderRequest](w, r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
		if req.ID == 0 {
			writeJSONError(w, http.StatusBadRequest, errors.New("id is required"))
			return
		}

		uc := *usecase
		err = uc.CancelOrder(r.Context(), req.ID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, CancelOrderResponse{
				OrderID: req.ID,
				Status:  "rejected",
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, CancelOrderResponse{
			OrderID: req.ID,
			Status:  "accepted",
		})
	})))
}
func bindUser(serverRouter *http.ServeMux) {
	serverRouter.Handle("GET /api/v1/user/{id}", logging(http.HandlerFunc(defaultHandler)))
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

// decodeJSON reads and unmarshals the request body into T with sane limits and timeouts.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var zero T

	// Optional: enforce a context timeout for reading large bodies
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	// Optional: limit max body size to prevent abuse
	const maxBody = int64(1 << 20) // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req T
	if err := dec.Decode(&req); err != nil {
		// Provide clearer errors
		if errors.Is(err, io.EOF) {
			return zero, errors.New("empty body")
		}
		return zero, err
	}

	// Ensure there’s no trailing garbage
	if dec.More() {
		return zero, errors.New("multiple JSON values in body")
	}

	return req, nil
}

// writeJSON marshals v and writes it with status and proper headers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	// Optional: for readability in dev
	// enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// writeJSONError writes a simple error response as JSON.
func writeJSONError(w http.ResponseWriter, status int, err error) {
	type errorResp struct {
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Message string `json:"message,omitempty"`
	}
	writeJSON(w, status, errorResp{
		Error:   http.StatusText(status),
		Status:  status,
		Message: err.Error(),
	})
}
