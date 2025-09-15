package middleware

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	tbTypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

type UserClaims struct {
	UserId int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func NewUserClaims(id int64, username string, duration time.Duration) (*UserClaims, error) {
	tokenID := tbTypes.ID().String()
	return &UserClaims{
		UserId: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		},
	}, nil
}
