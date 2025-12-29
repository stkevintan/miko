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
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtSecretName = "jwt_secret"

func (h *Handler) resolveJWTSecret(ctx context.Context) []byte {
	if jwtSecret, err := di.InvokeNamed[[]byte](ctx, jwtSecretName); err == nil {
		return jwtSecret
	}
	cfg := di.MustInvoke[*config.Config](ctx)

	// 1. Check config
	if cfg.Server.JWTSecret != "" {
		jwtSecret := []byte(cfg.Server.JWTSecret)
		di.ProvideNamed(ctx, jwtSecretName, jwtSecret)
		return jwtSecret
	}

	// 2. Check database
	db := di.MustInvoke[*gorm.DB](ctx)

	var setting models.SystemSetting
	if err := db.Where("key = ?", jwtSecretName).First(&setting).Error; err == nil {
		jwtSecret := []byte(setting.Value)
		di.ProvideNamed(ctx, jwtSecretName, jwtSecret)
		return jwtSecret
	}

	// 3. Generate and store
	log.Info("No JWT secret found in config or database, generating a new one...")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Critical failure: failed to generate random JWT secret: %v", err)
	}
	secret := []byte(hex.EncodeToString(b))
	if err := db.Create(&models.SystemSetting{
		Key:   jwtSecretName,
		Value: string(secret),
	}).Error; err != nil {
		log.Warn("failed to save generated JWT secret to database: %v", err)
	}
	di.ProvideNamed(ctx, jwtSecretName, secret)
	return secret
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
	return token.SignedString(h.resolveJWTSecret(ctx))
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
			return h.resolveJWTSecret(r.Context()), nil
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
