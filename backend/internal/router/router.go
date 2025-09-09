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

func bindTicker(serverRouter *http.ServeMux) {
	serverRouter.Handle("GET /api/v1/ticker", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-list", logging(http.HandlerFunc(defaultHandler)))
	serverRouter.Handle("GET /api/v1/ticker/{ticker}/order-queue/{price}", logging(http.HandlerFunc(defaultHandler)))
}

func bindOrder(serverRouter *http.ServeMux, usecase *order.OrderUseCase) {
	serverRouter.Handle("POST /api/v1/order/add", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		trades, orderID, err := uc.AddOrder(r.Context(), ord)
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
		orderUsecase := *usecase
		orderUsecase.AddOrder(model.Order{})
	}))
	serverRouter.Handle("PUT /api/v1/order/modify", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type ModifyOrderRequest struct {
			Id model.OrderId
			// Side     model.Side // i dont think we should let users modify order to change side becasue i will be confusing
			Price    model.Price
			Quantity model.Quantity
			Type     model.OrderType
		}
		type ModifyOrderResponse struct {
		}

		orderUsecase := *usecase
		orderUsecase.ModifyOrder(model.Order{})
	}))
	serverRouter.Handle("DELETE /api/v1/order/cancel", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type CancelOrder struct {
			Id model.OrderId
		}
		type AddOrderResponse struct {
		}

		orderUsecase := *usecase
		orderUsecase.CancelOrder(model.Order{})
	}))
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
	bindTicker(opts.ServerRouter)
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
