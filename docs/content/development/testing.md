### Testing with Testcontainers

```go
func setupTestDatabase(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()
	
	// Create PostgreSQL container with pgvector
	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:0.7.4-pg16@sha256:0f7adb53ad971b3fb78e71f2b5daaa88a94c8b0731b114e4673cbf281da0c691",
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
