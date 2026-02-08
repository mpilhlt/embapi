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

func TestInstancesFunc(t *testing.T) {

	fmt.Printf("\n")

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create user to be used in instance tests
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	// Create API standard to be used in instance tests
	openaiJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, openaiJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	fmt.Printf("\nRunning llm-instances tests ...\n\n")

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
			name:         "Put instance, invalid json",
			method:       http.MethodPut,
			requestPath:  "/v1/llm-instances/alice/embedding1",
			bodyPath:     "../../testdata/invalid_instance.json",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"unexpected property\",\n      \"location\": \"body.api_keX\",\n      \"value\": {\n        \"api_keX\": \"0123456789\",\n        \"api_standard\": \"openai\",\n        \"description\": \"My OpenAI reduced text-embedding-3-large service\",\n        \"dimensions\": 99,\n        \"endpoint\": \"https://api.openai.com/v1/embeddings\",\n        \"instance_handle\": \"embedding1\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Put instance, wrong path",
			method:       http.MethodPut,
			requestPath:  "/v1/llm-instances/alice/nonexistent",
			bodyPath:     "../../testdata/valid_instance_embedding1.json",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"instance handle in URL (\\\"nonexistent\\\") does not match instance handle in body (\\\"embedding1\\\")\"\n}\n",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Valid put instance",
			method:       http.MethodPut,
			requestPath:  "/v1/llm-instances/alice/embedding1",
			bodyPath:     "../../testdata/valid_instance_embedding1.json",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadInstanceResponseBody.json\",\n  \"owner\": \"alice\",\n  \"instance_handle\": \"embedding1\",\n  \"instance_id\": 1\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Post instance, invalid json",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/alice",
			bodyPath:     "../../testdata/invalid_instance.json",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"unexpected property\",\n      \"location\": \"body.api_keX\",\n      \"value\": {\n        \"api_keX\": \"0123456789\",\n        \"api_standard\": \"openai\",\n        \"description\": \"My OpenAI reduced text-embedding-3-large service\",\n        \"dimensions\": 99,\n        \"endpoint\": \"https://api.openai.com/v1/embeddings\",\n        \"instance_handle\": \"embedding1\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Valid post instance",
			method:       http.MethodPost,
			requestPath:  "/v1/llm-instances/alice",
			bodyPath:     "../../testdata/valid_instance_embedding1.json",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadInstanceResponseBody.json\",\n  \"owner\": \"alice\",\n  \"instance_handle\": \"embedding1\",\n  \"instance_id\": 1\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Get all of alice's instances, admin's api key",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice",
			bodyPath:     "",
			EmbAPIKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetUserInstancesResponseBody.json\",\n  \"instances\": [\n    {\n      \"owner\": \"alice\",\n      \"instance_handle\": \"embedding1\",\n      \"instance_id\": 1\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get all of alice's instances, alice's api key",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice",
			bodyPath:     "",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetUserInstancesResponseBody.json\",\n  \"instances\": [\n    {\n      \"owner\": \"alice\",\n      \"instance_handle\": \"embedding1\",\n      \"instance_id\": 1\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get all llm-instances, unauthorized",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Get all llm-instances, nonexistent user",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/john",
			bodyPath:     "",
			EmbAPIKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user john not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Get nonexistent instance",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice/nonexistent",
			bodyPath:     "",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"llm service nonexistent for user alice not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Get single instance, nonexistent path",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice/nonexistant",
			bodyPath:     "",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"llm service nonexistant for user alice not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Valid get single instance",
			method:       http.MethodGet,
			requestPath:  "/v1/llm-instances/alice/embedding1",
			bodyPath:     "",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/InstanceFull.json\",\n  \"owner\": \"alice\",\n  \"instance_handle\": \"embedding1\",\n  \"instance_id\": 1,\n  \"access_role\": \"owner\",\n  \"endpoint\": \"https://api.foo.bar/v1/embed\",\n  \"description\": \"An LLM Service just for testing if the dhamps-vdb code is working\",\n  \"has_api_key\": true,\n  \"api_standard\": \"openai\",\n  \"model\": \"embed-test1\",\n  \"dimensions\": 5\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Delete nonexistent instance",
			method:       http.MethodDelete,
			requestPath:  "/v1/llm-instances/alice/nonexistent",
			bodyPath:     "",
			EmbAPIKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"llm service nonexistent for user alice not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Delete instance, invalid user",
			method:       http.MethodDelete,
			requestPath:  "/v1/llm-instances/john/embedding1",
			bodyPath:     "",
			EmbAPIKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user john not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Delete instance, unauthorized",
			method:       http.MethodDelete,
			requestPath:  "/v1/llm-instances/alice/embedding1",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid delete instance",
			method:       http.MethodDelete,
			requestPath:  "/v1/llm-instances/alice/embedding1",
			bodyPath:     "",
			EmbAPIKey:       aliceAPIKey,
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

	// Verify that the expectations regarding the mock key generation were met
	mockKeyGen.AssertExpectations(t)

	// Cleanup removes items created by the put function test
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
