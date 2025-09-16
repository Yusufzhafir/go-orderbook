package router

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/router/middleware"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/user"
)

type UserRouter interface {
	GetUser(w http.ResponseWriter, r *http.Request)
	GetUserOrderList(w http.ResponseWriter, r *http.Request)
	GetUserTransactions(w http.ResponseWriter, r *http.Request)
	AddUserMoney(w http.ResponseWriter, r *http.Request)
	RegisterUser(w http.ResponseWriter, r *http.Request)
	LoginUser(w http.ResponseWriter, r *http.Request)
	// GetUser(w http.ResponseWriter, r *http.Request)
}

type userRouterImpl struct {
	usecase      *user.UserUseCase
	orderUsecase *order.OrderUseCase
	tokenMaker   *middleware.JWTMaker
}

func NewUserRouter(usecase *user.UserUseCase, tokenMaker *middleware.JWTMaker, orderUsecase *order.OrderUseCase) UserRouter {
	return &userRouterImpl{
		usecase:      usecase,
		tokenMaker:   tokenMaker,
		orderUsecase: orderUsecase,
	}
}

func (ur *userRouterImpl) GetUser(w http.ResponseWriter, r *http.Request) {
	type UserResponse struct {
		Id        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		Username  string    `json:"username"`
		Balance   string    `json:"balance"`
	}
	claims := r.Context().Value(middleware.AuthKey{}).(*middleware.UserClaims)

	user, err := (*ur.usecase).GetProfile(r.Context(), claims.UserId)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, UserResponse{
		Id:        fmt.Sprintf("%d", (*user).ID),
		CreatedAt: (*user).CreatedAt,
		Username:  user.Username,
		Balance:   (*user).UserBalance,
	})

}
func (ur *userRouterImpl) GetUserOrderList(w http.ResponseWriter, r *http.Request) {
	userClaim := r.Context().Value(middleware.AuthKey{}).(*middleware.UserClaims)
	orders, err := (*ur.orderUsecase).GetOrderByUserId(r.Context(), userClaim.UserId, true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, orders)

}
func (ur *userRouterImpl) GetUserTransactions(w http.ResponseWriter, r *http.Request) {

}
func (ur *userRouterImpl) AddUserMoney(w http.ResponseWriter, r *http.Request) {
	type AddMoneyReq struct {
		Amount int64 `json:"amount"`
	}
	type AddMoneyRes struct {
		Message string `json:"message"`
	}
	claims := r.Context().Value(middleware.AuthKey{}).(*middleware.UserClaims)

	req, err := decodeJSON[AddMoneyReq](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	errTransfer := (*ur.usecase).TopupMoney(r.Context(), claims.UserId, big.NewInt(req.Amount))
	if errTransfer != nil {
		writeJSONError(w, http.StatusBadRequest, errTransfer)
		return
	}

	writeJSON(w, http.StatusOK, AddMoneyRes{
		Message: fmt.Sprintf("successfully added %d to your account", req.Amount),
	})
}
func (ur *userRouterImpl) RegisterUser(w http.ResponseWriter, r *http.Request) {
	type RegisterRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	type UserResponse struct {
		Id        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		Username  string    `json:"username"`
	}
	req, err := decodeJSON[RegisterRequest](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	userId, err := (*ur.usecase).Register(r.Context(), req.Username, req.Password)

	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, UserResponse{
		Id:        fmt.Sprintf("%d", userId),
		CreatedAt: time.Now(),
		Username:  req.Username,
	})

}
func (ur *userRouterImpl) LoginUser(w http.ResponseWriter, r *http.Request) {
	type LoginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	type LoginRes struct {
		Token     string    `json:"token"`
		Id        string    `json:"id"`
		Username  string    `json:"username"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	req, err := decodeJSON[LoginReq](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	user, err := (*ur.usecase).Login(r.Context(), req.Username, req.Password)

	if err != nil || user == nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	tokenMaker := *ur.tokenMaker
	newToken, newClaim, err := tokenMaker.CreateToken(user.ID, req.Username, time.Hour*24)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, LoginRes{
		Token:     newToken,
		Id:        newClaim.ID,
		Username:  user.Username,
		ExpiresAt: newClaim.ExpiresAt.Time, // optional
	})
}
