---
title: "Testing"
weight: 1
---

# Testing Guide

This guide covers how to run and write tests for embapi.

## Running Tests

embapi uses integration tests that spin up real PostgreSQL containers using [testcontainers](https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/). This approach ensures tests run against actual database instances with pgvector support.

### Prerequisites

**Using Podman (Recommended for Linux):**

```bash
# Start podman socket
systemctl --user start podman.socket

# Export DOCKER_HOST for testcontainers
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
```

**Using Docker:**

Testcontainers works with Docker out of the box. Ensure Docker daemon is running.

### Running All Tests

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests without verbose output
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Running Specific Tests

```bash
# Run tests in a specific package
go test -v ./internal/handlers

# Run a specific test function
go test -v ./internal/handlers -run TestCreateUser

# Run tests matching a pattern
go test -v ./... -run ".*Sharing.*"
```

### Test Containers

Tests automatically manage PostgreSQL containers with pgvector:

- Container is started before tests run
- Database schema is migrated automatically
- Container is cleaned up after tests complete
- Each test suite gets a fresh database state

## Test Structure

### Package Organization

Tests are organized alongside the code they test:

```
internal/
├── handlers/
│   ├── users.go              # Implementation
│   ├── users_test.go         # Unit/integration tests
│   ├── projects.go
│   ├── projects_test.go
│   ├── projects_sharing_test.go
│   ├── embeddings.go
│   └── embeddings_test.go
├── database/
│   └── queries.sql           # SQL queries for sqlc
└── models/
    ├── users.go
    └── projects.go
```

### Test Files

Test files follow Go conventions:

- **Filename**: `*_test.go`
- **Package**: Same as code under test (e.g., `package handlers`)
- **Test functions**: `func TestFunctionName(t *testing.T)`

### Test Fixtures

Test data is stored in `testdata/`:

```
testdata/
├── postgres/
│   ├── enable-vector.sql     # Database initialization
│   └── users.yml
├── valid_embeddings.json
├── valid_user.json
├── valid_api_standard_openai_v1.json
├── valid_llm_service_openai-large-full.json
└── invalid_embeddings.json
```

## Writing Tests

### Basic Test Structure

```go
package handlers

import (
	"context"
	"testing"

	"github.com/mpilhlt/embapi/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestCreateUser(t *testing.T) {
	// Setup: Initialize database pool and test data
	ctx := context.Background()
	pool := setupTestDatabase(t)
	defer pool.Close()
	
	// Execute: Call function under test
	result, err := CreateUser(ctx, pool, userData)
	
	// Assert: Verify results
	assert.NoError(t, err)
	assert.Equal(t, "testuser", result.UserHandle)
}
```

### Integration Test Example

```go
func TestProjectSharingWorkflow(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()
	
	// Create test users
	alice := createTestUser(t, pool, "alice")
	bob := createTestUser(t, pool, "bob")
	
	// Create project as Alice
	project := createTestProject(t, pool, alice, "test-project")
	
	// Share project with Bob
	err := shareProject(t, pool, alice, project.ID, bob.Handle, "reader")
	assert.NoError(t, err)
	
	// Verify Bob can access project
	projects := getAccessibleProjects(t, pool, bob)
	assert.Contains(t, projects, project)
}
```

### Testing with Testcontainers

```go
func setupTestDatabase(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()
	
	// Create PostgreSQL container with pgvector
	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:0.7.4-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
		WaitStrategy: wait.ForLog("database system is ready to accept connections"),
	}
	
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	
	// Get connection details
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	
	// Connect and migrate
	connString := fmt.Sprintf("postgres://postgres:password@%s:%s/testdb", host, port.Port())
	pool := connectAndMigrate(t, connString)
	
	return pool
}
```

### Validation Testing

Test dimension and metadata validation:

```go
func TestEmbeddingDimensionValidation(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()
	
	// Create LLM service with 1536 dimensions
	llmService := createTestLLMService(t, pool, "openai", 1536)
	
	// Try to insert embedding with wrong dimensions
	embedding := models.Embedding{
		TextID:    "doc1",
		Vector:    make([]float32, 768), // Wrong size!
		VectorDim: 768,
	}
	
	err := insertEmbedding(t, pool, embedding)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dimension mismatch")
}

func TestMetadataSchemaValidation(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()
	
	// Create project with metadata schema
	schema := `{"type":"object","properties":{"author":{"type":"string"}},"required":["author"]}`
	project := createProjectWithSchema(t, pool, "alice", "test", schema)
	
	// Valid metadata should succeed
	validMeta := `{"author":"John Doe"}`
	err := insertEmbeddingWithMetadata(t, pool, project.ID, validMeta)
	assert.NoError(t, err)
	
	// Invalid metadata should fail
	invalidMeta := `{"year":2024}` // Missing required 'author'
	err = insertEmbeddingWithMetadata(t, pool, project.ID, invalidMeta)
	assert.Error(t, err)
}
```

### Table-Driven Tests

For testing multiple scenarios:

```go
func TestSimilaritySearch(t *testing.T) {
	tests := []struct {
		name      string
		threshold float32
		limit     int
		expected  int
	}{
		{"high threshold", 0.9, 10, 2},
		{"medium threshold", 0.7, 10, 5},
		{"low threshold", 0.5, 10, 8},
		{"with limit", 0.5, 3, 3},
	}
	
	pool := setupTestDatabase(t)
	defer pool.Close()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searchSimilar(t, pool, "doc1", tt.threshold, tt.limit)
			assert.Len(t, results, tt.expected)
		})
	}
}
```

### Cleanup Testing

Verify database cleanup:

```go
func TestDatabaseCleanup(t *testing.T) {
	pool := setupTestDatabase(t)
	defer pool.Close()
	
	// Create test data
	user := createTestUser(t, pool, "alice")
	project := createTestProject(t, pool, user, "test")
	createTestEmbeddings(t, pool, project, 10)
	
	// Delete user (should cascade)
	err := deleteUser(t, pool, user.Handle)
	assert.NoError(t, err)
	
	// Verify all related data is deleted
	assertTableEmpty(t, pool, "users")
	assertTableEmpty(t, pool, "projects")
	assertTableEmpty(t, pool, "embeddings")
}
```

## Test Helpers

Create helper functions to reduce boilerplate:

```go
// setupTestDatabase initializes a test database with migrations
func setupTestDatabase(t *testing.T) *pgxpool.Pool {
	// Implementation...
}

// createTestUser creates a user for testing
func createTestUser(t *testing.T, pool *pgxpool.Pool, handle string) *models.User {
	// Implementation...
}

// createTestProject creates a project for testing
func createTestProject(t *testing.T, pool *pgxpool.Pool, owner *models.User, handle string) *models.Project {
	// Implementation...
}

// assertTableEmpty verifies a table has no rows
func assertTableEmpty(t *testing.T, pool *pgxpool.Pool, tableName string) {
	var count int
	err := pool.QueryRow(context.Background(), 
		fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "table %s should be empty", tableName)
}
```

## CI/CD Integration

### GitHub Actions

Example GitHub Actions workflow:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.txt ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        files: ./coverage.txt
```

### Local CI Testing

Test as CI would:

```bash
# Clean test with race detection
go clean -testcache
go test -v -race ./...

# Test with coverage
go test -coverprofile=coverage.out ./...

# Check for test caching issues
go test -count=1 ./...
```

## Best Practices

### 1. **Use testcontainers for Real Databases**
- Don't mock the database layer
- Test against actual PostgreSQL with pgvector
- Catch SQL-specific issues

### 2. **Isolate Tests**
- Each test should be independent
- Use transactions or cleanup between tests
- Don't rely on test execution order

### 3. **Test Validation Logic**
- Test dimension validation
- Test metadata schema validation
- Test authorization checks

### 4. **Test Error Conditions**
- Invalid input
- Missing resources (404)
- Unauthorized access (403)
- Constraint violations

### 5. **Keep Tests Fast**
- Use parallel tests where possible: `t.Parallel()`
- Reuse test database containers when appropriate
- Avoid unnecessary sleeps

### 6. **Use Descriptive Names**
```go
// Good
func TestProjectSharingWithReaderRole(t *testing.T)

// Less clear
func TestSharing(t *testing.T)
```

### 7. **Assert Meaningfully**
```go
// Good - specific assertion
assert.Equal(t, "alice", project.Owner)

// Less helpful
assert.True(t, project.Owner == "alice")
```

## Debugging Tests

### Verbose Output

```bash
# See all test output
go test -v ./internal/handlers

# See SQL queries (if logging enabled)
SERVICE_DEBUG=true go test -v ./...
```

### Run Single Test

```bash
# Focus on one failing test
go test -v ./internal/handlers -run TestCreateProject
```

### Keep Test Database Running

For manual inspection, prevent container cleanup:

```go
func TestWithDebugContainer(t *testing.T) {
	container := setupContainer(t)
	// Comment out: defer container.Terminate(ctx)
	
	host, port := getContainerDetails(container)
	t.Logf("Database running at %s:%s", host, port)
	
	// Run test...
	
	// Container stays running for manual inspection
	time.Sleep(time.Hour)
}
```

Then connect with psql:

```bash
psql -h localhost -p <port> -U postgres -d testdb
```

## Common Issues

### Container Startup Failures

```bash
# Check Docker/Podman is running
systemctl --user status podman.socket

# Check for port conflicts
netstat -tulpn | grep 5432

# Clean up containers
podman rm -f $(podman ps -aq)
```

### Test Timeouts

Increase timeout for slow containers:

```go
WaitStrategy: wait.ForLog("ready").WithStartupTimeout(2 * time.Minute)
```

### Permission Errors

Ensure test user has proper permissions:

```sql
GRANT ALL PRIVILEGES ON DATABASE testdb TO testuser;
GRANT ALL ON SCHEMA public TO testuser;
```

## Further Reading

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Testcontainers for Go](https://golang.testcontainers.org/)
- [Testify Assertions](https://github.com/stretchr/testify)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
