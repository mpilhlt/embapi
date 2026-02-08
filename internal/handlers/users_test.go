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

func TestUserFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	fmt.Printf("\nRunning users tests ...\n\n")

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
			name:         "Put user, everything valid",
			method:       http.MethodPut,
			requestPath:  "/v1/users/alice",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UserResponse.json\",\n  \"user_handle\": \"alice\",\n  \"vdb_key\": \"12345678901234567890123456789012\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Valid get user",
			method:       http.MethodGet,
			requestPath:  "/v1/users/alice",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"user_handle\": \"alice\",\n  \"name\": \"Alice Doe\",\n  \"email\": \"alice@foo.bar\",\n  \"vdb_key\": \"e1b85b27d6bcb05846c18e6a48f118e89f0c0587140de9fb3359f8370d0dba08\"\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Put user, invalid API key",
			method:       http.MethodPut,
			requestPath:  "/v1/users/alice",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       "not-the-admin-key",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Put user, no API key",
			method:       http.MethodPut,
			requestPath:  "/v1/users/alice",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Put user, invalid json",
			method:       http.MethodPut,
			requestPath:  "/v1/users/john",
			bodyPath:     "../../testdata/invalid_user.json",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"expected required property email to be present\",\n      \"location\": \"body\",\n      \"value\": {\n        \"name\": \"John Doe\",\n        \"user_handle\": \"john\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Put user, valid json but invalid user handle",
			method:       http.MethodPut,
			requestPath:  "/v1/users/bob",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"user handle in URL (bob) does not match user handle in body (alice).\"\n}\n",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Post existing user, everything valid",
			method:       http.MethodPost,
			requestPath:  "/v1/users",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UserResponse.json\",\n  \"user_handle\": \"alice\",\n  \"vdb_key\": \"not changed\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Post user, invalid API key",
			method:       http.MethodPost,
			requestPath:  "/v1/users",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       "not-the-admin-key",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Post user, no API key",
			method:       http.MethodPost,
			requestPath:  "/v1/users",
			bodyPath:     "../../testdata/valid_user.json",
			apiKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Post user, invalid json",
			method:       http.MethodPost,
			requestPath:  "/v1/users",
			bodyPath:     "../../testdata/invalid_user.json",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"expected required property email to be present\",\n      \"location\": \"body\",\n      \"value\": {\n        \"name\": \"John Doe\",\n        \"user_handle\": \"john\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Valid get all users",
			method:       http.MethodGet,
			requestPath:  "/v1/users",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "[\n  \"alice\",\n  \"_system\"\n]\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get nonexistent user",
			method:       http.MethodGet,
			requestPath:  "/v1/users/alfons",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user alfons not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Get invalid path",
			method:       http.MethodGet,
			requestPath:  "/v1/uxers/alfons",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Delete user, valid path",
			method:       http.MethodDelete,
			requestPath:  "/v1/users/alice",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "",
			expectStatus: http.StatusNoContent,
		},
		{
			name:         "Delete nonexistent user",
			method:       http.MethodDelete,
			requestPath:  "/v1/users/alfons",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user alfons not found\"\n}\n",
			expectStatus: http.StatusNotFound,
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
			req.Header.Set("Authorization", "Bearer "+v.apiKey)
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
	// ('/users/alice' and '/users/bob' in case this has erroneously been created)
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
