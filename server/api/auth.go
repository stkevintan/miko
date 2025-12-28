package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stkevintan/miko/models"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (h *Handler) getJWTSecret() []byte {
	if h.cfg.Server.JWTSecret != "" {
		return []byte(h.cfg.Server.JWTSecret)
	}
	return []byte("miko-secret-key")
}

func (h *Handler) GenerateToken(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.getJWTSecret())
}

func (h *Handler) jwtAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
				return h.getJWTSecret(), nil
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

			ctx := context.WithValue(r.Context(), usernameKey, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
