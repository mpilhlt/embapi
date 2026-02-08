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

func TestInstanceSharingFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator - return different keys for each call
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Once()  // Alice's key
	mockKeyGen.On("RandomKey", 32).Return("abcdefghijklmnopqrstuvwxyz123456", nil).Once()  // Bob's key
	mockKeyGen.On("RandomKey", 32).Return("98765432109876543210987654321098", nil).Maybe() // Any additional keys

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create users to be used in sharing tests
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	bobJSON := `{"user_handle": "bob", "name": "Bob Smith", "email": "bob@foo.bar"}`
	bobAPIKey, err := createUser(t, bobJSON)
	if err != nil {
		t.Fatalf("Error creating user bob for testing: %v\n", err)
	}

	// Create API standard to be used in tests
	openaiJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, openaiJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create an instance for alice
	instanceJSON := `{"instance_handle": "embedding1", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Alice's OpenAI instance", "api_standard": "openai", "model": "text-embedding-3-large", "dimensions": 3072}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating instance for sharing tests: %v\n", err)
	}

	fmt.Printf("\nRunning llm-instances sharing tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyJSON     string
		VDBKey       string
		expectBody   string
		expectStatus int16
	}{
		{
			name:         "Share instance with nonexistent user - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/alice/embedding1/share",
			bodyJSON:     `{"share_with_handle": "charlie", "role": "reader"}`,
			VDBKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"target user charlie does not exist: user charlie not found\"\n}\n",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Share nonexistent instance - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/alice/nonexistent/share",
			bodyJSON:     `{"share_with_handle": "bob", "role": "reader"}`,
			VDBKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"instance alice/nonexistent not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Bob cannot share alice's instance - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/alice/embedding1/share",
			bodyJSON:     `{"share_with_handle": "alice", "role": "editor"}`,
			VDBKey:       bobAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Share instance with bob - valid",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/alice/embedding1/share",
			bodyJSON:     `{"share_with_handle": "bob", "role": "reader"}`,
			VDBKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ShareInstanceResponseBody.json\",\n  \"owner\": \"alice\",\n  \"instance_handle\": \"embedding1\",\n  \"shared_with\": [\n    {\n      \"user_handle\": \"bob\",\n      \"role\": \"reader\"\n    }\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Get shared users for instance",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice/embedding1/shared-with",
			bodyJSON:     "",
			VDBKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetInstanceSharedUsersResponseBody.json\",\n  \"owner\": \"alice\",\n  \"instance_handle\": \"embedding1\",\n  \"shared_with\": [\n    {\n      \"user_handle\": \"bob\",\n      \"role\": \"reader\"\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Unshare instance from bob",
			method:       http.MethodDelete,
			requestPath:  "/v1/llm-instances/alice/embedding1/share/bob",
			bodyJSON:     "",
			VDBKey:       aliceAPIKey,
			expectBody:   "",
			expectStatus: http.StatusNoContent,
		},
		{
			name:         "Get shared users after unsharing - should be empty",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice/embedding1/shared-with",
			bodyJSON:     "",
			VDBKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetInstanceSharedUsersResponseBody.json\",\n  \"owner\": \"alice\",\n  \"instance_handle\": \"embedding1\",\n  \"shared_with\": []\n}\n",
			expectStatus: http.StatusOK,
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {
			requestURL := fmt.Sprintf("http://%s:%d%s", options.Host, options.Port, v.requestPath)

			var req *http.Request
			if v.bodyJSON != "" {
				req, err = http.NewRequest(v.method, requestURL, bytes.NewBuffer([]byte(v.bodyJSON)))
			} else {
				req, err = http.NewRequest(v.method, requestURL, nil)
			}
			assert.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+v.VDBKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v\n", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != int(v.expectStatus) {
				t.Errorf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			} else {
				t.Logf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			}

			respBody, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			formattedResp := ""
			if v.expectBody != "" {
				fr := new(bytes.Buffer)
				err = json.Indent(fr, respBody, "", "  ")
				assert.NoError(t, err)
				formattedResp = fr.String()
			}
			assert.Equal(t, v.expectBody, formattedResp, "they should be equal")
		})
	}

	// Verify that the expectations regarding the mock key generation were met
	mockKeyGen.AssertExpectations(t)

	// Cleanup removes items created by the tests
	t.Cleanup(func() {
		fmt.Print("\n\nRunning cleanup ...\n\n")

		requestURL := fmt.Sprintf("http://%s:%d/v1/admin/footgun", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+options.AdminKey)
		_, err = http.DefaultClient.Do(req)
		if err != nil && err.Error() != "no rows in result set" {
			t.Fatalf("Error sending request: %v\n", err)
		}
		assert.NoError(t, err)

		fmt.Print("Shutting down server\n\n")
		shutDownServer()
	})

	fmt.Printf("\n")
}

func TestDefinitionSharingFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator - return different keys for each call
	mockKeyGen.On("RandomKey", 32).Return("11111111111111111111111111111111", nil).Once()  // Alice's key
	mockKeyGen.On("RandomKey", 32).Return("22222222222222222222222222222222", nil).Once()  // Bob's key
	mockKeyGen.On("RandomKey", 32).Return("33333333333333333333333333333333", nil).Maybe() // Any additional keys

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create users
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	bobJSON := `{"user_handle": "bob", "name": "Bob Smith", "email": "bob@foo.bar"}`
	bobAPIKey, err := createUser(t, bobJSON)
	if err != nil {
		t.Fatalf("Error creating user bob for testing: %v\n", err)
	}

	// Create API standard
	/*
		openaiJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
		_, err = createAPIStandard(t, openaiJSON, options.AdminKey)
		if err != nil {
			t.Fatalf("Error creating API standard openai for testing: %v\n", err)
		}
	*/

	// Create a definition for alice
	openaiLargeJSON := `{"owner": "alice", "definition_handle": "openai-large", "endpoint": "https://api.openai.com/v1/embeddings", "description": "OpenAI instance with large model", "api_standard": "openai", "model": "text-embedding-3-large", "dimensions": 3072, "context_limit": 8192, "is_public": false}`
	_, err = createDefinition(t, openaiLargeJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating definition for sharing tests: %v\n", err)
	}

	// Note: _system user and definitions are created by migration 004
	//       They are public by default, so we can test alice creating an
	//       instance based on it. We can also text alice creating a
	//       definition and sharing it, since users can share their own
	//       definitions without needing to be shared with first.
	//       Finally, we can test admin creating and sharing a new
	//       _system definition.
	fmt.Printf("\nRunning llm-definitions sharing tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyJSON     string
		VDBKey       string
		expectBody   string
		expectStatus int16
	}{
		{
			name:         "Bob cannot create an instance based on Alice's definition - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/bob/from-definition",
			bodyJSON:     `{"user_handle": "bob", "instance_handle": "bob-instance1", "definition_owner": "alice", "definition_handle": "openai-large", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Bob's instance based on Alice's definition"}`,
			VDBKey:       bobAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Forbidden\",\n  \"status\": 403,\n  \"detail\": \"user does not have access to definition alice/openai-large\"\n}\n",
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "Create an instance based on a nonexistent definition - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/bob/from-definition",
			bodyJSON:     `{"user_handle": "bob", "instance_handle": "bob-instance1", "definition_owner": "alice", "definition_handle": "nonexistant", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Bob's instance based on Alice's definition"}`,
			VDBKey:       bobAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"definition alice/nonexistant not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Bob can create an instance based on a _system definition - should succeed",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/bob/from-definition",
			bodyJSON:     `{"user_handle": "bob", "instance_handle": "bob-instance1", "definition_owner": "_system", "definition_handle": "openai-large", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Bob's instance based on _system's definition"}`,
			VDBKey:       bobAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadInstanceResponseBody.json\",\n  \"owner\": \"bob\",\n  \"instance_handle\": \"bob-instance1\",\n  \"instance_id\": 1\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Alice shares her definition with Bob - should succeed",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-definitions/alice/openai-large/share",
			bodyJSON:     `{"share_with_handle": "bob"}`,
			VDBKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ShareDefinitionResponseBody.json\",\n  \"owner\": \"alice\",\n  \"definition_handle\": \"openai-large\",\n  \"shared_with\": [\n    \"bob\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Get shared users for alice's definition - should succeed when called by alice",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-definitions/alice/openai-large/shared-with",
			bodyJSON:     "",
			VDBKey:       aliceAPIKey,
			expectBody:   "[\n  \"bob\"\n]\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Alice unshares her definition from Bob - should succeed",
			method:       http.MethodDelete,
			requestPath:  "/v1/llm-definitions/alice/openai-large/share/bob",
			bodyJSON:     "",
			VDBKey:       aliceAPIKey,
			expectBody:   "",
			expectStatus: http.StatusNoContent,
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {
			requestURL := fmt.Sprintf("http://%s:%d%s", options.Host, options.Port, v.requestPath)

			var req *http.Request
			if v.bodyJSON != "" {
				req, err = http.NewRequest(v.method, requestURL, bytes.NewBuffer([]byte(v.bodyJSON)))
			} else {
				req, err = http.NewRequest(v.method, requestURL, nil)
			}
			assert.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+v.VDBKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v\n", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != int(v.expectStatus) {
				t.Errorf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			} else {
				t.Logf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			}

			respBody, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			formattedResp := ""
			if v.expectBody != "" {
				fr := new(bytes.Buffer)
				err = json.Indent(fr, respBody, "", "  ")
				assert.NoError(t, err)
				formattedResp = fr.String()
			}
			assert.Equal(t, v.expectBody, formattedResp, "they should be equal")
		})
	}

	// Verify mock expectations
	mockKeyGen.AssertExpectations(t)

	// Cleanup
	t.Cleanup(func() {
		fmt.Print("\n\nRunning cleanup ...\n\n")

		requestURL := fmt.Sprintf("http://%s:%d/v1/admin/footgun", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+options.AdminKey)
		_, err = http.DefaultClient.Do(req)
		if err != nil && err.Error() != "no rows in result set" {
			t.Fatalf("Error sending request: %v\n", err)
		}
		assert.NoError(t, err)

		fmt.Print("Shutting down server\n\n")
		shutDownServer()
	})

	fmt.Printf("\n")
}
