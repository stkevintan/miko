package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/auth"
	"github.com/stkevintan/miko/pkg/di"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (h *Handler) GenerateToken(ctx context.Context, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(auth.ResolveJWTSecret(ctx))
}

func (h *Handler) jwtAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Authorization header is required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return auth.ResolveJWTSecret(r.Context()), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Token is expired"})
			} else {
				JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid token"})
			}
			return
		}

		if !token.Valid {
			JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid token"})
			return
		}

		ctx := di.Inherit(r.Context())
		di.Provide(ctx, models.Username(claims.Username))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
