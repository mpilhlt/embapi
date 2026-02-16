package auth_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/mpilhlt/embapi/internal/auth"
	"github.com/mpilhlt/embapi/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestVerboseFlag tests that authentication logging is controlled by the Verbose flag
func TestVerboseFlag(t *testing.T) {
	// Set required environment variables for testing
	if os.Getenv("ENCRYPTION_KEY") == "" {
		os.Setenv("ENCRYPTION_KEY", "test-encryption-key-32-bytes-long-1234567890")
	}

	tests := []struct {
		name       string
		verbose    bool
		expectLogs bool
	}{
		{
			name:       "Verbose=true should produce logs",
			verbose:    true,
			expectLogs: true,
		},
		{
			name:       "Verbose=false should suppress logs",
			verbose:    false,
			expectLogs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout to check for log messages
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create test options
			options := &models.Options{
				Verbose:  tt.verbose,
				AdminKey: "test-admin-key",
			}

			// Create a simple API and router
			config := huma.DefaultConfig("Test API", "1.0.0")
			router := http.NewServeMux()
			api := humago.New(router, config)

			// Register auth middlewares
			api.UseMiddleware(auth.EmbAPIKeyAdminAuth(api, options))
			api.UseMiddleware(auth.AuthTermination(api, options))

			// Register a test endpoint that requires authentication
			huma.Register(api, huma.Operation{
				OperationID: "test-endpoint",
				Method:      http.MethodGet,
				Path:        "/test",
				Security: []map[string][]string{
					{"adminAuth": {}},
				},
			}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
				return &struct{}{}, nil
			})

			// Test 1: Successful authentication (should log if verbose)
			req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
			req1.Header.Set("Authorization", "Bearer test-admin-key")
			resp1 := httptest.NewRecorder()
			router.ServeHTTP(resp1, req1)

			// Capture output
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			os.Stdout = oldStdout
			output := buf.String()

			// Verify status code (204 No Content is expected for successful operation with no body)
			assert.Equal(t, http.StatusNoContent, resp1.Code, "Expected successful authentication")

			// Check for log output based on verbose flag
			if tt.expectLogs {
				assert.Contains(t, output, "Admin authentication successful", "Expected admin auth log when verbose=true")
			} else {
				assert.NotContains(t, output, "Admin authentication successful", "Expected no admin auth log when verbose=false")
			}

			// Test 2: Failed authentication (should log if verbose)
			// Capture stdout again for failed auth test
			r2, w2, _ := os.Pipe()
			os.Stdout = w2

			req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
			req2.Header.Set("Authorization", "Bearer wrong-key")
			resp2 := httptest.NewRecorder()
			router.ServeHTTP(resp2, req2)

			// Capture output
			w2.Close()
			var buf2 bytes.Buffer
			io.Copy(&buf2, r2)
			os.Stdout = oldStdout
			output2 := buf2.String()

			// Verify status code
			assert.Equal(t, http.StatusUnauthorized, resp2.Code, "Expected failed authentication")

			// Check for log output based on verbose flag
			if tt.expectLogs {
				assert.Contains(t, output2, "Authentication failed", "Expected auth failure log when verbose=true")
			} else {
				assert.NotContains(t, output2, "Authentication failed", "Expected no auth failure log when verbose=false")
			}
		})
	}
}

// TestVerboseEnvironmentVariable tests that the environment variable is properly handled
func TestVerboseEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "SERVICE_VERBOSE=true",
			envValue: "true",
			expected: true,
		},
		{
			name:     "SERVICE_VERBOSE=false",
			envValue: "false",
			expected: false,
		},
		{
			name:     "SERVICE_VERBOSE empty",
			envValue: "",
			expected: false, // default value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("SERVICE_VERBOSE")

			if tt.envValue != "" {
				os.Setenv("SERVICE_VERBOSE", tt.envValue)
				defer os.Unsetenv("SERVICE_VERBOSE")
			}

			options := &models.Options{
				Verbose: false, // default
			}

			// Simulate what main.go does
			if os.Getenv("SERVICE_VERBOSE") != "" {
				options.Verbose = os.Getenv("SERVICE_VERBOSE") == "true"
			}

			assert.Equal(t, tt.expected, options.Verbose, "Verbose should match expected value")
		})
	}
}

// TestNoAuthLogsInQuietMode verifies that with Verbose=false, various auth scenarios don't produce logs
func TestNoAuthLogsInQuietMode(t *testing.T) {
	// This test demonstrates the fix for the bulk upload/crawler scenario
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	options := &models.Options{
		Verbose: false, // Quiet mode
		AdminKey:    "test-admin-key",
	}

	config := huma.DefaultConfig("Test API", "1.0.0")
	router := http.NewServeMux()
	api := humago.New(router, config)

	api.UseMiddleware(auth.EmbAPIKeyAdminAuth(api, options))
	api.UseMiddleware(auth.AuthTermination(api, options))

	huma.Register(api, huma.Operation{
		OperationID: "bulk-test",
		Method:      http.MethodPost,
		Path:        "/bulk",
		Security: []map[string][]string{
			{"adminAuth": {}},
		},
	}, func(ctx context.Context, input *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	})

	// Simulate many requests (like bulk upload or crawler)
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodPost, "/bulk", nil)
		req.Header.Set("Authorization", "Bearer test-admin-key")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusNoContent, resp.Code)
	}

	// Capture output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = oldStdout
	output := buf.String()

	// Count authentication success messages - should be 0 in quiet mode
	count := strings.Count(output, "authentication successful")
	assert.Equal(t, 0, count, "Expected no authentication logs in quiet mode during bulk operations")
}
