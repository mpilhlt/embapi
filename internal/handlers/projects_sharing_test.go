package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectSharingFunc(t *testing.T) {

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

	// Create a project for alice
	projectJSON := `{"project_handle": "project1", "description": "Alice's test project", "instance_owner": "alice", "instance_handle": "embedding1", "public_read": false}`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project for sharing tests: %v\n", err)
	}

	fmt.Printf("\nRunning projects sharing tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyPath     string
		apiKey       string
		expectBody   string
		expectStatus int16
	}{
		{
			name:         "Share project with nonexistent user - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/share",
			bodyPath:     "../../testdata/share_project_with_charlie_reader.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"target user charlie does not exist: user charlie not found\"\n}\n",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Share nonexistent project - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/nonexistent/share",
			bodyPath:     "../../testdata/share_project_with_bob_reader.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"project alice/nonexistent not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Bob cannot share alice's project - should fail",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/share",
			bodyPath:     "../../testdata/share_project_with_alice_editor.json",
			apiKey:       bobAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Share project with bob - valid",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/share",
			bodyPath:     "../../testdata/share_project_with_bob_reader.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ShareProjectResponseBody.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"project1\",\n  \"project_id\": 1,\n  \"shared_with_handle\": \"bob\",\n  \"shared_with_role\": \"reader\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Get shared users for project",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/project1/shared-with",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjectSharedUsersResponseBody.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"project1\",\n  \"shared_with\": [\n    {\n      \"user_handle\": \"bob\",\n      \"role\": \"reader\"\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Unshare project from bob",
			method:       http.MethodDelete,
			requestPath:  "/v1/projects/alice/project1/share/bob",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "",
			expectStatus: http.StatusNoContent,
		},
		{
			name:         "Get shared users after unsharing - should be empty",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/project1/shared-with",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjectSharedUsersResponseBody.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"project1\",\n  \"shared_with\": []\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Share project with bob again for access test",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/project1/share",
			bodyPath:     "../../testdata/share_project_with_bob_reader.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ShareProjectResponseBody.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"project1\",\n  \"project_id\": 1,\n  \"shared_with_handle\": \"bob\",\n  \"shared_with_role\": \"reader\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Bob can access shared project",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/project1",
			bodyPath:     "",
			apiKey:       bobAPIKey,
			expectBody:   "", // Just check status code
			expectStatus: http.StatusOK,
		},
		{
			name:         "Bob cannot see shared users list (not owner)",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/project1/shared-with",
			bodyPath:     "",
			apiKey:       bobAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {

			// Handle the body for PUT and POST requests
			reqBody := io.Reader(nil)
			if v.method == http.MethodGet || v.method == http.MethodDelete {
				reqBody = nil
			} else {
				f, err := os.Open(v.bodyPath)
				assert.NoError(t, err)
				defer func() {
					if err := f.Close(); err != nil {
						t.Fatal(err)
					}
				}()
				b := new(bytes.Buffer)
				_, err = io.Copy(b, f)
				assert.NoError(t, err)
				reqBody = bytes.NewReader(b.Bytes())
			}

			requestURL := fmt.Sprintf("http://%s:%d%s", options.Host, options.Port, v.requestPath)
			req, err := http.NewRequest(v.method, requestURL, reqBody)
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+v.apiKey)

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
