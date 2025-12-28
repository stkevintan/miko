package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (h *Handler) getJWTSecret() []byte {
	if h.jwtSecret != nil {
		return h.jwtSecret
	}

	// 1. Check config
	if h.cfg.Server.JWTSecret != "" {
		h.jwtSecret = []byte(h.cfg.Server.JWTSecret)
		return h.jwtSecret
	}

	// 2. Check database
	var setting models.SystemSetting
	if err := h.db.Where("key = ?", "jwt_secret").First(&setting).Error; err == nil {
		h.jwtSecret = []byte(setting.Value)
		return h.jwtSecret
	}

	// 3. Generate and store
	log.Info("No JWT secret found in config or database, generating a new one...")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Error("Failed to generate random JWT secret: %v", err)
		return []byte("miko-fallback-secret-key")
	}
	secret := hex.EncodeToString(b)
	h.db.Create(&models.SystemSetting{
		Key:   "jwt_secret",
		Value: secret,
	})

	h.jwtSecret = []byte(secret)
	return h.jwtSecret
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
