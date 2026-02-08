package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mpilhlt/embapi/internal/database"
	"github.com/mpilhlt/embapi/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// putAPIStandardFunc creates or updates an API standard
func putAPIStandardFunc(ctx context.Context, input *models.PutAPIStandardRequest) (*models.UploadAPIStandardResponse, error) {
	if input.APIStandardHandle != input.Body.APIStandardHandle {
		return nil, huma.Error400BadRequest(fmt.Sprintf("API standard handle in URL (%s) does not match handle in body (%v).", input.APIStandardHandle, input.Body.APIStandardHandle))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Build query parameters
	// Check if standard already exists
	queries := database.New(pool)
	a, err := queries.RetrieveAPIStandard(ctx, input.APIStandardHandle)
	if err != nil && err.Error() != "no rows in result set" {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to check if API standard %s already exists. %v", input.APIStandardHandle, err))
	}
	if a.APIStandardHandle == input.APIStandardHandle {
		// Standard exists, just update it
		fmt.Printf("        API Standard %s already existss.\n", input.APIStandardHandle)
	}
	apiParams := database.UpsertAPIStandardParams{
		APIStandardHandle: input.APIStandardHandle,
		Description:       pgtype.Text{String: input.Body.Description, Valid: true},
		KeyMethod:         input.Body.KeyMethod,
		KeyField:          pgtype.Text{String: input.Body.KeyField, Valid: true},
	}

	// Run the query
	api, err := queries.UpsertAPIStandard(ctx, apiParams)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to upload standard. %v", err))
	}

	// Build the response
	response := &models.UploadAPIStandardResponse{}
	response.Body.APIStandardHandle = api
	return response, nil
}

// Create an API standard (without a handle being present in the URL)
func postAPIStandardFunc(ctx context.Context, input *models.PostAPIStandardRequest) (*models.UploadAPIStandardResponse, error) {
	u, err := putAPIStandardFunc(ctx, &models.PutAPIStandardRequest{APIStandardHandle: input.Body.APIStandardHandle, Body: input.Body})
	if err != nil {
		return nil, err
	}
	// Build the response
	response := &models.UploadAPIStandardResponse{}
	response.Body.APIStandardHandle = u.Body.APIStandardHandle
	return response, nil
}

// Get all registered  API standards
func getAPIStandardsFunc(ctx context.Context, input *models.GetAPIStandardsRequest) (*models.GetAPIStandardsResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Run the query
	queries := database.New(pool)
	allAPIStandards, err := queries.GetAPIStandards(ctx, database.GetAPIStandardsParams{Limit: int32(input.Limit), Offset: int32(input.Offset)})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound("no API standards found.")
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get list of API standards. %v", err))
	}
	if len(allAPIStandards) == 0 {
		return nil, huma.Error404NotFound("no API standards found.")
	}

	// Build the response
	standards := []models.APIStandard{}
	for _, api := range allAPIStandards {
		a, err := queries.RetrieveAPIStandard(ctx, api)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get API standard data for standard %s. %v", api, err))
		}
		standard := models.APIStandard{
			APIStandardHandle: a.APIStandardHandle,
			Description:       a.Description.String,
			KeyMethod:         a.KeyMethod,
			KeyField:          a.KeyField.String,
		}
		standards = append(standards, standard)
	}
	response := &models.GetAPIStandardsResponse{}
	response.Body.APIStandards = standards

	return response, nil
}

// Get a specific API standard
func getAPIStandardFunc(ctx context.Context, input *models.GetAPIStandardRequest) (*models.GetAPIStandardResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Run the query
	queries := database.New(pool)
	a, err := queries.RetrieveAPIStandard(ctx, input.APIStandardHandle)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("API standard %s not found", input.APIStandardHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get API standard data for standard %s. %v", input.APIStandardHandle, err))
		// return nil, huma.Error404NotFound(fmt.Sprintf("API standard %s not found. %v", input.APIStandardHandle, err))
	}

	// Build the response
	returnAPIStandard := &models.APIStandard{
		APIStandardHandle: a.APIStandardHandle,
		Description:       a.Description.String,
		KeyMethod:         a.KeyMethod,
		KeyField:          a.KeyField.String,
	}
	response := &models.GetAPIStandardResponse{}
	response.Body = *returnAPIStandard

	return response, nil
}

// Delete a specific API standard
func deleteAPIStandardFunc(ctx context.Context, input *models.DeleteAPIStandardRequest) (*models.DeleteAPIStandardResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Check if API standard exists
	queries := database.New(pool)
	_, err = queries.RetrieveAPIStandard(ctx, input.APIStandardHandle)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("API standard %s not found", input.APIStandardHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to check if API standard %s exists before deleting. %v", input.APIStandardHandle, err))
	}

	// Check if API standard is still in use by any LLM service definitions and prevent deletion if so, or cascade delete, depending on desired behavior and use cases. For now, we just return an error if the standard is in use.
	inUse, err := queries.CheckIfAPIStandardInUse(ctx, input.APIStandardHandle)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to check if API standard %s is in use before deleting. %v", input.APIStandardHandle, err))
	}
	if inUse {
		return nil, huma.Error400BadRequest(fmt.Sprintf("cannot delete API standard %s because it is still in use by one or more LLM service definitions. Please update or delete those definitions first.", input.APIStandardHandle))
	}

	// Run the query
	err = queries.DeleteAPIStandard(ctx, input.APIStandardHandle)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete API standard %s. %v", input.APIStandardHandle, err))
	}

	// Build the response
	response := &models.DeleteAPIStandardResponse{}
	return response, nil
}

// RegisterAPIStandardsRoutes registers all the admin routes with the API
func RegisterAPIStandardsRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	putAPIStandardOp := huma.Operation{
		OperationID:   "putAPIStandard",
		Method:        http.MethodPut,
		Path:          "/v1/api-standards/{api_standard_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create or update an API standard",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		// MaxBodyBytes int64 `yaml:"-"` // Max size of the request body in bytes (-1 for unlimited)
		// BodyReadTimeout time.Duration `yaml:"-" // Time to wait for the request body to be read (-1 for unlimited)
		// Middlewares Middlewares `yaml:"-"` // Middleware to run before the operation, useful for logging, etc.
		Tags: []string{"admin", "api-standards"},
	}
	postAPIStandardOp := huma.Operation{
		OperationID:   "postAPIStandard",
		Method:        http.MethodPost,
		Path:          "/v1/api-standards",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create an API standard",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		Tags: []string{"admin", "api-standards"},
	}
	getAPIStandardsOp := huma.Operation{
		OperationID: "getAPIStandards",
		Method:      http.MethodGet,
		Path:        "/v1/api-standards",
		Summary:     "Get information about all API standards",
		Security:    []map[string][]string{},
		Tags:        []string{"public", "api-standards"},
	}
	getAPIStandardOp := huma.Operation{
		OperationID: "getAPIStandard",
		Method:      http.MethodGet,
		Path:        "/v1/api-standards/{api_standard_handle}",
		Summary:     "Get information about a specific API standard",
		Security:    []map[string][]string{},
		Tags:        []string{"public", "api-standards"},
	}
	deleteAPIStandardOp := huma.Operation{
		OperationID:   "deleteAPIStandard",
		Method:        http.MethodDelete,
		Path:          "/v1/api-standards/{api_standard_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete a specific API standard",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		Tags: []string{"admin", "api-standards"},
	}

	// Register the routes with middleware
	huma.Register(api, putAPIStandardOp, addPoolToContext(pool, putAPIStandardFunc))
	huma.Register(api, postAPIStandardOp, addPoolToContext(pool, postAPIStandardFunc))
	huma.Register(api, getAPIStandardsOp, addPoolToContext(pool, getAPIStandardsFunc))
	huma.Register(api, getAPIStandardOp, addPoolToContext(pool, getAPIStandardFunc))
	huma.Register(api, deleteAPIStandardOp, addPoolToContext(pool, deleteAPIStandardFunc))
	return nil
}
