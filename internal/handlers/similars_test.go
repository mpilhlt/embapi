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

func TestSimilarsFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create user
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	// Create API standard
	apiStandardJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, apiStandardJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create LLM Service
	InstanceJSON := `{ "instance_handle": "embedding1", "endpoint": "https://api.foo.bar/v1/embed", "description": "An LLM Service just for testing if the embapi code is working", "api_standard": "openai", "model": "embed-test1", "dimensions": 5}`
	_, err = createInstance(t, InstanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating LLM service openai-large for testing: %v\n", err)
	}

	// Create project
	projectJSON := `{"project_handle": "test1", "description": "A test project", "instance_owner": "alice", "instance_handle": "embedding1"}`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project alice/test1 for testing: %v\n", err)
	}

	// Upload embeddings
	embeddingsFilePath := "../../testdata/valid_embeddings.json"
	embeddingsFile, err := os.Open(embeddingsFilePath)
	if err != nil {
		t.Fatalf("Error opening embeddings file: %v\n", err)
	}
	// Defer closing embeddingsFile
	defer func() {
		if err := embeddingsFile.Close(); err != nil {
			t.Fatalf("Error closing embeddings file: %v\n", err)
		}
	}()
	embeddingsData, err := io.ReadAll(embeddingsFile)
	if err != nil {
		t.Fatalf("Error reading embeddings file: %v\n", err)
	}
	err = createEmbeddings(t, embeddingsData, "alice", "test1", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating embeddings for testing: %v\n", err)
	}

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
			name:         "Get similar passages, no parameters",
			method:       http.MethodGet,
			requestPath:  "/v1/similars/alice/test1/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
			bodyPath:     "",
			apiKey:       aliceAPIKey,
			expectBody:   "", // Will validate structure programmatically
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get similar passages, with filter",
			method:       http.MethodGet,
			requestPath:  "/v1/similars/alice/test1/https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1?metadata_path=author&metadata_value=Immanuel%20Kant",
			bodyPath:     "",
			apiKey:       options.AdminKey,
			expectBody:   "", // Will validate structure programmatically
			expectStatus: http.StatusOK,
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

			// Parse and validate JSON structure
			if v.expectBody == "" && resp.StatusCode == http.StatusOK {
				// Validate that response has correct structure with results array
				var result map[string]interface{}
				err = json.Unmarshal(respBody, &result)
				assert.NoError(t, err)

				// Check results field exists and is an array
				results, ok := result["results"].([]interface{})
				if !ok {
					t.Errorf("Response does not contain results array")
				} else if len(results) == 0 {
					t.Errorf("Results array is empty")
				} else {
					// Verify each result has id and similarity fields
					for i, r := range results {
						resultItem, ok := r.(map[string]interface{})
						if !ok {
							t.Errorf("Result item %d is not an object", i)
							continue
						}
						if _, hasID := resultItem["id"]; !hasID {
							t.Errorf("Result item %d missing 'id' field", i)
						}
						if similarity, hasSim := resultItem["similarity"]; !hasSim {
							t.Errorf("Result item %d missing 'similarity' field", i)
						} else if _, ok := similarity.(float64); !ok {
							t.Errorf("Result item %d 'similarity' is not a number", i)
						}
					}
					t.Logf("Found %d results with similarity scores", len(results))
				}
			} else if v.expectBody != "" {
				formattedResp := ""
				fr := new(bytes.Buffer)
				err = json.Indent(fr, respBody, "", "  ")
				assert.NoError(t, err)
				formattedResp = fr.String()
				assert.Equal(t, v.expectBody, formattedResp, "they should be equal")
			}
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

// TestPostSimilar tests the POST similar functionality.
func TestPostSimilar(t *testing.T) {
	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	// Set up expectations for the mock key generator
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

	// Start the server
	err, shutDownServer := startTestServer(t, pool, mockKeyGen)
	assert.NoError(t, err)

	// Create user
	aliceJSON := `{"user_handle": "alice", "name": "Alice Doe", "email": "alice@foo.bar"}`
	aliceAPIKey, err := createUser(t, aliceJSON)
	if err != nil {
		t.Fatalf("Error creating user alice for testing: %v\n", err)
	}

	// Create API standard
	apiStandardJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, apiStandardJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create LLM Service Instance with 5 dimensions
	InstanceJSON := `{ "instance_handle": "embedding1", "endpoint": "https://api.foo.bar/v1/embed", "description": "An LLM Service just for testing if the embapi code is working", "api_standard": "openai", "model": "embed-test1", "dimensions": 5}`
	_, err = createInstance(t, InstanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating LLM service embedding1 for testing: %v\n", err)
	}

	// Create project
	projectJSON := `{"project_handle": "test1", "description": "A test project", "instance_owner": "alice", "instance_handle": "embedding1"}`
	_, err = createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project alice/test1 for testing: %v\n", err)
	}

	// Create another project without an instance
	projectNoInstanceJSON := `{"project_handle": "test2", "description": "A test project without instance"}`
	_, err = createProject(t, projectNoInstanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project alice/test2 for testing: %v\n", err)
	}

	// Upload embeddings
	embeddingsFilePath := "../../testdata/valid_embeddings.json"
	embeddingsFile, err := os.Open(embeddingsFilePath)
	if err != nil {
		t.Fatalf("Error opening embeddings file: %v\n", err)
	}
	defer func() {
		if err := embeddingsFile.Close(); err != nil {
			t.Fatalf("Error closing embeddings file: %v\n", err)
		}
	}()
	embeddingsData, err := io.ReadAll(embeddingsFile)
	if err != nil {
		t.Fatalf("Error reading embeddings file: %v\n", err)
	}
	err = createEmbeddings(t, embeddingsData, "alice", "test1", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating embeddings for testing: %v\n", err)
	}

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		body         string
		apiKey       string
		expectStatus int
		expectIDs    []string
		expectError  bool
	}{
		{
			name:         "POST similar with valid 5D vector",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test1",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusOK,
			expectIDs: []string{
				"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.1.1.1.1",
				"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol1.2",
				"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2",
			},
			expectError: false,
		},
		{
			name:         "POST similar with valid 5D vector and metadata filter",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test1?metadata_path=author&metadata_value=Immanuel%20Kant",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusOK,
			expectIDs: []string{
				"https%3A%2F%2Fid.salamanca.school%2Ftexts%2FW0001%3Avol2",
			},
			expectError: false,
		},
		{
			name:         "POST similar with wrong dimension (3D instead of 5D)",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test1",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:         "POST similar with wrong dimension (7D instead of 5D)",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test1",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308, 0.01, 0.02]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:         "POST similar to nonexistent project",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/nonexistent",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusNotFound,
			expectError:  true,
		},
		{
			name:         "POST similar to project without instance",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test2",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:         "POST similar with missing metadata_value",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test1?metadata_path=author",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusBadRequest,
			expectError:  true,
		},
		{
			name:         "POST similar with missing metadata_path",
			method:       http.MethodPost,
			requestPath:  "/v1/similars/alice/test1?metadata_value=Immanuel%20Kant",
			body:         `{"vector": [-0.02085085, 0.01852216, 0.05327000, 0.07138438, 0.02000308]}`,
			apiKey:       aliceAPIKey,
			expectStatus: http.StatusBadRequest,
			expectError:  true,
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {
			requestURL := fmt.Sprintf("http://%v:%d%v", options.Host, options.Port, v.requestPath)
			req, err := http.NewRequest(v.method, requestURL, bytes.NewBufferString(v.body))
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+v.apiKey)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v\n", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != v.expectStatus {
				t.Errorf("Expected status code %d, got %d\n", v.expectStatus, resp.StatusCode)
			} else {
				t.Logf("Expected status code %d, got %d\n", v.expectStatus, resp.StatusCode)
			}

			respBody, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if !v.expectError {
				// Parse response
				var result map[string]interface{}
				err = json.Unmarshal(respBody, &result)
				assert.NoError(t, err)

				// Check that we got the expected structure with results array
				results, ok := result["results"].([]interface{})
				if !ok {
					t.Errorf("Response does not contain results array")
				} else {
					// Convert interface{} slice to string slice for IDs
					actualIDs := make([]string, len(results))
					for i, r := range results {
						resultItem, ok := r.(map[string]interface{})
						if !ok {
							t.Errorf("Result item %d is not an object", i)
							continue
						}
						id, hasID := resultItem["id"]
						if !hasID {
							t.Errorf("Result item %d missing 'id' field", i)
							continue
						}
						actualIDs[i] = id.(string)

						// Verify similarity field exists and is a number
						similarity, hasSim := resultItem["similarity"]
						if !hasSim {
							t.Errorf("Result item %d missing 'similarity' field", i)
						} else if _, ok := similarity.(float64); !ok {
							t.Errorf("Result item %d 'similarity' is not a number", i)
						}
					}

					// Check that all expected IDs are present (order doesn't matter for similar items)
					for _, expectedID := range v.expectIDs {
						found := false
						for _, actualID := range actualIDs {
							if actualID == expectedID {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Expected ID %s not found in response", expectedID)
						}
					}
				}
			}
		})
	}

	// Verify that the expectations regarding the mock key generation were met
	mockKeyGen.AssertExpectations(t)

	// Cleanup removes items created by the test
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
