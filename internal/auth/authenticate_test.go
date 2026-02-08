package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestApiKeyIsValid tests the apiKeyIsValid function
func TestApiKeyIsValid(t *testing.T) {
	tests := []struct {
		name       string
		rawKey     string
		storedHash string
		want       bool
	}{
		{
			name:   "Valid API key",
			rawKey: "test-api-key-12345",
			storedHash: func() string {
				hash := sha256.Sum256([]byte("test-api-key-12345"))
				return hex.EncodeToString(hash[:])
			}(),
			want: true,
		},
		{
			name:   "Invalid API key",
			rawKey: "wrong-api-key",
			storedHash: func() string {
				hash := sha256.Sum256([]byte("test-api-key-12345"))
				return hex.EncodeToString(hash[:])
			}(),
			want: false,
		},
		{
			name:   "Empty API key",
			rawKey: "",
			storedHash: func() string {
				hash := sha256.Sum256([]byte("test-api-key-12345"))
				return hex.EncodeToString(hash[:])
			}(),
			want: false,
		},
		{
			name:       "Empty stored hash",
			rawKey:     "test-api-key-12345",
			storedHash: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EmbAPIKeyIsValid(tt.rawKey, tt.storedHash); got != tt.want {
				t.Errorf("apiKeyIsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Note: The authentication middleware functions (APIKeyAdminAuth, APIKeyOwnerAuth, APIKeyReaderAuth)
// are thoroughly tested through integration tests in the handlers package.
// These tests verify:
// - Admin authentication with valid/invalid keys
// - Owner authentication for various resources
// - Reader authentication for shared projects
// - Public access for projects with "*" in shared_with
// - Authentication failure handling
// - Authorization checks for different operations
