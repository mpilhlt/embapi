package handlers_test

import (
"encoding/json"
"fmt"
"io"
"net/http"
"testing"

"github.com/stretchr/testify/assert"
)

func TestPaginationForGetUsers(t *testing.T) {
// Get the database connection pool from package variable
pool := connPool

// Create a mock key generator
mockKeyGen := new(MockKeyGen)
mockKeyGen.On("RandomKey", 32).Return("12345678901234567890123456789012", nil).Maybe()

// Start the server
err, shutDownServer := startTestServer(t, pool, mockKeyGen)
assert.NoError(t, err)
defer shutDownServer()

fmt.Printf("\nRunning pagination tests ...\n\n")

// Test pagination: limit=1, offset=0 (should get first user)
req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/v1/users?limit=1&offset=0", options.Port), nil)
assert.NoError(t, err)
req.Header.Set("Authorization", "Bearer "+options.AdminKey)

client := &http.Client{}
resp, err := client.Do(req)
assert.NoError(t, err)
defer resp.Body.Close()

assert.Equal(t, http.StatusOK, resp.StatusCode)

bodyBytes, err := io.ReadAll(resp.Body)
assert.NoError(t, err)

var userList []string
err = json.Unmarshal(bodyBytes, &userList)
assert.NoError(t, err)
assert.Equal(t, 1, len(userList), "Expected exactly 1 user with limit=1")

fmt.Printf("First page (limit=1, offset=0): %v\n", userList)

// Test getting all users with high limit
req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/v1/users?limit=100", options.Port), nil)
assert.NoError(t, err)
req.Header.Set("Authorization", "Bearer "+options.AdminKey)

resp3, err := client.Do(req)
assert.NoError(t, err)
defer resp3.Body.Close()

assert.Equal(t, http.StatusOK, resp3.StatusCode)

bodyBytes, err = io.ReadAll(resp3.Body)
assert.NoError(t, err)

err = json.Unmarshal(bodyBytes, &userList)
assert.NoError(t, err)
// We should have at least the _system user
assert.GreaterOrEqual(t, len(userList), 1, "Expected at least 1 user")

fmt.Printf("All users (limit=100): %d users found\n", len(userList))

fmt.Printf("\nPagination tests completed successfully!\n\n")
}
