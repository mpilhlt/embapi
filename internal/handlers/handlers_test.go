package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mpilhlt/embapi/internal/auth"
	"github.com/mpilhlt/embapi/internal/database"
	"github.com/mpilhlt/embapi/internal/handlers"
	"github.com/mpilhlt/embapi/internal/models"

	huma "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TODO: Use values from env or config to override default options.
// TODO: Set up timeouts!

// Each package ("handlers", in this case) can have its own TestMain function.
// This function is executed before any tests in the package are run and can
// be used to set up resources needed by the tests. It can also be used to
// run setup code that should only be run once for the entire package.
// It has a signature of func TestMain(m *testing.M), where m has a single
// method Run() that runs all the tests in the package. It should call os.Exit
// with the result of m.Run() to signal the test runner whether the tests
// passed or failed.

// While there is the humago package to set up a testing API against which we
// could register our routes and run requests, we use an actual API connecting
// to a real database. We still can choose between a live postgres database and
// a testcontainer spun up just for testing.

var (
	options = models.Options{
		Debug:       true,
		AuthVerbose: false, // Set to false for cleaner test output
		Host:        "localhost",
		Port:        8080,
		DBHost:      "localhost",
		DBName:      "testdb",
		DBUser:      "test",
		DBPassword:  "test",
		AdminKey:    "Password123",
	}
	connPool *pgxpool.Pool
	teardown func()
)

// TestMain function initializes the database container.
// Then it runs all the tests. Setup of api, router and server
// is done in the tests themselves to provide better isolation.
func TestMain(m *testing.M) {
	// Set required environment variables for testing
	if os.Getenv("ENCRYPTION_KEY") == "" {
		os.Setenv("ENCRYPTION_KEY", "test-encryption-key-32-bytes-long-1234567890")
	}

	// Get a database connection pool
	var err error
	connPool, err, teardown = getTestDatabase()
	if err != nil {
		fmt.Printf("Unable to get database connection pool: %v", err)
		teardown()
		os.Exit(1)
	}
	if connPool == nil {
		fmt.Print("Database connection pool is nil")
		teardown()
		os.Exit(1)
	}
	defer connPool.Close()
	defer teardown()
	fmt.Print("\n    Database ready\n    Running tests ...\n\n")

	// Run the tests
	code := m.Run() // Execute all the tests

	os.Exit(code)
}

// --- Helper functions and types ---

// GetTestDatabase spins up a new Postgres container and returns
// a connection pool, an error value and a closure.
// Please always make sure to call the closure as it is the teardown
// function terminating the container.
func getTestDatabase() (*pgxpool.Pool, error, func()) {
	ctx := context.Background()

	// 1. Run PostgreSQL container
	pgVectorContainer, err := postgres.Run(ctx,
		// "pgvector/pgvector:pg16",
		"pgvector/pgvector:0.7.4-pg16",
		postgres.WithDatabase(options.DBName),
		postgres.WithUsername(options.DBUser),
		postgres.WithPassword(options.DBPassword),
		postgres.WithInitScripts(filepath.Join("..", "..", "testdata", "postgres", "enable-vector.sql")),
		testcontainers.WithWaitStrategy(
			// First, we wait for the container to log readiness twice.
			// This is because it will restart itself after the first startup.
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(120*time.Second),
			// Then, we wait for docker to actually serve the port on localhost.
			// For non-linux OSes like Mac and Windows, Docker or Rancher Desktop will have to start a separate proxy.
			// Without this, the tests will be flaky on those OSes!
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(120*time.Second),
		),
	)
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
		return nil, err, nil
	}
	connStr, err := pgVectorContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("Error reading connection string: %v\n", err)
		return nil, err, nil
	}

	// 2. Connect to db
	connPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		fmt.Printf("Error creating connection pool: %v\n", err)
		return nil, err, nil
	}
	err = connPool.Ping(context.Background())
	if err != nil {
		fmt.Printf("Error pinging connection pool: %v\n", err)
		return nil, err, nil
	}
	fmt.Printf("Connection pool of database %v/%v established.\n", options.DBHost, options.DBName)

	// 3. Prepare test database
	err = database.VerifySchema(ctx, connStr)
	if err != nil {
		fmt.Printf("Error preparing test database: %v\n", err)
		return nil, err, nil
	}

	return connPool, nil, func() {
		err := pgVectorContainer.Terminate(context.Background())
		if err != nil {
			fmt.Printf("Error terminating container: %v\n", err)
		}
	}
}

// startTestServer sets up server, router and API for testing.
// It returns an error value and a closure function that
// should be called to clean up.
// It is supposed to be called from the various tests.
func startTestServer(t *testing.T, pool *pgxpool.Pool, keyGen handlers.RandomKeyGenerator) (error, func()) {
	/* Use transactions to isolate tests from each other.

	   // Get a database connection
	   conn, err := pool.Acquire(context.Background())
	   if err != nil {
	       t.Fatal(err)
	   }

	   // Start a transaction
	   _, err = conn.Exec(context.Background(), "BEGIN")
	   if err != nil {
	       t.Fatal(err)
	   }
	*/

	// Create a new router & API
	config := huma.DefaultConfig(models.APIName, models.APIVersion)
	config.Components.SecuritySchemes = auth.Config
	router := http.NewServeMux()
	api := humago.New(router, config)
	api.UseMiddleware(auth.EmbAPIKeyAdminAuth(api, &options))
	api.UseMiddleware(auth.EmbAPIKeyOwnerAuth(api, pool, &options))
	api.UseMiddleware(auth.EmbAPIKeyEditorAuth(api, pool, &options))
	api.UseMiddleware(auth.EmbAPIKeyReaderAuth(api, pool, &options))
	api.UseMiddleware(auth.AuthTermination(api, &options))

	err := handlers.AddRoutes(pool, keyGen, api)
	if err != nil {
		fmt.Printf("Unable to add routes to API: %v", err)
		return err, func() {}
	}

	// Add AutoPatch to automatically create PATCH endpoints for resources with GET+PUT
	autopatch.AutoPatch(api)

	fmt.Print("    Router ready\n")

	/* HTTP Server setup (we set up httptest.Server below instead)
	   // Create the HTTP server
	   server := &http.Server{
	     Addr:    fmt.Sprintf("%s:%d", options.Host, options.Port),
	     Handler: router,
	   }
	   // Start the server
	   go func() {
	     if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	       fmt.Printf("Unable to start server: %v", err)
	       server.Close()    // Close the server
	       connPool.Close()  // Close the database connection
	       teardown()
	       os.Exit(1)
	     }
	   }()
	   time.Sleep(1 * time.Second) // Wait for the server to start
	*/

	// For testing, we use a httptest.Server instead of a real server.
	// Running this on our custom port requires setting up a listener...
	// create a listener with the desired port.
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", options.Host, options.Port))
	assert.NoError(t, err)
	if err != nil {
		fmt.Printf("Error setting up listener: %v", err)
		return err, func() {}
	}
	// Create a new server with the router.
	server := httptest.NewUnstartedServer(router)
	// NewUnstartedServer creates a server-cum-listener.
	// Close that listener and replace with the one we created.
	server.Listener.Close()
	server.Listener = l
	// Start the server.
	server.Start()
	fmt.Printf("    Server listening on %s:%d\n", options.Host, options.Port)

	cleanup := func() {
		// Close the server
		server.Close()
		// Wait a moment to ensure the port is fully released
		time.Sleep(100 * time.Millisecond)
		/* Clean up transactions.
		   _, err := conn.Exec(context.Background(), "ROLLBACK")
		   if err != nil {
		       t.Fatal(err)
		   }
		   conn.Release()
		*/
	}

	return nil, cleanup
}

// MockKeyGen is a mock implementation of the RandomKeyGenerator interface.
type MockKeyGen struct{ mock.Mock }

// Implement mock's randomKey method
func (m *MockKeyGen) RandomKey(len int) (string, error) {
	args := m.Called(len)
	return args.String(0), args.Error(1)
}

// createUser creates a user and returns the API key and an error value
// it accepts a JSON string encoding the user object as input
func createUser(t *testing.T, userJSON string) (string, error) {
	// Extract user handle from JSON
	jsonInput := &struct {
		UserHandle string `json:"user_handle"`
		Name       string `json:"name"`
		Email      string `json:"email"`
	}{}
	err := json.Unmarshal([]byte(userJSON), jsonInput)
	if err != nil {
		fmt.Printf("Error unmarshalling user JSON: %v\n", err)
		return "", err
	}
	assert.NoError(t, err)

	fmt.Printf("    Creating user (\"%s\") for testing ...\n", jsonInput.UserHandle)
	requestURL := fmt.Sprintf("http://%s:%d/v1/users/%s", options.Host, options.Port, jsonInput.UserHandle)
	requestBody := bytes.NewReader([]byte(userJSON))
	req, err := http.NewRequest(http.MethodPut, requestURL, requestBody)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+options.AdminKey)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// get API key for user from response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Check if response was successful
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error creating user: status code %d, body: %s\n", resp.StatusCode, string(body))
		return "", fmt.Errorf("status code %d: %s", resp.StatusCode, string(body))
	}

	// fmt.Printf("Response body: %v\n", string(body))
	userInfo := models.UserResponse{}
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		fmt.Printf("Error unmarshalling user info: %v\nbody: %v\n", err, string(body))
		return "", err
	}
	assert.NoError(t, err)
	fmt.Printf("        Successfully created user (handle: \"%s\", EmbAPIKey: \"%s\").\n", userInfo.UserHandle, userInfo.EmbAPIKey)
	// fmt.Printf("        User info: %v\n", userInfo)
	return userInfo.EmbAPIKey, nil
}

// createProject creates a project and returns the project ID and an error value
// it accepts a JSON string encoding the project object as input
func createProject(t *testing.T, projectJSON string, user string, EmbAPIKey string) (int, error) {
	fmt.Print("    Creating project ")
	jsonInput := &struct {
		Handle         string `json:"project_handle" doc:"Handle of created or updated project"`
		InstanceOwner  string `json:"instance_owner,omitempty" doc:"User handle of the owner of the LLM Service Instance used in the project."`
		InstanceHandle string `json:"instance_handle,omitempty" doc:"Handle of the LLM Service Instance used in the project"`
		Description    string `json:"description" doc:"Description of the project"`
		IsPublic       bool   `json:"is_public,omitempty" default:"false" doc:"Whether the project should be public or not"`
	}{}

	err := json.Unmarshal([]byte(projectJSON), jsonInput)
	if err != nil {
		fmt.Printf("Error unmarshalling project JSON: %v\n", err)
	}
	assert.NoError(t, err)
	fmt.Printf("(\"%s/%s\") for testing ...\n", user, jsonInput.Handle)

	requestURL := fmt.Sprintf("http://%s:%d/v1/projects/%s/%s", options.Host, options.Port, user, jsonInput.Handle)
	requestBody := bytes.NewReader([]byte(projectJSON))
	req, err := http.NewRequest(http.MethodPut, requestURL, requestBody)
	req.Header.Set("Authorization", "Bearer "+EmbAPIKey)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// get project ID from response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	projectInfo := &struct {
		Owner         string `json:"owner" doc:"User handle of the project owner"`
		ProjectHandle string `json:"project_handle" doc:"Handle of created or updated project"`
		ProjectID     int    `json:"project_id" doc:"Unique project identifier"`
		PublicRead    bool   `json:"public_read" doc:"Whether the project is public or not"`
		Role          string `json:"role,omitempty" doc:"Role of the requesting user in the project (can be owner or some other role)"`
	}{}
	err = json.Unmarshal(body, &projectInfo)
	if err != nil {
		fmt.Printf("Error unmarshalling project info: %v\nbody: %v", err, string(body))
	}
	assert.NoError(t, err)

	fmt.Printf("        Successfully created project (handle \"%s/%s\", id \"%d\").\n", user, projectInfo.ProjectHandle, projectInfo.ProjectID)
	return projectInfo.ProjectID, nil
}

// createAPIStandard creates an API standard definition for testing and returns the handle and an error value
// it accepts a JSON string encoding the API standard object as input
func createAPIStandard(t *testing.T, apiStandardJSON string, EmbAPIKey string) (string, error) {
	fmt.Print("    Creating API standard ")
	jsonInput := &struct {
		APIStandardHandle string `json:"api_standard_handle" doc:"Handle of created or updated api standard"`
		Description       string `json:"description" doc:"Description of the api standard"`
		KeyMethod         string `json:"key_method" doc:"Method used to authenticate the API standard"`
		KeyField          string `json:"key_field" doc:"Field in the request used to authenticate the API standard"`
	}{}
	err := json.Unmarshal([]byte(apiStandardJSON), jsonInput)
	if err != nil {
		fmt.Printf("\nError unmarshalling api standard JSON: %v\n", err)
	}
	assert.NoError(t, err)
	fmt.Printf("(\"%s\") for testing ...\n", jsonInput.APIStandardHandle)

	requestURL := fmt.Sprintf("http://%s:%d/v1/api-standards/%s", options.Host, options.Port, jsonInput.APIStandardHandle)
	requestBody := bytes.NewReader([]byte(apiStandardJSON))
	req, err := http.NewRequest(http.MethodPut, requestURL, requestBody)
	req.Header.Set("Authorization", "Bearer "+EmbAPIKey)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// get API handle from response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	APIStandardInfo := &struct {
		APIStandardHandle string `json:"api_standard_handle" doc:"Handle of created or updated api standard"`
	}{}
	err = json.Unmarshal(body, &APIStandardInfo)
	if err != nil {
		fmt.Printf("Error unmarshalling api standard info: %v\nbody: %v", err, string(body))
	}
	assert.NoError(t, err)

	fmt.Printf("        Successfully created API Standard (handle \"%s\").\n", APIStandardInfo.APIStandardHandle)
	return APIStandardInfo.APIStandardHandle, nil
}

// createInstance creates an LLM service instance for testing and returns the handle and an error value
// it accepts a JSON string encoding the LLM service instance object as input
// NOTE: This function is kept for backward compatibility with existing tests
// It creates an LLM Service Instance (not a Definition)
func createInstance(t *testing.T, instanceJSON string, user string, EmbAPIKey string) (string, error) {
	fmt.Print("    Creating LLM service instance ")

	// Parse JSON to extract handle - support both old and new field names
	jsonInput := &struct {
		InstanceHandle string `json:"instance_handle" doc:"Old field name for backward compatibility"`
		Owner          string `json:"owner" doc:"User handle of the service owner"`
		Description    string `json:"description" doc:"Description of the LLM service"`
	}{}
	err := json.Unmarshal([]byte(instanceJSON), jsonInput)
	if err != nil {
		fmt.Printf("Error unmarshalling LLM service JSON: %v\n", err)
	}
	assert.NoError(t, err)
	handle := jsonInput.InstanceHandle
	fmt.Printf("(\"%s/%s\") for testing ...\n", user, handle)

	requestURL := fmt.Sprintf("http://%s:%d/v1/llm-instances/%s/%s", options.Host, options.Port, user, handle)
	requestBody := bytes.NewReader([]byte(instanceJSON))
	req, err := http.NewRequest(http.MethodPut, requestURL, requestBody)
	req.Header.Set("Authorization", "Bearer "+EmbAPIKey)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// get LLM service instance handle from response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	// Check if response was successful
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error creating LLM service instance: status code %d, body: %s\n", resp.StatusCode, string(body))
		return "", fmt.Errorf("status code %d: %s", resp.StatusCode, string(body))
	}

	InstanceInfo := &struct {
		Owner          string `json:"owner" doc:"User handle of the LLM Service Instance owner"`
		InstanceHandle string `json:"instance_handle" doc:"Handle of created or updated LLM Service Instance"`
		InstanceID     int    `json:"instance_id" doc:"System identifier of created or updated LLM Service Instance"`
	}{}
	err = json.Unmarshal(body, &InstanceInfo)
	if err != nil {
		fmt.Printf("Error unmarshalling LLM Service Instance info: %v\nbody: %v", err, string(body))
	}
	assert.NoError(t, err)

	fmt.Printf("        Successfully created LLM Service Instance (handle \"%s\", id %d).\n", InstanceInfo.InstanceHandle, InstanceInfo.InstanceID)
	return InstanceInfo.InstanceHandle, nil
}

// createDefinition creates an LLM service definition for testing and returns the handle and an error value
// it accepts a JSON string encoding the LLM service definition object as input
func createDefinition(t *testing.T, DefinitionJSON string, user string, EmbAPIKey string) (int32, error) {
	fmt.Print("    Creating LLM service definition ")
	jsonInput := &struct {
		Owner            string `json:"owner" doc:"User handle of the service definition owner"`
		DefinitionHandle string `json:"definition_handle" doc:"Handle of created or updated LLM service definition"`
		Description      string `json:"description,omitempty" doc:"Description of the LLM service definition"`
		Endpoint         string `json:"endpoint,omitempty" doc:"Endpoint of the LLM service definition"`
		APIStandard      string `json:"api_standard,omitempty" doc:"API standard followed by the LLM service definition"`
		Model            string `json:"model,omitempty" doc:"Model name used in the LLM service definition"`
		Dimensions       int    `json:"dimensions,omitempty" doc:"Dimensions of the embeddings used in the LLM service definition"`
		ContextLimit     int    `json:"context_limit,omitempty" doc:"Context limit of the LLM service definition"`
		IsPublic         bool   `json:"is_public" doc:"Whether the LLM service definition is public or not"`
	}{}
	err := json.Unmarshal([]byte(DefinitionJSON), jsonInput)
	if err != nil {
		fmt.Printf("Error unmarshalling LLM service definition JSON: %v\nJSON: %s", err, string(DefinitionJSON))
	}
	assert.NoError(t, err)
	fmt.Printf("(%s/%s) for testing ...\n", user, jsonInput.DefinitionHandle)

	requestURL := fmt.Sprintf("http://%s:%d/v1/llm-definitions/%s/%s", options.Host, options.Port, user, jsonInput.DefinitionHandle)
	requestBody := bytes.NewReader([]byte(DefinitionJSON))
	req, err := http.NewRequest(http.MethodPut, requestURL, requestBody)
	req.Header.Set("Authorization", "Bearer "+EmbAPIKey)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// get LLM service definition handle from response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	DefinitionInfo := &struct {
		Schema           string `json:"$schema" doc:"JSON schema URL for the response body"`
		DefinitionID     int    `json:"definition_id" doc:"System identifier of created or updated LLM Service Definition"`
		DefinitionHandle string `json:"definition_handle" doc:"Handle of created or updated LLM Service Definition"`
		Owner            string `json:"owner" doc:"User handle of the LLM Service Definition owner"`
		IsPublic         bool   `json:"is_public" doc:"Whether the LLM Service Definition is public or not"`
	}{}
	err = json.Unmarshal(body, &DefinitionInfo)
	if err != nil {
		fmt.Printf("Error unmarshalling LLM service definition info: %v\nbody: %v", err, string(body))
	}
	assert.NoError(t, err)

	fmt.Printf("        Successfully created LLM Service Definition (%s/%s, id %d).\n", DefinitionInfo.Owner, DefinitionInfo.DefinitionHandle, DefinitionInfo.DefinitionID)
	return int32(DefinitionInfo.DefinitionID), nil
}

// createEmbeddings creates some embeddings entries for testing and returns an error value
// it accepts a JSON string encording the embeddings db entries
func createEmbeddings(t *testing.T, embeddings []byte, user string, Instance string, EmbAPIKey string) error {
	fmt.Print("    Creating Embeddings for testing ...\n")

	// Upload embeddings for similars testing
	requestURL := fmt.Sprintf("http://%s:%d/v1/embeddings/%s/%s", options.Host, options.Port, user, Instance)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(embeddings))
	if err != nil {
		fmt.Printf("Error creating request for uploading embeddings: %v\n", err)
	}
	req.Header.Set("Authorization", "Bearer "+EmbAPIKey)
	req.Header.Set("Content-Type", "application/json")
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request to upload embeddings: %v\n", err)
	}
	defer resp.Body.Close()
	assert.NoError(t, err)

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Expected status code %d, got %d. Response: %s\n", http.StatusCreated, resp.StatusCode, string(respBody))
	}
	assert.NoError(t, err)
	fmt.Print("        Successfully created embeddings.\n")
	return nil
}

// isJSON checks if a string is a valid JSON.
func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}
