package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSystemUserRestrictions verifies that the _system user cannot send requests
// The _system user is a read-only account that can only own resources, not create them
func TestSystemUserRestrictions(t *testing.T) {
	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)
	defer shutDownServer()

	// Create a regular user (alice) for testing
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	// Create API standard for LLM service testing
	apiStandardJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, apiStandardJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create LLM Service for alice
	instanceJSON := `{ "instance_handle": "embedding1", "endpoint": "https://api.foo.bar/v1/embed", "description": "An LLM Service just for testing", "api_standard": "openai", "model": "embed-test1", "dimensions": 5}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating LLM service for testing: %v\n", err)
	}

	// Create project for alice
	projectJSON := `{"project_handle": "test1", "description": "A test project", "instance_owner": "alice", "instance_handle": "embedding1"}`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project alice/test1 for testing: %v\n", err)
	}

	// Get _system user's API key by retrieving the user (admin only)
	reqURL := fmt.Sprintf("http://%v:%d/v1/users/_system", options.Host, options.Port)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+options.AdminKey)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should be able to retrieve _system user info")

	var systemUser struct {
		UserHandle string `json:"user_handle"`
		VDBKey     string `json:"vdb_key"`
	}
	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(respBody, &systemUser)
	assert.NoError(t, err)
	systemAPIKey := systemUser.VDBKey

	// Define test cases - all should fail with 403 Forbidden
	testCases := []struct {
		name         string
		method       string
		path         string
		body         string
		expectStatus int
	}{
		{
			name:         "System user cannot create projects",
			method:       http.MethodPut,
			path:         "/v1/projects/_system/forbidden-project",
			body:         `{"project_handle": "forbidden-project", "description": "Should not be allowed"}`,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot delete projects",
			method:       http.MethodDelete,
			path:         "/v1/projects/_system/some-project",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot upload embeddings",
			method:       http.MethodPost,
			path:         "/v1/embeddings/_system/test1",
			body:         `{"embeddings": []}`,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot delete all project embeddings",
			method:       http.MethodDelete,
			path:         "/v1/embeddings/_system/test1",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot delete specific document embeddings",
			method:       http.MethodDelete,
			path:         "/v1/embeddings/_system/test1/doc-id-123",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot post similarity requests",
			method:       http.MethodPost,
			path:         "/v1/similars/_system/test1",
			body:         `{"vector": [1.0, 2.0, 3.0], "vector_dim": 3}`,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot get similarity requests",
			method:       http.MethodGet,
			path:         "/v1/similars/_system/test1/some-text-id",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot update itself",
			method:       http.MethodPut,
			path:         "/v1/users/_system",
			body:         `{"user_handle": "_system", "name": "System", "email": "system@test.com"}`,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot delete itself",
			method:       http.MethodDelete,
			path:         "/v1/users/_system",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot create LLM service definitions",
			method:       http.MethodPut,
			path:         "/v1/llm-definitions/_system/test-def",
			body:         `{"definition_handle": "test-def", "api_standard": "openai", "model": "test", "dimensions": 5}`,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot delete LLM service definitions",
			method:       http.MethodDelete,
			path:         "/v1/llm-definitions/_system/openai-large",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot create LLM service instances",
			method:       http.MethodPut,
			path:         "/v1/llm-instances/_system/test-instance",
			body:         `{"instance_handle": "test-instance", "endpoint": "https://test.com", "api_standard": "openai", "model": "test", "dimensions": 5}`,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "System user cannot delete LLM service instances",
			method:       http.MethodDelete,
			path:         "/v1/llm-instances/_system/test-instance",
			body:         "",
			expectStatus: http.StatusForbidden,
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := io.Reader(nil)
			if tc.body != "" {
				reqBody = bytes.NewReader([]byte(tc.body))
			}

			requestURL := fmt.Sprintf("http://%v:%d%v", options.Host, options.Port, tc.path)
			req, err := http.NewRequest(tc.method, requestURL, reqBody)
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+systemAPIKey)
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectStatus, resp.StatusCode,
				"Expected %s request to %s to return status %d, got %d",
				tc.method, tc.path, tc.expectStatus, resp.StatusCode)

			// Verify error message contains expected text
			if resp.StatusCode == http.StatusForbidden {
				respBody, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)

				var errorResp struct {
					Detail string `json:"detail"`
				}
				err = json.Unmarshal(respBody, &errorResp)
				assert.NoError(t, err)
				assert.Contains(t, errorResp.Detail, "cannot send requests",
					"Error message should indicate _system user cannot send requests")
			}
		})
	}
}
