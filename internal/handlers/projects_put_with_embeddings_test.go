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

// TestPutProjectWithEmbeddings tests that PUT returns correct number_of_embeddings count
// This test verifies the fix for the issue where PUT was returning 0 embeddings
// even when embeddings existed in the project
func TestPutProjectWithEmbeddings(t *testing.T) {

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

	// Create API standard
	apiStandardJSON := `{"api_standard_handle": "openai", "description": "OpenAI Embeddings API", "key_method": "auth_bearer", "key_field": "Authorization" }`
	_, err = createAPIStandard(t, apiStandardJSON, options.AdminKey)
	if err != nil {
		t.Fatalf("Error creating API standard openai for testing: %v\n", err)
	}

	// Create LLM Service Instance
	instanceJSON := `{ "instance_handle": "embedding1", "endpoint": "https://api.foo.bar/v1/embed", "description": "An LLM Service just for testing", "api_standard": "openai", "model": "embed-test1", "dimensions": 5}`
	_, err = createInstance(t, instanceJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating LLM service embedding1 for testing: %v\n", err)
	}

	fmt.Printf("\nRunning PUT project with embeddings test ...\n\n")

	// Step 1: Create a project
	projectJSON := `{"project_handle": "test1", "description": "This is a test project", "instance_owner": "alice", "instance_handle": "embedding1"}`
	projectID, err := createProject(t, projectJSON, "alice", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error creating project: %v\n", err)
	}
	fmt.Printf("Created project with ID: %d\n", projectID)

	// Step 2: Upload embeddings to the project
	embeddingsFile, err := os.ReadFile("../../testdata/valid_embeddings.json")
	if err != nil {
		t.Fatalf("Error reading embeddings file: %v\n", err)
	}
	err = createEmbeddings(t, embeddingsFile, "alice", "test1", aliceAPIKey)
	if err != nil {
		t.Fatalf("Error uploading embeddings: %v\n", err)
	}
	fmt.Printf("Uploaded embeddings to project\n")

	// Step 3: Verify GET returns correct count
	getURL := fmt.Sprintf("http://%v:%d/v1/projects/alice/test1", options.Host, options.Port)
	req, err := http.NewRequest(http.MethodGet, getURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+aliceAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var getResponse map[string]interface{}
	err = json.Unmarshal(body, &getResponse)
	assert.NoError(t, err)

	embeddingCount := int(getResponse["number_of_embeddings"].(float64))
	fmt.Printf("GET returned number_of_embeddings: %d\n", embeddingCount)
	assert.Equal(t, 3, embeddingCount, "GET should return 3 embeddings")

	// Step 4: Update project with PUT (change description)
	updatedProjectJSON := `{"project_handle": "test1", "description": "This is an updated test project", "instance_owner": "alice", "instance_handle": "embedding1"}`
	putURL := fmt.Sprintf("http://%v:%d/v1/projects/alice/test1", options.Host, options.Port)
	req, err = http.NewRequest(http.MethodPut, putURL, bytes.NewBuffer([]byte(updatedProjectJSON)))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var putResponse map[string]interface{}
	err = json.Unmarshal(body, &putResponse)
	assert.NoError(t, err)

	embeddingCountPut := int(putResponse["number_of_embeddings"].(float64))
	fmt.Printf("PUT returned number_of_embeddings: %d\n", embeddingCountPut)
	
	// This is the key assertion: PUT should now return the correct count
	assert.Equal(t, 3, embeddingCountPut, "PUT should return 3 embeddings (same as GET)")

	// Step 5: Verify GET still returns correct count after PUT
	req, err = http.NewRequest(http.MethodGet, getURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+aliceAPIKey)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	assert.NoError(t, err)

	err = json.Unmarshal(body, &getResponse)
	assert.NoError(t, err)

	embeddingCountAfter := int(getResponse["number_of_embeddings"].(float64))
	fmt.Printf("GET after PUT returned number_of_embeddings: %d\n", embeddingCountAfter)
	assert.Equal(t, 3, embeddingCountAfter, "GET after PUT should still return 3 embeddings")

	fmt.Printf("\nRunning cleanup ...\n\n")

	// Cleanup - reset database
	footgunURL := fmt.Sprintf("http://%s:%d/v1/admin/footgun", options.Host, options.Port)
	req, err = http.NewRequest(http.MethodGet, footgunURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+options.AdminKey)
	_, err = client.Do(req)
	if err != nil && err.Error() != "no rows in result set" {
		t.Fatalf("Error resetting database: %v\n", err)
	}

	shutDownServer()
}
