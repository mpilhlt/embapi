package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/mpilhlt/embapi/internal/handlers"
	"github.com/stretchr/testify/assert"
)

func TestAdminFunc(t *testing.T) {

	// Get the database connection pool from package variable
	pool := connPool

	// Start the server
	err, shutDownServer := startTestServer(t, pool, handlers.StandardKeyGen{})
	assert.NoError(t, err)

	fmt.Printf("\nRunning admin tests ...\n\n")

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
			name:         "Reset database, unauthorized",
			method:       http.MethodGet,
			requestPath:  "/v1/admin/footgun",
			bodyPath:     "",
			EmbAPIKey:       "",
			expectBody:   "{\n  \"$schema\": \"http://localhost:8080/schemas/ErrorModel.json\",\n  \"title\": \"Unauthorized\",\n  \"status\": 401,\n  \"detail\": \"Authentication failed. Perhaps a missing or incorrect API key?\"\n}\n",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Valid reset database",
			method:       http.MethodGet,
			requestPath:  "/v1/admin/footgun",
			bodyPath:     "",
			EmbAPIKey:       options.AdminKey,
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

	t.Cleanup(func() {
		fmt.Print("\nRunning cleanup ...\n\n")
		fmt.Print("Shutting down server\n\n")
		shutDownServer()
	})

	fmt.Printf("\n")

}
