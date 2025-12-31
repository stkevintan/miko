package crypto

import (
	"context"
	"crypto/md5"
	"fmt"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
)

// ResolvePasswordSecret resolves the password encryption secret from the configuration.
// It does NOT fall back to the database or generate a new one if missing, as per security requirements.
func ResolvePasswordSecret(ctx context.Context) []byte {
	cfg := di.MustInvoke[*config.Config](ctx)
	if cfg.Server.PasswordSecret != "" {
		return []byte(cfg.Server.PasswordSecret)
	}
	log.Fatal("Password secret is not configured. Please set 'server.passwordSecret' or environment variable in the configuration.")
	return nil
}

// ResolveJWTSecret resolves the JWT signing secret from the configuration.
// It does NOT fall back to the database or generate a new one if missing, as per security requirements.
func ResolveJWTSecret(ctx context.Context) []byte {
	cfg := di.MustInvoke[*config.Config](ctx)
	if cfg.Server.JWTSecret != "" {
		return []byte(cfg.Server.JWTSecret)
	}
	log.Fatal("JWT secret is not configured. Please set 'server.jwtSecret' or environment variable in the configuration.")
	return nil
}

// DecryptPassword attempts to decrypt a stored password using the configured secret.
// If decryption fails (e.g., the password was stored in plain text), it returns the original string.
func DecryptPassword(ctx context.Context, storedPassword string) string {
	secret := ResolvePasswordSecret(ctx)
	if secret == nil {
		return storedPassword
	}
	decrypted, err := Decrypt(storedPassword, secret)
	if err != nil {
		return storedPassword
	}
	return decrypted
}

// VerifyPassword verifies if the provided plain text password matches the stored password.
func VerifyPassword(ctx context.Context, storedPassword, plainPassword string) bool {
	return DecryptPassword(ctx, storedPassword) == plainPassword
}

// VerifySubsonicToken verifies the Subsonic token authentication.
// token = md5(password + salt)
func VerifySubsonicToken(ctx context.Context, storedPassword, token, salt string) bool {
	decrypted := DecryptPassword(ctx, storedPassword)
	expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(decrypted+salt)))
	return expectedToken == token
}
