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

func TestEditorPermissionsFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator - return different keys for each call
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Once()  // Alice's key
	mockKeyGen.On("RandomKey", 32).Return("abcdefghijklmnopqrstuvwxyz123456", nil).Once()  // Bob's key
	mockKeyGen.On("RandomKey", 32).Return("fedcba9876543210fedcba9876543210", nil).Once()  // Charlie's key
	mockKeyGen.On("RandomKey", 32).Return("98765432109876543210987654321098", nil).Maybe() // Any additional keys

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

	charlieJSON := `{"user_handle": "charlie", "name": "Charlie Brown", "email": "charlie@foo.bar"}`
	charlieAPIKey, err := createUser(t, charlieJSON)
	if err != nil {
		t.Fatalf("Error creating user charlie for testing: %v\n", err)
	}

	// Create API standard
	openaiJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, openaiJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create an instance for alice
	instanceJSON := `{"instance_handle": "embedding1", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Alice's OpenAI instance", "api_standard": "openai", "model": "text-embedding-3-large", "dimensions": 5}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating instance for testing: %v\n", err)
	}

	// Create a project for alice
	projectJSON := `{"project_handle": "test-project", "description": "Test project for editor permissions", "instance_owner": "alice", "instance_handle": "embedding1", "public_read": false}`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project for testing: %v\n", err)
	}

	fmt.Printf("\nRunning editor permissions tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyPath     string
		apiKey       string
		expectStatus int16
		description  string
	}{
		{
			name:         "Share project with bob as reader",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/test-project/share",
			bodyPath:     "../../testdata/share_project_with_bob_reader.json",
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusCreated,
			description:  "Alice shares project with bob as reader",
		},
		{
			name:         "Bob (reader) cannot POST embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test-project",
			bodyPath:     "../../testdata/test_embeddings.json",
			apiKey:       bobAPIKey,
			expectStatus: http.StatusUnauthorized,
			description:  "Reader role should not be able to POST embeddings",
		},
		{
			name:         "Share project with charlie as editor",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/test-project/share",
			bodyPath:     "../../testdata/share_project_with_charlie_reader.json",
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusCreated,
			description:  "Alice shares project with charlie as editor (we'll update the role)",
		},
		{
			name:         "Unshare charlie to update role",
			method:       http.MethodDelete,
			requestPath:  "/v1/projects/alice/test-project/share/charlie",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusNoContent,
			description:  "Remove charlie to re-add with editor role",
		},
		{
			name:         "Re-share with charlie as editor",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice/test-project/share",
			bodyPath:     "../../testdata/share_project_charlie_editor.json",
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusCreated,
			description:  "Alice shares project with charlie as editor",
		},
		{
			name:         "Charlie (editor) can POST embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test-project",
			bodyPath:     "../../testdata/test_embeddings.json",
			apiKey:       charlieAPIKey,
			expectStatus: http.StatusCreated,
			description:  "Editor role should be able to POST embeddings",
		},
		{
			name:         "Bob (reader) can GET embeddings",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/test-project/test-doc-1",
			bodyPath:     "",
			apiKey:       bobAPIKey,
			expectStatus: http.StatusOK,
			description:  "Reader role should be able to GET embeddings",
		},
		{
			name:         "Bob (reader) cannot DELETE embeddings",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test-project/test-doc-1",
			bodyPath:     "",
			apiKey:       bobAPIKey,
			expectStatus: http.StatusUnauthorized,
			description:  "Reader role should not be able to DELETE embeddings",
		},
		{
			name:         "Charlie (editor) can DELETE embeddings",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test-project/test-doc-1",
			bodyPath:     "",
			apiKey:       charlieAPIKey,
			expectStatus: http.StatusNoContent,
			description:  "Editor role should be able to DELETE embeddings",
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {
			fmt.Printf("  Testing: %s\n", v.description)

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

			// Read response body for debugging
			respBody, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			if resp.StatusCode >= 400 {
				fr := new(bytes.Buffer)
				err = json.Indent(fr, respBody, "", "  ")
				if err == nil {
					t.Logf("Response body: %s\n", fr.String())
				}
			}
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
