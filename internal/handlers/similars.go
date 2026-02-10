package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mpilhlt/embapi/internal/database"
	"github.com/mpilhlt/embapi/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// Define handler functions for each route
func getSimilarFunc(ctx context.Context, input *models.GetSimilarRequest) (*models.SimilarResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	// Check if only one of input.MetadataField and input.MetadataValue are given
	if input.MetadataPath != "" && input.MetadataValue == "" {
		return nil, huma.Error400BadRequest("metadata_path is set but metadata_value is not")
	}
	if input.MetadataPath == "" && input.MetadataValue != "" {
		return nil, huma.Error400BadRequest("metadata_value is set but metadata_path is not")
	}

	// Check if user exists
	_, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		return nil, err
	}

	// Check if project exists
	_, err = getProjectFunc(ctx, &models.GetProjectRequest{UserHandle: input.UserHandle, ProjectHandle: input.ProjectHandle})
	if err != nil {
		return nil, err
	}

	// Check if text exists
	_, err = getDocEmbeddingsFunc(ctx, &models.GetDocEmbeddingsRequest{UserHandle: input.UserHandle, ProjectHandle: input.ProjectHandle, TextID: input.TextID})
	// fmt.Printf("getting doc embeddings for %s\n", input.TextID)
	if err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("database connection error: %v", err))
	}

	// Run the query, either with or without metadata filter
	queries := database.New(pool)
	var sim []database.GetSimilarsByIDRow

	if input.MetadataPath == "" {
		params := database.GetSimilarsByIDParams{
			TextID:        pgtype.Text{String: url.QueryEscape(input.TextID), Valid: true},
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			Column4:       input.Threshold,
			Limit:         min(int32(input.Limit), int32(input.Count)),
			Offset:        int32(input.Offset),
		}
		fmt.Printf("getting similar items for %v\n", params)
		sim, err = queries.GetSimilarsByID(ctx, params)
	} else {
		params := database.GetSimilarsByIDWithFilterParams{
			TextID:        pgtype.Text{String: url.QueryEscape(input.TextID), Valid: true},
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			Column4:       input.Threshold,
			Column5:       input.MetadataPath,
			Column6:       input.MetadataValue,
			Limit:         min(int32(input.Limit), int32(input.Count)),
			Offset:        int32(input.Offset),
		}
		fmt.Printf("getting similar items for %v\n", params)
		var simWithFilter []database.GetSimilarsByIDWithFilterRow
		simWithFilter, err = queries.GetSimilarsByIDWithFilter(ctx, params)
		// Convert to common row type
		for _, r := range simWithFilter {
			sim = append(sim, database.GetSimilarsByIDRow(r))
		}
	}
	fmt.Printf("got this response from the database: %v\n", sim)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound("no similar items found")
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get similar items. %v", err))
	}
	if len(sim) == 0 {
		return nil, huma.Error404NotFound("no similar items found")
	}

	// Build response
	results := []models.SimilarResultItem{}
	for _, r := range sim {
		results = append(results, models.SimilarResultItem{
			ID:         r.TextID.String,
			Similarity: r.Similarity,
		})
	}
	response := &models.SimilarResponse{}
	response.Body.UserHandle = input.UserHandle
	response.Body.ProjectHandle = input.ProjectHandle
	response.Body.Results = results
	return response, nil
}

func postSimilarFunc(ctx context.Context, input *models.PostSimilarRequest) (*models.SimilarResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	// Check if only one of input.MetadataPath and input.MetadataValue are given
	if input.MetadataPath != "" && input.MetadataValue == "" {
		return nil, huma.Error400BadRequest("metadata_path is set but metadata_value is not")
	}
	if input.MetadataPath == "" && input.MetadataValue != "" {
		return nil, huma.Error400BadRequest("metadata_value is set but metadata_path is not")
	}

	// Check if user exists
	_, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("database connection error: %v", err))
	}

	queries := database.New(pool)

	// Check if project exists and get the LLM service instance information
	project, err := queries.RetrieveProject(ctx, database.RetrieveProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s's project %s not found", input.UserHandle, input.ProjectHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get project. %v", err))
	}

	// Get the LLM service instance to validate dimensions
	if !project.InstanceID.Valid {
		return nil, huma.Error400BadRequest("project does not have an associated LLM service instance")
	}

	instance, err := queries.RetrieveInstanceByID(ctx, project.InstanceID.Int32)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve LLM service instance. %v", err))
	}

	// Validate that the vector dimensions match the LLM service instance dimensions
	if !instance.Dimensions.Valid {
		return nil, huma.Error500InternalServerError("LLM service instance does not have dimensions configured")
	}
	if len(input.Body.Vector) != int(instance.Dimensions.Int32) {
		return nil, huma.Error400BadRequest(fmt.Sprintf("vector dimension mismatch: expected %d dimensions, got %d", instance.Dimensions.Int32, len(input.Body.Vector)))
	}

	// Convert the vector to pgvector HalfVector format (half-precision float16)
	// The input []float32 is converted to half-precision during serialization
	vector := pgvector.NewHalfVector(input.Body.Vector)

	// Run the query, either with or without metadata filter
	var sim []database.GetSimilarsByVectorWithProjectRow

	if input.MetadataPath == "" {
		params := database.GetSimilarsByVectorWithProjectParams{
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			Column3:       vector,
			Column4:       input.Threshold,
			Limit:         min(int32(input.Limit), int32(input.Count)),
			Offset:        int32(input.Offset),
		}
		sim, err = queries.GetSimilarsByVectorWithProject(ctx, params)
	} else {
		params := database.GetSimilarsByVectorWithProjectAndFilterParams{
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			Column3:       vector,
			Column4:       input.Threshold,
			Column5:       input.MetadataPath,
			Column6:       input.MetadataValue,
			Limit:         min(int32(input.Limit), int32(input.Count)),
			Offset:        int32(input.Offset),
		}
		var simWithFilter []database.GetSimilarsByVectorWithProjectAndFilterRow
		simWithFilter, err = queries.GetSimilarsByVectorWithProjectAndFilter(ctx, params)
		// Convert to common row type
		for _, r := range simWithFilter {
			sim = append(sim, database.GetSimilarsByVectorWithProjectRow(r))
		}
	}
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound("no similar items found")
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get similar items. %v", err))
	}
	if len(sim) == 0 {
		return nil, huma.Error404NotFound("no similar items found")
	}

	// Build response
	results := []models.SimilarResultItem{}
	for _, r := range sim {
		results = append(results, models.SimilarResultItem{
			ID:         r.TextID.String,
			Similarity: r.Similarity,
		})
	}
	response := &models.SimilarResponse{}
	response.Body.UserHandle = input.UserHandle
	response.Body.ProjectHandle = input.ProjectHandle
	response.Body.Results = results
	return response, nil
}

// RegisterSimilarRoutes registers the routes for the Similar service
func RegisterSimilarRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	getSimilarOp := huma.Operation{
		OperationID: "getSimilar",
		Method:      http.MethodGet,
		Path:        "/v1/similars/{user_handle}/{project_handle}/{text_id}",
		Summary:     "Retrieve similar items for a particular document",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"similars"},
	}
	postSimilarOp := huma.Operation{
		OperationID: "postSimilar",
		Method:      http.MethodPost,
		Path:        "/v1/similars/{user_handle}/{project_handle}",
		Summary:     "Retrieve similar items for a query document",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"similars"},
	}

	huma.Register(api, getSimilarOp, addPoolToContext(pool, getSimilarFunc))
	huma.Register(api, postSimilarOp, addPoolToContext(pool, postSimilarFunc))
	return nil
}
