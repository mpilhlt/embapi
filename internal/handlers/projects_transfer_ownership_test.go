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

func TestProjectTransferOwnershipFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator - return different keys for each call
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Once()  // Alice's key
	mockKeyGen.On("RandomKey", 32).Return("abcdefghijklmnopqrstuvwxyz123456", nil).Once()  // Bob's key
	mockKeyGen.On("RandomKey", 32).Return("98765432109876543210987654321098", nil).Once()  // Charlie's key
	mockKeyGen.On("RandomKey", 32).Return("11111111111111111111111111111111", nil).Maybe() // Any additional keys

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create users to be used in transfer tests
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

	charlieJSON := `{"user_handle": "charlie", "name": "Charlie Brown", "email": "charlie@foo.bar"}`
	charlieAPIKey, err := createUser(t, charlieJSON)
	if err != nil {
		t.Fatalf("Error creating user charlie for testing: %v\n", err)
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
		t.Fatalf("Error creating instance for transfer tests: %v\n", err)
	}

	// Create a project for alice
	projectJSON := `{"project_handle": "project1", "description": "Alice's test project", "instance_owner": "alice", "instance_handle": "embedding1", "public_read": false}`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project for transfer tests: %v\n", err)
	}

	// Create another project for alice that will be transferred to bob (bob already has this project handle)
	project2JSON := `{"project_handle": "project2", "description": "Alice's second project", "instance_owner": "alice", "instance_handle": "embedding1", "public_read": false}`
	_, err = createProject(t, project2JSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating second project for transfer tests: %v\n", err)
	}

	// Create a project for bob with the same handle (to test conflict scenario)
	bobProject2JSON := `{"project_handle": "project2", "description": "Bob's project", "instance_owner": "alice", "instance_handle": "embedding1", "public_read": false}`
	_, err = createProject(t, bobProject2JSON, "bob", bobAPIKey)
	if err != nil {
		t.Fatalf("Error creating project for bob: %v\n", err)
	}

	// Create a project for alice that charlie is shared with (to test shared user becoming owner)
	project3JSON := `{"project_handle": "project3", "description": "Alice's third project", "instance_owner": "alice", "instance_handle": "embedding1", "public_read": false}`
	_, err = createProject(t, project3JSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating third project for transfer tests: %v\n", err)
	}

	// Share project3 with charlie as editor
	shareJSON := `{"share_with_handle": "charlie", "role": "editor"}`
	_, err = shareProject(t, shareJSON, "alice", "project3", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error sharing project with charlie: %v\n", err)
	}

	fmt.Printf("\nRunning project transfer ownership tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyJSON     string
		apiKey       string
		expectStatus int
		expectBody   string
	}{
		{
			name:         "Transfer ownership of nonexistent project - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/nonexistent/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "bob"}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusNotFound,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"project alice/nonexistent not found\"\n}\n",
		},
		{
			name:         "Transfer ownership to nonexistent user - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "nonexistent"}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusNotFound,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"new owner user nonexistent not found\"\n}\n",
		},
		{
			name:         "Transfer ownership to self - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "alice"}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusBadRequest,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"new owner must be different from current owner\"\n}\n",
		},
		{
			name:         "Bob cannot transfer ownership of alice's project - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "bob"}`,
			apiKey:       bobAPIKey,
			expectStatus: http.StatusUnauthorized,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
		},
		{
			name:         "Transfer ownership when new owner already has project with same handle - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project2/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "bob"}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusConflict,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Conflict\",\n  \"status\": 409,\n  \"detail\": \"new owner bob already has a project with handle project2\"\n}\n",
		},
		{
			name:         "Successfully transfer ownership from alice to bob",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "bob"}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusOK,
			expectBody:   "", // We'll validate the response body separately
		},
		{
			name:         "Successfully transfer ownership from alice to charlie (shared user becomes owner)",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project3/transfer-ownership",
			bodyJSON:     `{"new_owner_handle": "charlie"}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusOK,
			expectBody:   "", // We'll validate the response body separately
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare the request body
			var bodyReader io.Reader
			if tc.bodyJSON != "" {
				bodyReader = bytes.NewBufferString(tc.bodyJSON)
			}

			// Build the request URL
			requestURL := fmt.Sprintf("http://%s:%d%s", options.Host, options.Port, tc.requestPath)

			// Create the request
			req, err := http.NewRequest(tc.method, requestURL, bodyReader)
			if err != nil {
				t.Fatalf("Error creating request: %v\n", err)
			}

			// Set headers
			req.Header.Set("Authorization", "Bearer "+tc.apiKey)

			// Make the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Error making request: %v\n", err)
			}
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tc.expectStatus, resp.StatusCode, "Status code mismatch for test: %s", tc.name)

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Error reading response body: %v\n", err)
			}

			// Check response body
			if tc.expectBody != "" {
				fr := new(bytes.Buffer)
				err = json.Indent(fr, body, "", "  ")
				assert.NoError(t, err)
				formattedResp := fr.String()
				assert.Equal(t, tc.expectBody, formattedResp, "Response body mismatch for test: %s", tc.name)
			} else if tc.expectStatus == http.StatusOK {
				// Parse and validate successful response
				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err, "Error parsing JSON response for test: %s", tc.name)

				// Validate response structure
				assert.Contains(t, response, "project_id")
				assert.Contains(t, response, "project_handle")
				assert.Contains(t, response, "old_owner")
				assert.Contains(t, response, "new_owner")

				// Parse body to get new owner
				var bodyMap map[string]string
				json.Unmarshal([]byte(tc.bodyJSON), &bodyMap)
				expectedNewOwner := bodyMap["new_owner_handle"]

				// Validate ownership was actually transferred
				assert.Equal(t, "alice", response["old_owner"])
				assert.Equal(t, expectedNewOwner, response["new_owner"])
			}
		})
	}

	// Additional verification tests
	t.Run("Verify bob is now owner of project1", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/bob/project1", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			t.Fatalf("Error creating request: %v\n", err)
		}
		req.Header.Set("Authorization", "Bearer "+bobAPIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Error making request: %v\n", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %v\n", err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		assert.NoError(t, err)
		assert.Equal(t, "bob", response["owner"])
		assert.Equal(t, "owner", response["role"])
	})

	t.Run("Verify alice cannot access project1 anymore", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/bob/project1", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			t.Fatalf("Error creating request: %v\n", err)
		}
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Error making request: %v\n", err)
		}
		defer resp.Body.Close()

		// Alice should not be able to access it anymore - auth middleware returns 401
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Verify charlie is now owner of project3", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/charlie/project3", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			t.Fatalf("Error creating request: %v\n", err)
		}
		req.Header.Set("Authorization", "Bearer "+charlieAPIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Error making request: %v\n", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %v\n", err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		assert.NoError(t, err)
		assert.Equal(t, "charlie", response["owner"])
		assert.Equal(t, "owner", response["role"])
	})

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

}

// Helper function to share a project
func shareProject(t *testing.T, jsonBody, owner, projectHandle, apiKey string) (string, error) {
	requestURL := fmt.Sprintf("http://%s:%d/v1/projects/%s/%s/share", options.Host, options.Port, owner, projectHandle)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}
