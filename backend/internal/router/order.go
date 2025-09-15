package router

import (
	"errors"
	"net/http"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
)

type OrderRouter interface {
	Add(w http.ResponseWriter, r *http.Request)
	Modify(w http.ResponseWriter, r *http.Request)
	Cancel(w http.ResponseWriter, r *http.Request)
}

type orderRouterImpl struct {
	usecase *order.OrderUseCase
}

func NewOrderRouter(usecase *order.OrderUseCase) OrderRouter {
	return &orderRouterImpl{
		usecase: usecase,
	}
}

func (or *orderRouterImpl) Add(w http.ResponseWriter, r *http.Request) {
	type AddOrderRequest struct {
		Side     model.Side      `json:"side"` // "BID" or "ASK" (or BUY/SELL depending on your model)
		Price    model.Price     `json:"price"`
		Quantity model.Quantity  `json:"quantity"`
		Type     model.OrderType `json:"type"` // e.g., LIMIT, MARKET, FOK, IOC, etc.
		Ticker   string          `json:"ticker"`
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

	uc := *or.usecase
	trades, orderID, err := uc.AddOrder(r.Context(), req.Ticker, req.Side, req.Price, req.Quantity, req.Type)
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
}

func (or *orderRouterImpl) Modify(w http.ResponseWriter, r *http.Request) {
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
	uc := *or.usecase
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
}

func (or *orderRouterImpl) Cancel(w http.ResponseWriter, r *http.Request) {
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

	uc := *or.usecase
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
}
