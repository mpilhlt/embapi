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

// TestPublicAccess tests the public access functionality when "*" is in shared_with
func TestPublicAccess(t *testing.T) {

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

	/*
		bobJSON := `{"user_handle": "bob", "name": "Bob Smith", "email": "bob@foo.bar"}`
		bobAPIKey, err := createUser(t, bobJSON)
		if err != nil {
			t.Fatalf("Error creating user bob for testing: %v\n", err)
		}
	*/

	// Create API standard to be used in tests
	openaiJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, openaiJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create an instance for alice
	instanceJSON := `{"instance_handle": "embedding1", "endpoint": "https://api.openai.com/v1/embeddings", "description": "Alice's OpenAI instance", "api_standard": "openai", "model": "text-embedding-3-large", "dimensions": 5}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating instance for sharing tests: %v\n", err)
	}

	// Create public project to be used in embeddings tests
	projectJSON := `{ "project_handle": "public-test", "instance_owner": "alice", "instance_handle": "embedding1", "description": "This is a test project", "public_read": true }`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project alice/public-test for testing: %v\n", err)
	}

	/*
		shareProjectJSON := `{"share_with_handle": "*", "role": "reader"}`
		_, err = shareProject(t, "bob", "public-test", shareProjectJSON, bobAPIKey)
		if err != nil {
			t.Fatalf("Error sharing project bob/public-test with *: %v\n", err)
		}
	*/

	// Post some embeddings to the public project
	_, err = postEmbeddings(t, "../../testdata/valid_embeddings.json", "alice", "public-test", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error posting embeddings: %v\n", err)
	}

	fmt.Printf("\nRunning public access tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyPath     string
		EmbAPIKey       string
		expectBody   string
		expectStatus int16
	}{
		{
			name:         "Get project metadata without authentication (public project)",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/public-test",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ProjectFull.json\",\n  \"project_id\": 1,\n  \"project_handle\": \"public-test\",\n  \"owner\": \"alice\",\n  \"description\": \"This is a test project\",\n  \"public_read\": false,\n  \"instance\": {\n    \"owner\": \"alice\",\n    \"instance_handle\": \"embedding1\",\n    \"instance_id\": 1,\n    \"number_of_projects\": 1,\n    \"number_of_shared_users\": 0\n  },\n  \"role\": \"owner\",\n  \"number_of_embeddings\": 3\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get project embeddings without authentication (public project)",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/public-test",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjEmbeddingsResponseBody.json\",\n  \"embeddings\": [\n    {\n      \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n      \"user_handle\": \"alice\",\n      \"project_handle\": \"public-test\",\n      \"project_id\": 1,\n      \"instance_handle\": \"embedding1\",\n      \"text\": \"This is a test document\",\n      \"vector\": [\n        -0.020843506,\n        0.01852417,\n        0.05328369,\n        0.07141113,\n        0.020004272\n      ],\n      \"vector_dim\": 5,\n      \"metadata\": {\n        \"author\": \"Immanuel Kant\"\n      }\n    },\n    {\n      \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n      \"user_handle\": \"alice\",\n      \"project_handle\": \"public-test\",\n      \"project_id\": 1,\n      \"instance_handle\": \"embedding1\",\n      \"text\": \"This is a similar test document\",\n      \"vector\": [\n        -0.020843506,\n        0.01852417,\n        0.05328369,\n        0.07141113,\n        0.020004272\n      ],\n      \"vector_dim\": 5,\n      \"metadata\": {\n        \"author\": \"Immanuel Kant\"\n      }\n    },\n    {\n      \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\",\n      \"user_handle\": \"alice\",\n      \"project_handle\": \"public-test\",\n      \"project_id\": 1,\n      \"instance_handle\": \"embedding1\",\n      \"text\": \"This is a similar test document\",\n      \"vector\": [\n        -0.020843506,\n        0.01852417,\n        0.05328369,\n        0.07141113,\n        0.020004272\n      ],\n      \"vector_dim\": 5,\n      \"metadata\": {\n        \"author\": \"Immanuel Other\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get document embeddings without authentication (public project)",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/public-test/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/Embeddings.json\",\n  \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n  \"user_handle\": \"alice\",\n  \"project_handle\": \"public-test\",\n  \"project_id\": 1,\n  \"instance_handle\": \"embedding1\",\n  \"text\": \"This is a test document\",\n  \"vector\": [\n    -0.020843506,\n    0.01852417,\n    0.05328369,\n    0.07141113,\n    0.020004272\n  ],\n  \"vector_dim\": 5,\n  \"metadata\": {\n    \"author\": \"Immanuel Kant\"\n  }\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get similars without authentication (public project)",
			method:       http.MethodGet,
			requestPath:  "/v1/similars/alice/public-test/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/SimilarResponseBody.json\",\n  \"user_handle\": \"alice\",\n  \"project_handle\": \"public-test\",\n  \"results\": [\n    {\n      \"id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n      \"similarity\": 1\n    },\n    {\n      \"id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\",\n      \"similarity\": 1\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Post embeddings without authentication (public project)",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/public-test",
			bodyPath:     "../../testdata/valid_embeddings.json",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {

			// We need to handle the body only for PUT and POST requests
			// For GET and DELETE requests, the body is nil
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
			requestURL := fmt.Sprintf("http://%v:%d%v", options.Host, options.Port, v.requestPath)
			req, err := http.NewRequest(v.method, requestURL, reqBody)
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+v.EmbAPIKey)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v\n", err)
			}
			// assert.NoError(t, err)
			defer resp.Body.Close()

			if resp.StatusCode != int(v.expectStatus) {
				t.Errorf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			} else {
				t.Logf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			}

			respBody, err := io.ReadAll(resp.Body) // response body is []byte
			assert.NoError(t, err)
			formattedResp := ""
			if v.expectBody != "" {
				fr := new(bytes.Buffer)
				err = json.Indent(fr, respBody, "", "  ")
				assert.NoError(t, err)
				formattedResp = fr.String()
			}
			// if (resp.StatusCode != http.StatusOK) || (resp.StatusCode != int(v.expectStatus)) {
			assert.Equal(t, v.expectBody, formattedResp, "they should be equal")
			// }
		})
	}

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

// Helper function to post embeddings
func postEmbeddings(t *testing.T, bodyPath, user, project, apiKey string) (string, error) {
	f, err := os.Open(bodyPath)
	if err != nil {
		fmt.Printf("%v", err)
	}
	defer f.Close()
	assert.NoError(t, err)

	b := new(bytes.Buffer)
	_, err = io.Copy(b, f)
	if err != nil {
		fmt.Printf("%v", err)
	}
	assert.NoError(t, err)

	requestURL := fmt.Sprintf("http://%v:%d/v1/embeddings/%s/%s", options.Host, options.Port, user, project)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(b.Bytes()))
	if err != nil {
		fmt.Printf("%v", err)
	}
	req.Header.Add("Authorization", "Bearer "+apiKey)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%v", err)
	}
	defer resp.Body.Close()
	assert.NoError(t, err)

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("status code %d: %s", resp.StatusCode, string(bodyBytes))
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Expected status code 201 Created")

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v", err)
	}
	assert.NoError(t, err)

	return string(bodyBytes), nil
}
