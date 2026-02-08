package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/mpilhlt/dhamps-vdb/internal/handlers"
	"github.com/stretchr/testify/assert"
)

func TestAPIStandardFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Start the server
	err, shutDownServer := startTestServer(t, pool, handlers.StandardKeyGen{})
	assert.NoError(t, err)

	fmt.Printf("\nRunning api_standard tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		bodyPath     string
		VDBKey       string
		expectBody   string
		expectStatus int16
	}{
		{
			name:         "Put invalid API standard",
			method:       http.MethodPut,
			requestPath:  "/v1/api-standards/error1",
			bodyPath:     "../../testdata/invalid_api_standard.json",
			VDBKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"expected required property key_field to be present\",\n      \"location\": \"body\",\n      \"value\": {\n        \"api_standard_handle\": \"error1\",\n        \"description\": \"Erroneous definition of an APi standard\",\n        \"keX_method\": \"auth_bearer\"\n      }\n    },\n    {\n      \"message\": \"expected required property key_method to be present\",\n      \"location\": \"body\",\n      \"value\": {\n        \"api_standard_handle\": \"error1\",\n        \"description\": \"Erroneous definition of an APi standard\",\n        \"keX_method\": \"auth_bearer\"\n      }\n    },\n    {\n      \"message\": \"unexpected property\",\n      \"location\": \"body.keX_method\",\n      \"value\": {\n        \"api_standard_handle\": \"error1\",\n        \"description\": \"Erroneous definition of an APi standard\",\n        \"keX_method\": \"auth_bearer\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Put valid API standard, wrong path",
			method:       http.MethodPut,
			requestPath:  "/v1/api-standards/wrongpath",
			bodyPath:     "../../testdata/valid_api_standard_openai_v1.json",
			VDBKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"API standard handle in URL (wrongpath) does not match handle in body (openai).\"\n}\n",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Valid Put API standard",
			method:       http.MethodPut,
			requestPath:  "/v1/api-standards/test",
			bodyPath:     "../../testdata/valid_api_standard_test.json",
			VDBKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadAPIStandardResponseBody.json\",\n  \"api_standard_handle\": \"test\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "get all API standards",
			method:       http.MethodGet,
			requestPath:  "/v1/api-standards",
			VDBKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetAPIStandardsResponseBody.json\",\n  \"api_standards\": [\n    {\n      \"api_standard_handle\": \"cohere\",\n      \"description\": \"Cohere Embed API, Version 2, as documented in https://docs.cohere.com/reference/embed\",\n      \"key_method\": \"auth_bearer\",\n      \"key_field\": \"Authorization\"\n    },\n    {\n      \"api_standard_handle\": \"gemini\",\n      \"description\": \"Gemini Embeddings API, as documented in https://ai.google.dev/gemini-api/docs/embeddings\",\n      \"key_method\": \"auth_bearer\",\n      \"key_field\": \"x-goog-api-key\"\n    },\n    {\n      \"api_standard_handle\": \"openai\",\n      \"description\": \"OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings\",\n      \"key_method\": \"auth_bearer\",\n      \"key_field\": \"Authorization\"\n    },\n    {\n      \"api_standard_handle\": \"test\",\n      \"description\": \"OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings\",\n      \"key_method\": \"auth_bearer\",\n      \"key_field\": \"Authorization\"\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "get single API standard",
			method:       http.MethodGet,
			requestPath:  "/v1/api-standards/test",
			VDBKey:       "",
			expectBody:   "{\n  \"api_standard_handle\": \"test\",\n  \"description\": \"OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings\",\n  \"key_method\": \"auth_bearer\",\n  \"key_field\": \"Authorization\"\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Delete nonexistent path",
			method:       http.MethodDelete,
			requestPath:  "/v1/api-standards/wrongpath",
			VDBKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"API standard wrongpath not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "delete API standard",
			method:       http.MethodDelete,
			requestPath:  "/v1/api-standards/test",
			VDBKey:       options.AdminKey,
			expectStatus: http.StatusNoContent,
		},
		{
			name:         "post valid API standard",
			method:       http.MethodPut,
			requestPath:  "/v1/api-standards/openai",
			bodyPath:     "../../testdata/valid_api_standard_openai_v1.json",
			VDBKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/UploadAPIStandardResponseBody.json\",\n  \"api_standard_handle\": \"openai\"\n}\n",
			expectStatus: http.StatusCreated,
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
			if v.VDBKey != "" {
				req.Header.Set("Authorization", "Bearer "+v.VDBKey)
			}
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

	fmt.Printf("\n")
}
