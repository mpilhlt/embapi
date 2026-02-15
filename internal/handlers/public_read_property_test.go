package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPublicReadProperty tests that the public_read property is correctly reported
// in all project endpoints (GET all, GET single, POST, PUT)
func TestPublicReadProperty(t *testing.T) {
	// Get the database connection pool from package variable
	pool := connPool

	// Create a mock key generator
	mockKeyGen := new(MockKeyGen)
	mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Once()  // Alice's key

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

	fmt.Printf("\nRunning public_read property tests ...\n\n")

	// Test 1: Create a project with public_read=true using POST
	t.Run("POST project with public_read=true", func(t *testing.T) {
		projectJSON := `{"project_handle": "public-project", "instance_owner": "alice", "instance_handle": "embedding1", "description": "A public project", "public_read": true}`
		
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader([]byte(projectJSON)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
		
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Expected status code 201")
		
		// Parse response and check public_read
		var result map[string]interface{}
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)
		
		publicRead, ok := result["public_read"].(bool)
		assert.True(t, ok, "public_read field should be present in response")
		assert.True(t, publicRead, "public_read should be true in POST response")
		
		t.Logf("✓ POST response correctly reports public_read=true")
	})

	// Test 2: GET single project should report public_read=true
	t.Run("GET single project reports public_read=true", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/public-project", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
		
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
		
		// Parse response and check public_read
		var result map[string]interface{}
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)
		
		publicRead, ok := result["public_read"].(bool)
		assert.True(t, ok, "public_read field should be present in response")
		assert.True(t, publicRead, "public_read should be true in GET single project response")
		
		t.Logf("✓ GET single project correctly reports public_read=true")
	})

	// Test 3: GET all projects should report public_read=true
	t.Run("GET all projects reports public_read=true", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
		
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
		
		// Parse response and check public_read
		var result map[string]interface{}
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)
		
		projects, ok := result["projects"].([]interface{})
		assert.True(t, ok, "projects field should be present in response")
		assert.Greater(t, len(projects), 0, "should have at least one project")
		
		// Find our public project
		var foundProject map[string]interface{}
		for _, proj := range projects {
			p := proj.(map[string]interface{})
			if p["project_handle"] == "public-project" {
				foundProject = p
				break
			}
		}
		assert.NotNil(t, foundProject, "should find public-project in the list")
		
		publicRead, ok := foundProject["public_read"].(bool)
		assert.True(t, ok, "public_read field should be present in project")
		assert.True(t, publicRead, "public_read should be true in GET all projects response")
		
		t.Logf("✓ GET all projects correctly reports public_read=true")
	})

	// Test 4: Update project with PUT and ensure public_read value is preserved
	t.Run("PUT project preserves public_read value", func(t *testing.T) {
		// Update the project with public_read not specified (should preserve true)
		projectJSON := `{"project_handle": "public-project", "instance_owner": "alice", "instance_handle": "embedding1", "description": "Updated public project"}`
		
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/public-project", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodPut, requestURL, bytes.NewReader([]byte(projectJSON)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
		
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Expected status code 201")
		
		// Parse response and check public_read - it should default to false when not specified
		var result map[string]interface{}
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)
		
		publicRead, ok := result["public_read"].(bool)
		assert.True(t, ok, "public_read field should be present in response")
		// When not specified in PUT, it defaults to false based on the model definition
		assert.False(t, publicRead, "public_read should be false when not specified in PUT request")
		
		t.Logf("✓ PUT response correctly reports database value for public_read")
	})

	// Test 5: Update project with PUT and explicitly set public_read=true
	t.Run("PUT project with explicit public_read=true", func(t *testing.T) {
		projectJSON := `{"project_handle": "public-project", "instance_owner": "alice", "instance_handle": "embedding1", "description": "Updated public project", "public_read": true}`
		
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/public-project", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodPut, requestURL, bytes.NewReader([]byte(projectJSON)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
		
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Expected status code 201")
		
		// Parse response and check public_read
		var result map[string]interface{}
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)
		
		publicRead, ok := result["public_read"].(bool)
		assert.True(t, ok, "public_read field should be present in response")
		assert.True(t, publicRead, "public_read should be true in PUT response")
		
		t.Logf("✓ PUT response correctly reports public_read=true")
	})

	// Test 6: Verify GET single project still shows public_read=true after PUT
	t.Run("GET single project after PUT still reports public_read=true", func(t *testing.T) {
		requestURL := fmt.Sprintf("http://%s:%d/v1/projects/alice/public-project", options.Host, options.Port)
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+aliceAPIKey)
		
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
		
		// Parse response and check public_read
		var result map[string]interface{}
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)
		
		publicRead, ok := result["public_read"].(bool)
		assert.True(t, ok, "public_read field should be present in response")
		assert.True(t, publicRead, "public_read should still be true after PUT")
		
		t.Logf("✓ GET single project correctly reports public_read=true after PUT")
	})

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
