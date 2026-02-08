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

func TestProjectCreationWithSharingFunc(t *testing.T) {

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

	// Create users
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	bobJSON := `{"user_handle": "bob", "name": "Bob Smith", "email": "bob@foo.bar"}`
	_, err = createUser(t, bobJSON)
	if err != nil {
		t.Fatalf("Error creating user bob for testing: %v\n", err)
	}

	// Create API standard
	openaiJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, openaiJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create an instance for alice
	instanceJSON := `{"instance_handle": "embedding1", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Alice's OpenAI instance", "api_standard": "openai", "model": "text-embedding-3-large", "dimensions": 3072}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating instance for testing: %v\n", err)
	}

	fmt.Printf("\nRunning project creation with sharing test ...\n\n")

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
			name:         "Create project with shared user at creation",
			method:       http.MethodPut,
			requestPath:  "/v1/projects/alice/shared-project",
			bodyPath:     "../../testdata/project_with_shared_user.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ProjectBrief.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"shared-project\",\n  \"project_id\": 1,\n  \"public_read\": false,\n  \"role\": \"owner\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Verify project was shared with bob",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/shared-project/shared-with",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjectSharedUsersResponseBody.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"shared-project\",\n  \"shared_with\": [\n    {\n      \"user_handle\": \"bob\",\n      \"role\": \"reader\"\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
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
