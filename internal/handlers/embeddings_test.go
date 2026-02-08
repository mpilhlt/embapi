package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmbeddingsFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create user to be used in project tests
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	// Create API standard to be used in embeddings tests
	apiStandardJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, apiStandardJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create LLM Service Instance to be used in embeddings tests
	instanceJSON := `{ "instance_handle": "embedding1", "endpoint": "https://api.foo.bar/v1/embed", "description": "An LLM Service just for testing if the embapi code is working", "api_standard": "openai", "model": "embed-test1", "dimensions": 5}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating LLM service embedding1 for testing: %v\n", err)
	}

	// Create project to be used in embeddings tests
	projectJSON := `{ "project_handle": "test1", "instance_owner": "alice", "instance_handle": "embedding1", "description": "This is a test project" }`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project alice/test1 for testing: %v\n", err)
	}

	fmt.Printf("\nRunning embeddings tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyPath     string
		apiKeyHeader string
		expectBody   string
		expectStatus int16
	}{
		{
			name:         "Post embeddings, invalid json",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/invalid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"expected required property text_id to be present\",\n      \"location\": \"body.embeddings[0]\",\n      \"value\": {\n        \"foo\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n        \"instance_handle\": \"embedding1\",\n        \"instance_owner\": \"alice\",\n        \"metadata\": {\n          \"author\": \"Immanuel Kant\"\n        },\n        \"project_handle\": \"test1\",\n        \"text\": \"This is a test document\",\n        \"user_handle\": \"alice\",\n        \"vector\": [],\n        \"vector_dim\": 10\n      }\n    },\n    {\n      \"message\": \"unexpected property\",\n      \"location\": \"body.embeddings[0].foo\",\n      \"value\": {\n        \"foo\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n        \"instance_handle\": \"embedding1\",\n        \"instance_owner\": \"alice\",\n        \"metadata\": {\n          \"author\": \"Immanuel Kant\"\n        },\n        \"project_handle\": \"test1\",\n        \"text\": \"This is a test document\",\n        \"user_handle\": \"alice\",\n        \"vector\": [],\n        \"vector_dim\": 10\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Post embeddings, unauthorized",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid post embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadProjEmbeddingsResponseBody.json\",\n  \"ids\": [\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Get project embeddings, wrong path",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/testX",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user alice's project testX not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Get project embeddings, unauthorized",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "",
			apiKeyHeader: "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid get project embeddings",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjEmbeddingsResponseBody.json\",\n  \"embeddings\": [\n    {\n      \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n      \"user_handle\": \"alice\",\n      \"project_handle\": \"test1\",\n      \"project_id\": 1,\n      \"instance_handle\": \"embedding1\",\n      \"text\": \"This is a test document\",\n      \"vector\": [\n        -0.020843506,\n        0.01852417,\n        0.05328369,\n        0.07141113,\n        0.020004272\n      ],\n      \"vector_dim\": 5,\n      \"metadata\": {\n        \"author\": \"Immanuel Kant\"\n      }\n    },\n    {\n      \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n      \"user_handle\": \"alice\",\n      \"project_handle\": \"test1\",\n      \"project_id\": 1,\n      \"instance_handle\": \"embedding1\",\n      \"text\": \"This is a similar test document\",\n      \"vector\": [\n        -0.020843506,\n        0.01852417,\n        0.05328369,\n        0.07141113,\n        0.020004272\n      ],\n      \"vector_dim\": 5,\n      \"metadata\": {\n        \"author\": \"Immanuel Kant\"\n      }\n    },\n    {\n      \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\",\n      \"user_handle\": \"alice\",\n      \"project_handle\": \"test1\",\n      \"project_id\": 1,\n      \"instance_handle\": \"embedding1\",\n      \"text\": \"This is a similar test document\",\n      \"vector\": [\n        -0.020843506,\n        0.01852417,\n        0.05328369,\n        0.07141113,\n        0.020004272\n      ],\n      \"vector_dim\": 5,\n      \"metadata\": {\n        \"author\": \"Immanuel Other\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get document embeddings, wrong path",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/test1/nonexistent",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"no embeddings found for user alice, project test1, id nonexistent.\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Get document embeddings, unauthorized",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/test1/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			apiKeyHeader: "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid get document embeddings",
			method:       http.MethodGet,
			requestPath:  "/v1/embeddings/alice/test1/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/Embeddings.json\",\n  \"text_id\": \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n  \"user_handle\": \"alice\",\n  \"project_handle\": \"test1\",\n  \"project_id\": 1,\n  \"instance_handle\": \"embedding1\",\n  \"text\": \"This is a test document\",\n  \"vector\": [\n    -0.020843506,\n    0.01852417,\n    0.05328369,\n    0.07141113,\n    0.020004272\n  ],\n  \"vector_dim\": 5,\n  \"metadata\": {\n    \"author\": \"Immanuel Kant\"\n  }\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Delete document embeddings, wrong path",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test1/nonexistent",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"no embeddings found for user alice, project test1, id nonexistent.\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Valid post embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadProjEmbeddingsResponseBody.json\",\n  \"ids\": [\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Delete document embeddings, unauthorized",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test1/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			apiKeyHeader: "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid post embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadProjEmbeddingsResponseBody.json\",\n  \"ids\": [\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Delete all project embeddings, wrong path",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/nonexistant",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"alice's project nonexistant not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Valid post embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadProjEmbeddingsResponseBody.json\",\n  \"ids\": [\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Delete all projet embeddings, unauthorized",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "",
			apiKeyHeader: "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid post embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadProjEmbeddingsResponseBody.json\",\n  \"ids\": [\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Valid delete document embeddings",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test1/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "",
			expectStatus: http.StatusNoContent,
		},
		{
			name:         "Valid post embeddings",
			method:       http.MethodPost,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "../../testdata/valid_embeddings.json",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadProjEmbeddingsResponseBody.json\",\n  \"ids\": [\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2\",\n    \"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2\"\n  ]\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Valid delete all project embeddings",
			method:       http.MethodDelete,
			requestPath:  "/v1/embeddings/alice/test1",
			bodyPath:     "",
			apiKeyHeader: aliceAPIKey,
			expectBody:   "",
			expectStatus: http.StatusNoContent,
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
			req.Header.Add("Authorization", "Bearer "+v.apiKeyHeader)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v\n", err)
			}
			assert.NoError(t, err)
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
				if respBody == nil {
					t.Errorf("Expected body %s, got nil\n", v.expectBody)
				} else {
					fr := new(bytes.Buffer)
					if isJSON(string(respBody)) && (strings.Contains(string(respBody), "{") || strings.Contains(string(respBody), "[")) {
						err = json.Indent(fr, respBody, "", "  ")
						// fmt.Printf("Error: %v\nresponse: %v\n", err, string(respBody))
						assert.NoError(t, err)
						formattedResp = fr.String()
					} else {
						formattedResp = string(respBody)
					}
				}
			}
			// if (resp.StatusCode != http.StatusOK) || (resp.StatusCode != int(v.expectStatus)) {
			assert.Equal(t, v.expectBody, formattedResp, "they should be equal")
			// }
		})
	}

	// Verify that the expectations regarding the mock key generation were met
	mockKeyGen.AssertExpectations(t)

	// Cleanup removes items created by the put function test
	// (deleting '/users/alice' should delete all the
	//  projects, instances and embeddings connected to alice as well)
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
