package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

type AuthKey struct{}

func AuthMiddleware(tokenMaker *JWTMaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := verifyClaimsFromAuthHeader(r, tokenMaker)
			if err != nil {
				http.Error(w, fmt.Sprintf("error verifying token: %v", err), http.StatusUnauthorized)
				return
			}

			// pass the payload/claims down the context
			ctx := context.WithValue(r.Context(), AuthKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func verifyClaimsFromAuthHeader(r *http.Request, tokenMaker *JWTMaker) (*UserClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header is missing")
	}

	fields := strings.Fields(authHeader)
	if len(fields) != 2 || fields[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header")
	}

	token := fields[1]
	claims, err := tokenMaker.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}
