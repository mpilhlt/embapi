package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrInvalidKey        = errors.New("encryption key must be 32 bytes")
	ErrInvalidCiphertext = errors.New("ciphertext is too short or invalid")
)

// EncryptionKey holds the AES encryption key
type EncryptionKey struct {
	key []byte
}

// NewEncryptionKey creates a new encryption key from the provided string
// The key is hashed using SHA256 to ensure it's exactly 32 bytes (AES-256)
func NewEncryptionKey(keyString string) *EncryptionKey {
	hash := sha256.Sum256([]byte(keyString))
	return &EncryptionKey{key: hash[:]}
}

// GenerateEncryptionKey generates a random encryption key
func GenerateEncryptionKey() (*EncryptionKey, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}
	return &EncryptionKey{key: key}, nil
}

// GetEncryptionKeyFromEnv retrieves the encryption key from environment variable
// If not set, it returns an error
func GetEncryptionKeyFromEnv() (*EncryptionKey, error) {
	keyString := os.Getenv("ENCRYPTION_KEY")
	if keyString == "" {
		return nil, errors.New("ENCRYPTION_KEY environment variable is not set")
	}
	return NewEncryptionKey(keyString), nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *EncryptionKey) Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil // Allow empty strings to be stored as NULL
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *EncryptionKey) Decrypt(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil // Return empty string for NULL/empty data
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptToBase64 encrypts plaintext and returns base64-encoded string
func (e *EncryptionKey) EncryptToBase64(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	if ciphertext == nil {
		return "", nil
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 decrypts base64-encoded ciphertext
func (e *EncryptionKey) DecryptFromBase64(base64Ciphertext string) (string, error) {
	if base64Ciphertext == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	return e.Decrypt(ciphertext)
}
