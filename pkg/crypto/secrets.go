package crypto

import (
	"context"
	"crypto/md5"
	"fmt"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/pkg/di"
)

// ResolvePasswordSecret resolves the password encryption secret from the configuration.
// It does NOT fall back to the database or generate a new one if missing, as per security requirements.
func ResolvePasswordSecret(ctx context.Context) []byte {
	cfg := di.MustInvoke[*config.Config](ctx)
	return []byte(cfg.Server.PasswordSecret)
}

// ResolveJWTSecret resolves the JWT signing secret from the configuration.
// It does NOT fall back to the database or generate a new one if missing, as per security requirements.
func ResolveJWTSecret(ctx context.Context) []byte {
	cfg := di.MustInvoke[*config.Config](ctx)
	return []byte(cfg.Server.JWTSecret)
}

// DecryptPassword attempts to decrypt a stored password using the configured secret.
// If decryption fails (e.g., the password was stored in plain text), it returns the original string.
func DecryptPassword(ctx context.Context, storedPassword string) string {
	secret := ResolvePasswordSecret(ctx)
	decrypted, err := Decrypt(storedPassword, secret)
	if err != nil {
		return storedPassword // Return original if decryption fails (likely plain text)
	}
	return decrypted
}

// VerifyPassword verifies if the provided plain text password matches the stored password.
func VerifyPassword(ctx context.Context, storedPassword, plainPassword string) (bool, error) {
	decrypted := DecryptPassword(ctx, storedPassword)
	return decrypted == plainPassword, nil
}

// VerifySubsonicToken verifies the Subsonic token authentication.
// token = md5(password + salt)
func VerifySubsonicToken(ctx context.Context, storedPassword, token, salt string) (bool, error) {
	decrypted := DecryptPassword(ctx, storedPassword)
	expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(decrypted+salt)))
	return expectedToken == token, nil
}
