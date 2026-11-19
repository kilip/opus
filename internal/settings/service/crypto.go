package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

const (
	secretHKDFInfo = "opus/secrets/v1"
	aes256KeySize  = 32
	currentKeyVer  = 1
)

// DeriveKey derives a 256-bit AES-GCM key from AUTH_SECRET via HKDF-SHA256.
// Rotating AUTH_SECRET invalidates existing ciphertexts; stored secrets must be re-entered.
func DeriveKey(secret string) ([]byte, error) {
	if secret == "" {
		return nil, errors.New("encryption secret is required")
	}
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, secretHKDFInfo, aes256KeySize)
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}
	return key, nil
}

func encryptSecret(key []byte, plaintext string) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create gcm: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext = gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

func decryptSecret(key []byte, ciphertext, nonce []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return "", errors.New("invalid nonce size")
	}
	pt, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(pt), nil
}
