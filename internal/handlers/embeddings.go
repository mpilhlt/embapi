package handlers

import (
	"context"
	"encoding/json"
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

// Get user and project
func getUserProj(ctx context.Context, user, project string) (string, string, int32, error) {
	// Check if user and project exist
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: user})
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == fmt.Sprintf("user %s not found", user) {
			return "", "", 0, huma.Error404NotFound(fmt.Sprintf("user %s not found", user))
		}
		return "", "", 0, huma.Error500InternalServerError(fmt.Sprintf("unable to get user %s. %v", user, err))
	}
	if u.Body.UserHandle != user {
		return "", "", 0, huma.Error404NotFound(fmt.Sprintf("user %s not found", user))
	}
	p, err := getProjectFunc(ctx, &models.GetProjectRequest{UserHandle: user, ProjectHandle: project})
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == fmt.Sprintf("user %s's project %s not found", user, project) {
			return "", "", 0, huma.Error404NotFound(fmt.Sprintf("%s's project %s not found", user, project))
		}
		return "", "", 0, huma.Error500InternalServerError(fmt.Sprintf("unable to get %s's project %s. %v", user, project, err))
	}
	if p.Body.ProjectHandle != project {
		return "", "", 0, huma.Error404NotFound(fmt.Sprintf("%s's project %s not found", user, project))
	}
	return u.Body.UserHandle, p.Body.ProjectHandle, int32(p.Body.ProjectID), nil
}

// Create a new embeddings
func postProjEmbeddingsFunc(ctx context.Context, input *models.PostProjEmbeddingsRequest) (*models.UploadProjEmbeddingsResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("Could not acces database connection pool: %v", err))
	}
	queries := database.New(pool)

	// Check if user exists
	_, err = queries.RetrieveUser(ctx, input.UserHandle)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("User %s does not exist: %v", input.UserHandle, err))
	}

	// Retrieve project details
	project, err := queries.RetrieveProject(ctx, database.RetrieveProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
	})
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("Project %s/%s not found: %v", input.UserHandle, input.ProjectHandle, err))
	}

	fmt.Printf("Found project %s ...", project.ProjectHandle)

	// Even though each of the submitted embeddings specifies an instance handle,
	// we may postulate that only one instance is involved: the one that is
	// connected to the project (every project has exactly one instance) passed
	// in the request's URL path. Thus, we verify that it exists only once.
	// Rather than checking for each embeddings' instance whether it exists and
	// the current user can read it, we just validate that the instance handle
	// specified in the embeddings record matches the one connected to the project.
	instance, err := queries.RetrieveInstanceByProjectID(ctx, int32(project.ProjectID))
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("Cannot access LLM Service Instance specified in the project %s/%s: %v", input.UserHandle, input.ProjectHandle, err))
	}

	// For each embedding, validate input, build query parameters and run the query
	ids := []string{}
	for _, embedding := range input.Body.Embeddings {

		// Validate if instance specified in the embedding matches the one connected to the project
		if embedding.InstanceHandle != instance.InstanceHandle {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Instance handle '%s' for embedding with text_id '%s' does not match the instance handle '%s' connected to project '%s/%s'", embedding.InstanceHandle, embedding.TextID, instance.InstanceHandle, input.UserHandle, input.ProjectHandle))
		}

		// Validate embedding dimensions
		if !instance.Dimensions.Valid {
			return nil, huma.Error500InternalServerError("LLM service instance does not have dimensions configured")
		}
		if err := ValidateEmbeddingDimensions(embedding, instance.Dimensions.Int32); err != nil {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Dimension validation failed for input %s: %v", embedding.TextID, err))
		}

		// Check if embedding already exists to determine if this is an update
		existingEmbedding, err := queries.RetrieveEmbeddings(ctx, database.RetrieveEmbeddingsParams{
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			TextID:        pgtype.Text{String: embedding.TextID, Valid: true},
		})
		isUpdate := err == nil
		var existingMetadata json.RawMessage
		// If it already exists, integrate the update with existing data before schema validation.
		if isUpdate {
			// If the update does not include text, keep the existing text
			if embedding.Text == "" {
				embedding.Text = existingEmbedding.Text.String
			}
			existingMetadata = existingEmbedding.Metadata
			// If the update has metadata, integrate it with the existing metadata
			// (new keys are added, existing keys are updated, keys with null value are deleted)
			if len(embedding.Metadata) != 0 {
				mergedMetadata, err := mergeMetadata(existingEmbedding.Metadata, embedding.Metadata)
				if err != nil {
					return nil, huma.Error400BadRequest(fmt.Sprintf("Invalid metadata for text_id '%s': %v", embedding.TextID, err))
				}
				embedding.Metadata = mergedMetadata
			}
		}

		// Validate metadata against schema if provided
		if !project.MetadataScheme.Valid || project.MetadataScheme.String != "" {
			if err := ValidateMetadataAgainstSchema(embedding.Metadata, project.MetadataScheme.String, isUpdate, existingMetadata); err != nil {
				return nil, huma.Error400BadRequest(fmt.Sprintf("metadata validation failed for text_id '%s': %v", embedding.TextID, err))
			}
		}

		// Build query parameters (embeddings)
		params := database.UpsertEmbeddingsParams{
			TextID:     pgtype.Text{String: embedding.TextID, Valid: true},
			Owner:      input.UserHandle,
			ProjectID:  project.ProjectID,
			InstanceID: instance.InstanceID,
			Text:       pgtype.Text{String: embedding.Text, Valid: true},
			Vector:     pgvector.NewHalfVector(embedding.Vector),
			VectorDim:  embedding.VectorDim,
			Metadata:   embedding.Metadata,
		}
		// Run the queries (upload embeddings)
		result, err := queries.UpsertEmbeddings(ctx, params)
		if err != nil {
			fmt.Printf("Error: %v\n(Params were: %v)\n", err, params)
			return nil, huma.Error500InternalServerError(fmt.Sprintf("Unable to upload embeddings. %v", err))
		}

		ids = append(ids, result.TextID.String)
	}

	// Build response
	response := &models.UploadProjEmbeddingsResponse{}
	response.Body.IDs = ids
	return response, nil
}

func mergeMetadata(existing, new json.RawMessage) (json.RawMessage, error) {
	var existingMap map[string]interface{}
	if len(existing) != 0 {
		if err := json.Unmarshal(existing, &existingMap); err != nil {
			return nil, fmt.Errorf("unable to unmarshal existing metadata: %v", err)
		}
	} else {
		existingMap = make(map[string]interface{})
	}

	var newMap map[string]interface{}
	if err := json.Unmarshal(new, &newMap); err != nil {
		return nil, fmt.Errorf("unable to unmarshal new metadata: %v", err)
	}

	for k, v := range newMap {
		if v == nil {
			delete(existingMap, k)
		} else {
			existingMap[k] = v
		}
	}

	merged, err := json.Marshal(existingMap)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal merged metadata: %v", err)
	}
	return merged, nil
}

func getProjEmbeddingsFunc(ctx context.Context, input *models.GetProjEmbeddingsRequest) (*models.GetProjEmbeddingsResponse, error) {
	// Check if user exists
	if _, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle}); err != nil {
		return nil, err
	}

	// Check if project exists
	if _, err := getProjectFunc(ctx, &models.GetProjectRequest{UserHandle: input.UserHandle, ProjectHandle: input.ProjectHandle}); err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Build query parameters (embeddings)
	params := database.GetEmbeddingsByProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
		Limit:         int32(input.Limit),
		Offset:        int32(input.Offset),
	}

	// Run the query
	queries := database.New(pool)
	embeddingss, err := queries.GetEmbeddingsByProject(ctx, params)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("no embeddings found for user %s, project %s.", input.UserHandle, input.ProjectHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get embeddings for user %s, project %s. %v", input.UserHandle, input.ProjectHandle, err))
	}
	if len(embeddingss) == 0 {
		return nil, huma.Error404NotFound(fmt.Sprintf("no embeddings found for user %s, project %s.", input.UserHandle, input.ProjectHandle))
	}

	// Build the response
	e := []models.Embeddings{}
	for _, emb := range embeddingss {
		embeddings, err := queries.RetrieveEmbeddings(ctx, database.RetrieveEmbeddingsParams{
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			TextID:        pgtype.Text{String: emb.TextID.String, Valid: true},
		})
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get embeddings for user %s, project %s, id %s. %v", input.UserHandle, input.ProjectHandle, emb.TextID.String, err))
		}

		md := map[string]interface{}{}
		err = json.Unmarshal(embeddings.Metadata, &md)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to unmarshal metadata for user %s, project %s, id %s. Metadata: %s. %v", input.UserHandle, input.ProjectHandle, embeddings.TextID.String, string(embeddings.Metadata), err))
		}
		e = append(e, models.Embeddings{
			TextID:         embeddings.TextID.String,
			UserHandle:     embeddings.Owner,
			ProjectHandle:  embeddings.ProjectHandle,
			ProjectID:      int(embeddings.ProjectID),
			InstanceHandle: embeddings.InstanceHandle,
			Vector:         embeddings.Vector.Slice(),
			VectorDim:      embeddings.VectorDim,
			Text:           embeddings.Text.String,
			Metadata:       md,
		})
	}
	response := &models.GetProjEmbeddingsResponse{}
	response.Body.Embeddings = e
	return response, nil
}

func deleteProjEmbeddingsFunc(ctx context.Context, input *models.DeleteProjEmbeddingsRequest) (*models.DeleteProjEmbeddingsResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	// Check if user and project exist
	_, _, _, err := getUserProj(ctx, input.UserHandle, input.ProjectHandle)
	if err != nil {
		if err.Error() == "no rows in result set" || err == huma.Error404NotFound(fmt.Sprintf("user %s's project %s not found", input.UserHandle, input.ProjectHandle)) {
			return nil, huma.Error404NotFound(fmt.Sprintf("project %s of user %s not found", input.ProjectHandle, input.UserHandle))
		}
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Build query parameters (embeddings)
	params := database.DeleteEmbeddingsByProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
	}

	// Run the query
	queries := database.New(pool)
	err = queries.DeleteEmbeddingsByProject(ctx, params)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete embeddings for %s's project %s. %v", input.UserHandle, input.ProjectHandle, err))
	}

	// Build the response
	response := &models.DeleteProjEmbeddingsResponse{}
	return response, nil
}

func getDocEmbeddingsFunc(ctx context.Context, input *models.GetDocEmbeddingsRequest) (*models.GetDocEmbeddingsResponse, error) {
	// Check if user and project exist
	_, _, _, err := getUserProj(ctx, input.UserHandle, input.ProjectHandle)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("project %s of user %s not found", input.ProjectHandle, input.UserHandle))
		}
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	textid := url.QueryEscape(input.TextID)

	// Build query parameters (embeddings)
	params := database.RetrieveEmbeddingsParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
		TextID:        pgtype.Text{String: textid, Valid: true},
	}

	// fmt.Printf("getDocEmbeddings, textid: %v\n", textid)

	// Run the query
	queries := database.New(pool)
	embeddings, err := queries.RetrieveEmbeddings(ctx, params)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("no embeddings found for user %s, project %s, id %s.", input.UserHandle, input.ProjectHandle, textid))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get embeddings for user %s, project %s, id %s. %v", input.UserHandle, input.ProjectHandle, textid, err))
	}
	if embeddings.TextID.String == "" {
		return nil, huma.Error404NotFound(fmt.Sprintf("no embeddings found for user %s, project %s, id %s.", input.UserHandle, input.ProjectHandle, textid))
	}

	// Build the response
	md := map[string]interface{}{}
	err = json.Unmarshal(embeddings.Metadata, &md)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to unmarshal metadata for user %s, project %s, id %s. Metadata: %s. %v", input.UserHandle, input.ProjectHandle, embeddings.TextID.String, string(embeddings.Metadata), err))
	}
	e := models.Embeddings{
		TextID:         embeddings.TextID.String,
		UserHandle:     embeddings.Owner,
		ProjectHandle:  embeddings.ProjectHandle,
		ProjectID:      int(embeddings.ProjectID),
		InstanceHandle: embeddings.InstanceHandle,
		Vector:         embeddings.Vector.Slice(),
		VectorDim:      embeddings.VectorDim,
		Text:           embeddings.Text.String,
		Metadata:       md,
	}
	response := &models.GetDocEmbeddingsResponse{}
	response.Body = e

	return response, nil
}

func deleteDocEmbeddingsFunc(ctx context.Context, input *models.DeleteEmbeddingsByDocIDRequest) (*models.DeleteEmbeddingsByDocIDResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	// Check if user and project exist
	_, _, _, err := getUserProj(ctx, input.UserHandle, input.ProjectHandle)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("project %s of user %s not found", input.ProjectHandle, input.UserHandle))
		}
		return nil, err
	}

	textid := url.QueryEscape(input.TextID)

	// Check if embeddings with ID exist
	textidForChecking := input.TextID // the getDocEmbeddings expects a url-decoded path parameter
	_, err = getDocEmbeddingsFunc(ctx, &models.GetDocEmbeddingsRequest{UserHandle: input.UserHandle, ProjectHandle: input.ProjectHandle, TextID: textidForChecking})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("text id %s in %s's project %s not found", textid, input.UserHandle, input.ProjectHandle))
		}
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Build query parameters for DeleteEmbeddings
	params := database.DeleteEmbeddingsByDocIDParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
		TextID:        pgtype.Text{String: textid, Valid: true},
	}

	// fmt.Printf("deleteDocEmbeddings, textid: %v\n", textid)

	// Run the query
	queries := database.New(pool)
	err = queries.DeleteEmbeddingsByDocID(ctx, params)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete embeddings for text id %s in %s's project %s. %v", textid, input.UserHandle, input.ProjectHandle, err))
	}

	// Build the response
	response := &models.DeleteEmbeddingsByDocIDResponse{}
	return response, nil
}

// RegisterEmbeddingsRoutes registers all the embeddings routes with the API
func RegisterEmbeddingsRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	postProjEmbeddingsOp := huma.Operation{
		OperationID:   "postEmbeddings",
		Method:        http.MethodPost,
		Path:          "/v1/embeddings/{user_handle}/{project_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create embeddings for a project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"editorAuth": []string{"editor"}},
		},
		Tags: []string{"embeddings"},
	}
	getProjEmbeddingsOp := huma.Operation{
		OperationID: "getEmbeddings",
		Method:      http.MethodGet,
		Path:        "/v1/embeddings/{user_handle}/{project_handle}",
		Summary:     "Get all embeddings for a project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"embeddings"},
	}
	deleteProjEmbeddingsOp := huma.Operation{
		OperationID:   "deleteEmbeddings",
		Method:        http.MethodDelete,
		Path:          "/v1/embeddings/{user_handle}/{project_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete all embeddings for a project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"editorAuth": []string{"editor"}},
		},
		Tags: []string{"embeddings"},
	}
	getDocEmbeddingsOp := huma.Operation{
		OperationID: "getDocEmbeddings",
		Method:      http.MethodGet,
		Path:        "/v1/embeddings/{user_handle}/{project_handle}/{text_id}",
		Summary:     "Get embeddings for a specific document",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"embeddings"},
	}
	deleteDocEmbeddingsOp := huma.Operation{
		OperationID:   "deleteDocEmbeddings",
		Method:        http.MethodDelete,
		Path:          "/v1/embeddings/{user_handle}/{project_handle}/{text_id}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete embeddings for a specific document",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"editorAuth": []string{"editor"}},
		},
		Tags: []string{"embeddings"},
	}

	// huma.Register(api, putProjEmbeddingsOp, addPoolToContext(pool, putProjEmbeddingsFunc))
	huma.Register(api, postProjEmbeddingsOp, addPoolToContext(pool, postProjEmbeddingsFunc))
	huma.Register(api, getProjEmbeddingsOp, addPoolToContext(pool, getProjEmbeddingsFunc))
	huma.Register(api, deleteProjEmbeddingsOp, addPoolToContext(pool, deleteProjEmbeddingsFunc))
	huma.Register(api, getDocEmbeddingsOp, addPoolToContext(pool, getDocEmbeddingsFunc))
	huma.Register(api, deleteDocEmbeddingsOp, addPoolToContext(pool, deleteDocEmbeddingsFunc))
	return nil
}
