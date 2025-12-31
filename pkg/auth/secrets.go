package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
)

var jwtSecretName = "jwt_secret"
var passwordSecretName = "password_secret"

func ResolveJWTSecret(ctx context.Context) []byte {
	return ResolveSecret(ctx, jwtSecretName, func(cfg *config.Config) string {
		return cfg.Server.JWTSecret
	})
}

func ResolvePasswordSecret(ctx context.Context) []byte {
	return ResolveSecret(ctx, passwordSecretName, func(cfg *config.Config) string {
		return cfg.Server.PasswordSecret
	})
}

func ResolveSecret(ctx context.Context, name string, fromConfig func(*config.Config) string) []byte {
	if secret, err := di.InvokeNamed[[]byte](ctx, name); err == nil {
		return secret
	}
	cfg := di.MustInvoke[*config.Config](ctx)

	// 1. Check config
	if s := fromConfig(cfg); s != "" {
		secret := []byte(s)
		di.ProvideNamed(ctx, name, secret)
		return secret
	}

	// 2. Check database
	db := di.MustInvoke[*gorm.DB](ctx)

	var setting models.SystemSetting
	if err := db.Where("key = ?", name).First(&setting).Error; err == nil {
		secret := []byte(setting.Value)
		di.ProvideNamed(ctx, name, secret)
		return secret
	}

	// 3. Generate and store
	log.Info("No %s found in config or database, generating a new one...", name)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Critical failure: failed to generate random %s: %v", name, err)
	}
	secret := []byte(hex.EncodeToString(b))
	if err := db.Create(&models.SystemSetting{
		Key:   name,
		Value: string(secret),
	}).Error; err != nil {
		log.Warn("failed to save generated %s to database: %v", name, err)
	}
	di.ProvideNamed(ctx, name, secret)
	return secret
}
