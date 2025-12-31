package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

func ensureKey32(key []byte) []byte {
	hash := sha256.Sum256(key)
	return hash[:]
}

// Encrypt encrypts plain text using AES-GCM with the provided key.
func Encrypt(plainText string, key []byte) (string, error) {
	if len(key) == 0 {
		return "", errors.New("encryption key is empty")
	}
	block, err := aes.NewCipher(ensureKey32(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64 encoded cipher text using AES-GCM with the provided key.
func Decrypt(cryptoText string, key []byte) (string, error) {
	if len(key) == 0 {
		return "", errors.New("decryption key is empty")
	}
	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(ensureKey32(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
