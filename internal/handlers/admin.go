package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mpilhlt/dhamps-vdb/internal/database"
	"github.com/mpilhlt/dhamps-vdb/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func resetDbFunc(ctx context.Context, input *models.ResetDbRequest) (*models.ResetDbResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		fmt.Printf("    Resetting Database: error getting database connection pool: %v\n", err)
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get database connection pool. %v", err))
	} else if pool == nil {
		fmt.Print("    Resetting Database: database connection pool is nil\n")
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}
	queries := database.New(pool)

	// delete all records
	fmt.Print("    Resetting Database: deleting all records...\n")
	err = queries.DeleteAllRecords(ctx)
	if err != nil {
		fmt.Printf("    Resetting Database: error deleting all records: %v\n", err)
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete all records. %v", err))
	}

	fmt.Print("    Resetting Database: resetting serials...\n")
	err = queries.ResetAllSerials(ctx)
	if err != nil {
		fmt.Printf("    Resetting Database: error resetting serials: %v\n", err)
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to reset serials. %v", err))
	}

	// Build response
	response := &models.ResetDbResponse{}
	return response, nil
}

func sanityCheckFunc(ctx context.Context, input *models.SanityCheckRequest) (*models.SanityCheckResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get database connection pool. %v", err))
	}

	queries := database.New(pool)

	// Get all projects with their metadata schemes
	// Use maxAdminQueryLimit since sanity check needs to validate all projects
	projects, err := queries.GetAllProjects(ctx, database.GetAllProjectsParams{
		Limit:  maxAdminQueryLimit,
		Offset: 0,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get projects. %v", err))
	}

	var issues []string
	var warnings []string

	// Check each project
	for _, p := range projects {
		project, err := queries.RetrieveProject(ctx, database.RetrieveProjectParams(p))
		if err != nil {
			issues = append(issues, fmt.Sprintf("Project %s/%s: unable to retrieve project: %v", p.Owner, p.ProjectHandle, err))
			continue
		}
		projectName := fmt.Sprintf("%s/%s", project.Owner, project.ProjectHandle)

		// Get the LLM service instance for this project (1:1 relationship)
		instance, err := queries.RetrieveInstanceByProject(ctx, database.RetrieveInstanceByProjectParams{
			Owner:         project.Owner,
			ProjectHandle: project.ProjectHandle,
		})
		if err != nil {
			issues = append(issues, fmt.Sprintf("Project %s: unable to get LLM service instance: %v", projectName, err))
			continue
		}

		// Create a map with the single LLM service instance
		llmDimensions := make(map[int32]int32)
		llmDimensions[instance.InstanceID] = instance.Dimensions

		// Get all embeddings for this project
		embeddings, err := queries.GetEmbeddingsByProject(ctx, database.GetEmbeddingsByProjectParams{
			Owner:         project.Owner,
			ProjectHandle: project.ProjectHandle,
			Limit:         9999999,
			Offset:        0,
		})
		if err != nil {
			issues = append(issues, fmt.Sprintf("Project %s: unable to get embeddings: %v", projectName, err))
			continue
		}

		// Check each embedding
		for _, e := range embeddings {
			embedding, err := queries.RetrieveEmbeddingsByID(ctx, e.EmbeddingsID)
			if err != nil {
				issues = append(issues, fmt.Sprintf("Project %s, embedding ID %d: unable to retrieve embedding: %v",
					projectName, e.EmbeddingsID, err))
				continue
			}
			textID := embedding.TextID.String

			// Check dimension consistency
			expectedDim, ok := llmDimensions[embedding.InstanceID]
			if !ok {
				issues = append(issues, fmt.Sprintf("Project %s, text_id '%s': LLM service ID %d not found",
					projectName, textID, embedding.InstanceID))
				continue
			}

			if err := ValidateEmbeddingAgainstLLMDimension(embedding.VectorDim, expectedDim, textID); err != nil {
				issues = append(issues, fmt.Sprintf("Project %s: %v", projectName, err))
			}

			// Check metadata against schema if schema is defined
			if project.MetadataScheme.Valid && project.MetadataScheme.String != "" {
				// For sanity check, we're checking existing data, so isUpdate=true and we have existing metadata
				if err := ValidateEmbeddingMetadataAgainstProjectSchema(embedding.Metadata, project.MetadataScheme.String, textID, true, embedding.Metadata); err != nil {
					issues = append(issues, fmt.Sprintf("Project %s: %v", projectName, err))
				}
			}
		}

		// Warn if project has embeddings but no metadata scheme defined
		if len(embeddings) > 0 && (!project.MetadataScheme.Valid || project.MetadataScheme.String == "") {
			warnings = append(warnings, fmt.Sprintf("Project %s has %d embeddings but no metadata schema defined",
				projectName, len(embeddings)))
		}
	}

	// Build response
	response := &models.SanityCheckResponse{}
	response.Body.TotalProjects = len(projects)
	response.Body.Issues = issues
	response.Body.Warnings = warnings
	response.Body.IssuesCount = len(issues)
	response.Body.WarningsCount = len(warnings)

	if len(issues) > 0 {
		response.Body.Status = "FAILED"
	} else if len(warnings) > 0 {
		response.Body.Status = "WARNING"
	} else {
		response.Body.Status = "PASSED"
	}

	return response, nil
}

// RegisterUsersRoutes registers all the admin routes with the API
func RegisterAdminRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	footgunOp := huma.Operation{
		OperationID:   "footgun",
		Method:        http.MethodGet,
		Path:          "/v1/admin/footgun",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Remove all records from database and reset serials/counters",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		// MaxBodyBytes int64 `yaml:"-"` // Max size of the request body in bytes (-1 for unlimited)
		// BodyReadTimeout time.Duration `yaml:"-" // Time to wait for the request body to be read (-1 for unlimited)
		// Middlewares Middlewares `yaml:"-"` // Middleware to run before the operation, useful for logging, etc.
		Tags: []string{"admin"},
	}
	sanityCheckOp := huma.Operation{
		OperationID: "sanityCheck",
		Method:      http.MethodGet,
		Path:        "/v1/admin/sanity-check",
		Summary:     "Verify all data in database conforms to schemas and dimension requirements",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		Tags: []string{"admin"},
	}

	// Register the routes with middleware
	huma.Register(api, footgunOp, addPoolToContext(pool, resetDbFunc))
	huma.Register(api, sanityCheckOp, addPoolToContext(pool, sanityCheckFunc))
	return nil
}
