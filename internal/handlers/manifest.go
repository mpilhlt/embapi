package handlers

import (
	"context"
	"net/http"

	"github.com/mpilhlt/embapi/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// getManifestFunc returns the service manifest
func getManifestFunc(ctx context.Context, input *models.GetManifestRequest) (*models.GetManifestResponse, error) {
	// Get options from context
	options, err := GetOptions(ctx)
	if err != nil {
		return nil, err
	}

	// Use configured values or fall back to defaults
	apiName := models.APIName
	if options.APIName != "" {
		apiName = options.APIName
	}

	apiDescription := models.APIDescription
	if options.APIDescription != "" {
		apiDescription = options.APIDescription
	}

	apiDocURL := "https://mpilhlt.github.io/embapi/"
	if options.APIDocURL != "" {
		apiDocURL = options.APIDocURL
	}

	// Build the manifest
	// Note: The endpoint list is manually maintained to provide comprehensive API documentation.
	// While it could be generated dynamically from registered routes, the manual approach ensures
	// accurate descriptions and proper grouping, and allows filtering of internal/test endpoints.
	manifest := models.ServiceManifest{
		Name:           apiName,
		Versions:       []string{"v1"},
		Description:    apiDescription,
		Documentation:  apiDocURL,
		ServiceVersion: models.APIVersion,
		Authentication: map[string]interface{}{
			"adminAuth": map[string]interface{}{
				"type":   "http",
				"scheme": "bearer",
				"description": "Admin API key for administrative operations",
			},
			"ownerAuth": map[string]interface{}{
				"type":   "http",
				"scheme": "bearer",
				"description": "Owner API key for resource management",
			},
			"editorAuth": map[string]interface{}{
				"type":   "http",
				"scheme": "bearer",
				"description": "Editor API key for editing shared resources",
			},
			"readerAuth": map[string]interface{}{
				"type":   "http",
				"scheme": "bearer",
				"description": "Reader API key for read-only access to shared resources",
			},
		},
		Endpoints: []models.EndpointInfo{
			// Users
			{
				Path:        "/v1/users",
				Methods:     []string{"GET", "POST"},
				Description: "Manage users",
				Tags:        []string{"users"},
			},
			{
				Path:        "/v1/users/{user_handle}",
				Methods:     []string{"GET", "PUT", "DELETE"},
				Description: "Manage a specific user",
				Tags:        []string{"users"},
			},
			// Projects
			{
				Path:        "/v1/projects/{user_handle}",
				Methods:     []string{"GET", "POST"},
				Description: "List or create projects for a user",
				Tags:        []string{"projects"},
			},
			{
				Path:        "/v1/projects/{user_handle}/{project_handle}",
				Methods:     []string{"GET", "PUT", "DELETE"},
				Description: "Manage a specific project",
				Tags:        []string{"projects"},
			},
			{
				Path:        "/v1/projects/{user_handle}/{project_handle}/share",
				Methods:     []string{"GET", "POST"},
				Description: "Manage project sharing",
				Tags:        []string{"projects", "sharing"},
			},
			{
				Path:        "/v1/projects/{user_handle}/{project_handle}/share/{unshare_with_handle}",
				Methods:     []string{"DELETE"},
				Description: "Remove sharing access to a project",
				Tags:        []string{"projects", "sharing"},
			},
			{
				Path:        "/v1/projects/{user_handle}/{project_handle}/shared-with",
				Methods:     []string{"GET"},
				Description: "List users with whom a project is shared",
				Tags:        []string{"projects", "sharing"},
			},
			{
				Path:        "/v1/projects/{user_handle}/{project_handle}/transfer-ownership",
				Methods:     []string{"POST"},
				Description: "Transfer ownership of a project to another user",
				Tags:        []string{"projects"},
			},
			// Embeddings
			{
				Path:        "/v1/embeddings/{user_handle}/{project_handle}",
				Methods:     []string{"GET", "POST", "DELETE"},
				Description: "Manage embeddings in a project",
				Tags:        []string{"embeddings"},
			},
			{
				Path:        "/v1/embeddings/{user_handle}/{project_handle}/{text_id}",
				Methods:     []string{"GET", "PUT", "DELETE"},
				Description: "Manage a specific embedding by text ID",
				Tags:        []string{"embeddings"},
			},
			// Similarity Search
			{
				Path:        "/v1/similars/{user_handle}/{project_handle}/{text_id}",
				Methods:     []string{"GET"},
				Description: "Find similar documents by text ID",
				Tags:        []string{"similarity"},
			},
			{
				Path:        "/v1/similars/{user_handle}/{project_handle}",
				Methods:     []string{"POST"},
				Description: "Find similar documents by posting a vector",
				Tags:        []string{"similarity"},
			},
			// LLM Service Definitions
			{
				Path:        "/v1/llm-definitions/{user_handle}",
				Methods:     []string{"GET", "POST"},
				Description: "List or create LLM service definitions",
				Tags:        []string{"llm-services"},
			},
			{
				Path:        "/v1/llm-definitions/{user_handle}/{definition_handle}",
				Methods:     []string{"GET", "PUT", "DELETE"},
				Description: "Manage a specific LLM service definition",
				Tags:        []string{"llm-services"},
			},
			{
				Path:        "/v1/llm-definitions/{user_handle}/{definition_handle}/share",
				Methods:     []string{"POST"},
				Description: "Share an LLM service definition with other users",
				Tags:        []string{"llm-services", "sharing"},
			},
			{
				Path:        "/v1/llm-definitions/{user_handle}/{definition_handle}/share/{unshare_with_handle}",
				Methods:     []string{"DELETE"},
				Description: "Remove sharing access to an LLM service definition",
				Tags:        []string{"llm-services", "sharing"},
			},
			{
				Path:        "/v1/llm-definitions/{user_handle}/{definition_handle}/shared-with",
				Methods:     []string{"GET"},
				Description: "List users with whom an LLM service definition is shared",
				Tags:        []string{"llm-services", "sharing"},
			},
			// LLM Service Instances
			{
				Path:        "/v1/llm-instances/{user_handle}",
				Methods:     []string{"GET", "POST"},
				Description: "List or create LLM service instances",
				Tags:        []string{"llm-services"},
			},
			{
				Path:        "/v1/llm-instances/{user_handle}/from-definition",
				Methods:     []string{"POST"},
				Description: "Create an LLM service instance from a definition",
				Tags:        []string{"llm-services"},
			},
			{
				Path:        "/v1/llm-instances/{user_handle}/{instance_handle}",
				Methods:     []string{"GET", "PUT", "DELETE"},
				Description: "Manage a specific LLM service instance",
				Tags:        []string{"llm-services"},
			},
			{
				Path:        "/v1/llm-instances/{user_handle}/{instance_handle}/share",
				Methods:     []string{"POST"},
				Description: "Share an LLM service instance with other users",
				Tags:        []string{"llm-services", "sharing"},
			},
			{
				Path:        "/v1/llm-instances/{user_handle}/{instance_handle}/share/{unshare_with_handle}",
				Methods:     []string{"DELETE"},
				Description: "Remove sharing access to an LLM service instance",
				Tags:        []string{"llm-services", "sharing"},
			},
			{
				Path:        "/v1/llm-instances/{user_handle}/{instance_handle}/shared-with",
				Methods:     []string{"GET"},
				Description: "List users with whom an LLM service instance is shared",
				Tags:        []string{"llm-services", "sharing"},
			},
			// API Standards
			{
				Path:        "/v1/api-standards",
				Methods:     []string{"GET", "POST"},
				Description: "List or create API standards",
				Tags:        []string{"api-standards"},
			},
			{
				Path:        "/v1/api-standards/{api_standard_handle}",
				Methods:     []string{"GET", "PUT", "DELETE"},
				Description: "Manage a specific API standard",
				Tags:        []string{"api-standards"},
			},
			// Admin
			{
				Path:        "/v1/admin/footgun",
				Methods:     []string{"GET"},
				Description: "Reset the database (admin only, for testing)",
				Tags:        []string{"admin"},
			},
			{
				Path:        "/v1/admin/sanity-check",
				Methods:     []string{"GET"},
				Description: "Perform a sanity check on the database",
				Tags:        []string{"admin"},
			},
		},
	}

	// Build the response
	response := &models.GetManifestResponse{}
	response.Body = manifest
	return response, nil
}

// RegisterManifestRoutes registers the manifest routes with the API
func RegisterManifestRoutes(pool *pgxpool.Pool, api huma.API, options *models.Options) error {
	// Define huma.Operations for the manifest endpoint
	// This endpoint is public and doesn't require authentication
	// We register at both / (root) and /v1 (versioned root) for discoverability
	getManifestRootOp := huma.Operation{
		OperationID: "getManifestRoot",
		Method:      http.MethodGet,
		Path:        "/{$}",
		Summary:     "Get service manifest describing the API",
		Description: "Returns a service manifest with metadata about the API and a list of available endpoints",
		Security:    []map[string][]string{},
		Tags:        []string{"public", "manifest"},
	}

	getManifestV1Op := huma.Operation{
		OperationID: "getManifestV1",
		Method:      http.MethodGet,
		Path:        "/v1",
		Summary:     "Get service manifest for API version 1",
		Description: "Returns a service manifest with metadata about API version 1 and a list of available endpoints",
		Security:    []map[string][]string{},
		Tags:        []string{"public", "manifest"},
	}

	// Create a handler wrapper that includes both pool and options in context
	handler := func(ctx context.Context, input *models.GetManifestRequest) (*models.GetManifestResponse, error) {
		return getManifestFunc(ctx, input)
	}

	// Register the routes with both pool and options in context
	huma.Register(api, getManifestRootOp, addOptionsToContext(options, addPoolToContext(pool, handler)))
	huma.Register(api, getManifestV1Op, addOptionsToContext(options, addPoolToContext(pool, handler)))
	return nil
}
