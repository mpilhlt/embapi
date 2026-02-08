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

func TestProjectsFunc(t *testing.T) {

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
	instanceJSON := `{ "instance_handle": "embedding1", "endpoint": "https://api.foo.bar/v1/embed", "description": "An LLM Service just for testing if the dhamps-vdb code is working", "api_standard": "openai", "model": "embed-test1", "dimensions": 5}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating LLM service embedding1 for testing: %v\n", err)
	}

	fmt.Printf("\nRunning projects tests ...\n\n")

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
			name:         "Valid get all projects, no projects present, admin's api key",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjectsResponseBody.json\",\n  \"projects\": []\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Valid get all projects, no projects present, alice's api key",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjectsResponseBody.json\",\n  \"projects\": []\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Valid put project",
			method:       http.MethodPut,
			requestPath:  "/v1/projects/alice/test1",
			bodyPath:     "../../testdata/valid_project.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ProjectBrief.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"test1\",\n  \"project_id\": 1,\n  \"public_read\": false,\n  \"role\": \"owner\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Put project, invalid json",
			method:       http.MethodPut,
			requestPath:  "/v1/projects/alice/test2",
			bodyPath:     "../../testdata/invalid_project.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"expected required property project_handle to be present\",\n      \"location\": \"body\",\n      \"value\": {\n        \"description\": \"This is a test project\",\n        \"foo\": \"test1\"\n      }\n    },\n    {\n      \"message\": \"unexpected property\",\n      \"location\": \"body.foo\",\n      \"value\": {\n        \"description\": \"This is a test project\",\n        \"foo\": \"test1\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Put project, valid json but invalid project handle",
			method:       http.MethodPut,
			requestPath:  "/v1/projects/alice/test3",
			bodyPath:     "../../testdata/valid_project.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Bad Request\",\n  \"status\": 400,\n  \"detail\": \"project handle in URL (test3) does not match project handle in body (test1)\"\n}\n",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Valid post project",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice",
			bodyPath:     "../../testdata/valid_project.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ProjectBrief.json\",\n  \"owner\": \"alice\",\n  \"project_handle\": \"test1\",\n  \"project_id\": 1,\n  \"public_read\": false,\n  \"role\": \"owner\"\n}\n",
			expectStatus: http.StatusCreated,
		},
		{
			name:         "Post project, invalid json",
			method:       http.MethodPost,
			requestPath:  "/v1/projects/alice",
			bodyPath:     "../../testdata/invalid_project.json",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unprocessable Entity\",\n  \"status\": 422,\n  \"detail\": \"validation failed\",\n  \"errors\": [\n    {\n      \"message\": \"expected required property project_handle to be present\",\n      \"location\": \"body\",\n      \"value\": {\n        \"description\": \"This is a test project\",\n        \"foo\": \"test1\"\n      }\n    },\n    {\n      \"message\": \"unexpected property\",\n      \"location\": \"body.foo\",\n      \"value\": {\n        \"description\": \"This is a test project\",\n        \"foo\": \"test1\"\n      }\n    }\n  ]\n}\n",
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:         "Valid get project",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/test1",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ProjectFull.json\",\n  \"project_id\": 1,\n  \"project_handle\": \"test1\",\n  \"owner\": \"alice\",\n  \"description\": \"This is a test project\",\n  \"public_read\": false,\n  \"shared_with\": [\n    {\n      \"user_handle\": \"alice\",\n      \"role\": \"owner\"\n    }\n  ],\n  \"instance\": {\n    \"owner\": \"alice\",\n    \"instance_handle\": \"embedding1\",\n    \"instance_id\": 1,\n    \"access_role\": \"owner\"\n  },\n  \"role\": \"owner\",\n  \"number_of_embeddings\": 0\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Valid get all projects",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/GetProjectsResponseBody.json\",\n  \"projects\": [\n    {\n      \"owner\": \"alice\",\n      \"project_handle\": \"test1\",\n      \"project_id\": 1,\n      \"public_read\": false,\n      \"role\": \"owner\"\n    }\n  ]\n}\n",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get all projects, invalid user",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/john",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Get nonexistent project",
			method:       http.MethodGet,
			requestPath:  "/v1/projects/alice/test2",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user alice's project test2 not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Delete nonexistent project",
			method:       http.MethodDelete,
			requestPath:  "/v1/projects/alice/test2",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Not Found\",\n  \"status\": 404,\n  \"detail\": \"user alice's project test2 not found\"\n}\n",
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "Delete project, invalid user",
			method:       http.MethodDelete,
			requestPath:  "/v1/projects/john/test1",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid delete project",
			method:       http.MethodDelete,
			requestPath:  "/v1/projects/alice/test1",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
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
	// (deleting '/users/alice' should delete all the projects connected to alice as well)
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

// TestProjectTransactionRollback verifies that transactions are properly rolled back
// when an error occurs during project creation, ensuring no orphaned records.

/* We don't test transactions now because we don't link to authorized users in project creation ...
func TestProjectTransactionRollback(t *testing.T) {

	fmt.Printf("\n")

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator with unique keys for each user
	mockKeyGen := new(MockKeyGen)
	mockKeyGen.On("RandomKey", 32).Return("alicekey123456789012345678901", nil).Once()
	mockKeyGen.On("RandomKey", 32).Return("bobkey1234567890123456789012", nil).Once()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create user to be used in project tests
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	fmt.Printf("\nRunning project transaction rollback tests ...\n\n")

	t.Run("Project creation, invalid reader (full rollback)", func(t *testing.T) {
		// Attempt to create a project with a non-existent reader
		// This should fail during user validation and not even reach the transaction
		f, err := os.Open("../../testdata/project_with_invalid_reader.json")
		assert.NoError(t, err)
		defer f.Close()

		b := new(bytes.Buffer)
		_, err = io.Copy(b, f)
		assert.NoError(t, err)

		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/test-rollback", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodPut, requestURL, bytes.NewReader(b.Bytes()))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// The request should fail with 500 Internal Server Error
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
			"Expected 500 error when creating project with invalid reader")

		// TODO: Test against actual JSON body
		respBody, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(respBody), "unable to get user nonexistent_user",
			"Error message should mention the invalid user")

		// Verify that no project was created (transaction rolled back)
		// Try to get the project - it should not exist
		getURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/test-rollback", options.Host, options.Port)
		getReq, err := http.NewRequest(http.MethodGet, getURL, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+aliceAPIKey)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		// The project should not exist
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode,
			"Project should not exist after rollback")

		t.Log("Transaction rollback verified: no orphaned project record")
	})

	t.Run("Project creation, commit all", func(t *testing.T) {
		// Create a second user to be a reader
		bobJSON := `{"user_handle": "bob", "name": "Bob Smith", "email": "bob@foo.bar"}`
		_, err := createUser(t, bobJSON)
		assert.NoError(t, err)

		// Create a project with Bob as a reader
		projectJSON := `{"project_handle": "test-success", "description": "Test successful transaction", "shared_with": [{ "user_handle": "bob", "role": "reader"}]}`
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/test-success", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodPut, requestURL, bytes.NewReader([]byte(projectJSON)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// The request should succeed
		assert.Equal(t, http.StatusCreated, resp.StatusCode,
			"Expected 201 when creating project with valid reader")

		// Verify the project exists
		getURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/test-success", options.Host, options.Port)
		getReq, err := http.NewRequest(http.MethodGet, getURL, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+aliceAPIKey)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode,
			"Project should exist after successful transaction")

		// Verify the response includes both alice (owner) and bob (reader)
		respBody, err := io.ReadAll(getResp.Body)
		assert.NoError(t, err)

		var projectData map[string]interface{}
		err = json.Unmarshal(respBody, &projectData)
		assert.NoError(t, err)

		readers, ok := projectData["shared_with"].([]interface{})
		assert.True(t, ok, "shared_with should be an array")

		// Convert to string slice for easier checking
		readerStrings := make([]string, len(readers))
		for i, r := range readers {
			readerStrings[i] = r.(string)
		}

		// Check that alice (owner) and bob (reader) are both in the list
		assert.Contains(t, readerStrings, "alice", "Alice should be in authorized readers")
		assert.Contains(t, readerStrings, "bob", "Bob should be in authorized readers")

		t.Log("Transaction commit verified: all records created successfully")
	})

	// Cleanup
	t.Cleanup(func() {
		fmt.Print("\n\nRunning transaction test cleanup ...\n\n")

		requestURL := fmt.Sprintf("http://%s:%d/v1/admin/footgun", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+options.AdminKey)
		_, err = http.DefaultClient.Do(req)
		if err != nil && err.Error() != "no rows in result set" {
			t.Fatalf("Error sending request: %v\n", err)
		}

		fmt.Print("Shutting down server\n\n")
		shutDownServer()
	})

	fmt.Printf("\n")
}
*/
