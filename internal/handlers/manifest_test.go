package handlers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/mpilhlt/embapi/internal/handlers"
	"github.com/mpilhlt/embapi/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestManifestFunc(t *testing.T) {
	// Get the database connection pool from package variable
	pool := connPool

	// Start the server
	err, shutDownServer := startTestServer(t, pool, handlers.StandardKeyGen{})
	assert.NoError(t, err)

	fmt.Printf("\nRunning manifest tests ...\n\n")

	// Define test cases
	tt := []struct {
		name         string
		method       string
		requestPath  string
		expectStatus int
	}{
		{
			name:         "Get manifest at root",
			method:       http.MethodGet,
			requestPath:  "/",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Get manifest at /v1",
			method:       http.MethodGet,
			requestPath:  "/v1",
			expectStatus: http.StatusOK,
		},
	}

	for _, v := range tt {
		t.Run(v.name, func(t *testing.T) {
			requestURL := fmt.Sprintf("http://%v:%d%v", options.Host, options.Port, v.requestPath)
			req, err := http.NewRequest(v.method, requestURL, nil)
			assert.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v\n", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != v.expectStatus {
				t.Errorf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			} else {
				t.Logf("Expected status code %d, got %s\n", v.expectStatus, resp.Status)
			}

			respBody, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			// Verify the response structure
			var manifest models.ServiceManifest
			err = json.Unmarshal(respBody, &manifest)
			assert.NoError(t, err)

			// Check required fields
			assert.NotEmpty(t, manifest.Name, "name field should not be empty")
			assert.NotEmpty(t, manifest.Versions, "versions field should not be empty")
			assert.Contains(t, manifest.Versions, "v1", "versions should contain v1")

			// Check optional fields are present
			assert.NotEmpty(t, manifest.Description, "description field should be present")
			assert.NotEmpty(t, manifest.Documentation, "documentation field should be present")
			assert.Equal(t, "https://mpilhlt.github.io/embapi/", manifest.Documentation, "documentation URL should match")
			assert.NotEmpty(t, manifest.ServiceVersion, "serviceVersion field should be present")
			assert.NotEmpty(t, manifest.Authentication, "authentication field should be present")
			
			// Check endpoints
			assert.NotEmpty(t, manifest.Endpoints, "endpoints field should not be empty")
			
			// Verify some key endpoints are present
			foundUsers := false
			foundProjects := false
			foundEmbeddings := false
			foundSimilars := false
			
			for _, endpoint := range manifest.Endpoints {
				if endpoint.Path == "/v1/users" {
					foundUsers = true
					assert.Contains(t, endpoint.Methods, "GET")
					assert.Contains(t, endpoint.Methods, "POST")
				}
				if endpoint.Path == "/v1/projects/{user_handle}/{project_handle}" {
					foundProjects = true
				}
				if endpoint.Path == "/v1/embeddings/{user_handle}/{project_handle}" {
					foundEmbeddings = true
				}
				if endpoint.Path == "/v1/similars/{user_handle}/{project_handle}/{text_id}" {
					foundSimilars = true
				}
			}
			
			assert.True(t, foundUsers, "users endpoint should be in the manifest")
			assert.True(t, foundProjects, "projects endpoint should be in the manifest")
			assert.True(t, foundEmbeddings, "embeddings endpoint should be in the manifest")
			assert.True(t, foundSimilars, "similars endpoint should be in the manifest")

			t.Logf("Manifest contains %d endpoints\n", len(manifest.Endpoints))
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
