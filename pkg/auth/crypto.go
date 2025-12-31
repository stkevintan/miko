package auth

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

var transmissionKey = []byte("miko-transmission-key-32bytes-!!")
var transmissionIV = []byte("miko-iv-16bytes!")

// DecryptTransmission decrypts the password sent from the client.
func DecryptTransmission(cryptoText string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(transmissionKey)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, transmissionIV)
	plainText := make([]byte, len(data))
	mode.CryptBlocks(plainText, data)

	// Unpad PKCS7
	padding := int(plainText[len(plainText)-1])
	if padding < 1 || padding > aes.BlockSize {
		return "", errors.New("invalid padding")
	}
	return string(plainText[:len(plainText)-padding]), nil
}
