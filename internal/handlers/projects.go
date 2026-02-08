package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mpilhlt/dhamps-vdb/internal/auth"
	"github.com/mpilhlt/dhamps-vdb/internal/database"
	"github.com/mpilhlt/dhamps-vdb/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// maxSharedUsersPerQuery is the maximum number of shared users to retrieve in a single query
	// This prevents memory issues when a project is shared with many users
	maxSharedUsersPerQuery = 1000
	
	// maxAdminQueryLimit is the maximum limit for admin operations that need to scan all records
	// Used for operations like sanity checks that validate all data in the database
	maxAdminQueryLimit = 999999
)

// Create a new project
func putProjectFunc(ctx context.Context, input *models.PutProjectRequest) (*models.UploadProjectResponse, error) {
	if input.ProjectHandle != input.Body.ProjectHandle {
		return nil, huma.Error400BadRequest(fmt.Sprintf("project handle in URL (%s) does not match project handle in body (%s)", input.ProjectHandle, input.Body.ProjectHandle))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("database connection error: %v", err)
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}
	queries := database.New(pool)

	// 1. Validation

	// - check if user exists
	if _, err := queries.RetrieveUser(ctx, input.UserHandle); err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}
	// - check if instance exists (if provided)
	instanceID := pgtype.Int4{Valid: false}
	if input.Body.InstanceHandle != "" {
		instance, err := queries.RetrieveInstance(ctx, database.RetrieveInstanceParams{Owner: input.Body.InstanceOwner, InstanceHandle: input.Body.InstanceHandle})
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("LLM Service Instance %s owned by %s not found", input.Body.InstanceHandle, input.Body.InstanceOwner))
		}
		instanceID = pgtype.Int4{Int32: int32(instance.InstanceID), Valid: true}
	}

	// - validate shared users exist and roles are valid
	for _, sharedUser := range input.Body.SharedWith {
		// Check that we're not sharing with the owner
		if sharedUser.UserHandle == input.UserHandle {
			return nil, huma.Error400BadRequest(fmt.Sprintf("cannot share project with owner %s", input.UserHandle))
		}
		// Check if role is valid
		if sharedUser.Role != "editor" && sharedUser.Role != "reader" {
			return nil, huma.Error400BadRequest(fmt.Sprintf("invalid role %s for user %s. Role must be either \"editor\" or \"reader\"", sharedUser.Role, sharedUser.UserHandle))
		}
		// Check if target user exists
		_, err := queries.RetrieveUser(ctx, sharedUser.UserHandle)
		if err != nil {
			if err.Error() == "no rows in result set" {
				return nil, huma.Error400BadRequest(fmt.Sprintf("shared user %s does not exist", sharedUser.UserHandle))
			}
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to validate user %s: %v", sharedUser.UserHandle, err))
		}
	}

	// release queries so that they can be used in the transaction below (to link project to users)
	queries = nil

	// 2. Upload project

	var projectID int32
	var projectHandle string

	// - build query parameters (project)
	project := database.UpsertProjectParams{
		ProjectHandle:  input.ProjectHandle,
		Owner:          input.UserHandle,
		Description:    pgtype.Text{String: input.Body.Description, Valid: true},
		MetadataScheme: pgtype.Text{String: input.Body.MetadataScheme, Valid: input.Body.MetadataScheme != ""},
		PublicRead:     pgtype.Bool{Bool: input.Body.PublicRead, Valid: true},
		InstanceID:     instanceID,
	}
	// - execute all database operations within a transaction
	err = database.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		queries := database.New(tx)

		// 1. Upload project
		p, err := queries.UpsertProject(ctx, project)
		if err != nil {
			return fmt.Errorf("unable to upload project. %v", err)
		}
		projectID = p.ProjectID
		projectHandle = p.ProjectHandle

		// 2. Link project and owner
		params := database.LinkProjectToUserParams{ProjectID: projectID, UserHandle: input.UserHandle, Role: "owner"}
		_, err = queries.LinkProjectToUser(ctx, params)
		if err != nil {
			return fmt.Errorf("unable to link project to owner %s. %v", input.UserHandle, err)
		}

		// 3. Link project and other shared users (if any)
		for _, sharedUser := range input.Body.SharedWith {
			params := database.LinkProjectToUserParams{
				ProjectID:  projectID,
				UserHandle: sharedUser.UserHandle,
				Role:       sharedUser.Role,
			}
			_, err := queries.LinkProjectToUser(ctx, params)
			if err != nil {
				return fmt.Errorf("unable to link project to shared user %s. %v", sharedUser.UserHandle, err)
			}
		}

		return nil
	}) // end transaction
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	// 3. Build the response

	response := &models.UploadProjectResponse{}
	response.Body.Owner = input.UserHandle
	response.Body.ProjectHandle = projectHandle
	response.Body.ProjectID = int(projectID)
	response.Body.PublicRead = input.Body.PublicRead
	response.Body.Role = "owner" // the user creating/updating the project is always the owner

	return response, nil
}

// Create a project (without a handle being present in the URL)
func postProjectFunc(ctx context.Context, input *models.PostProjectRequest) (*models.UploadProjectResponse, error) {
	return putProjectFunc(ctx, &models.PutProjectRequest{UserHandle: input.UserHandle, ProjectHandle: input.Body.ProjectHandle, Body: input.Body})
}

// Get all projects for a specific user
func getProjectsFunc(ctx context.Context, input *models.GetProjectsRequest) (*models.GetProjectsResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("database connection error: %v", err)
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}
	queries := database.New(pool)

	// - check if user exists
	if _, err := queries.RetrieveUser(ctx, input.UserHandle); err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}

	// Get the list of projects
	projectHandles, err := queries.GetAccessibleProjectsByUser(ctx, database.GetAccessibleProjectsByUserParams{Owner: input.UserHandle, Limit: int32(input.Limit), Offset: int32(input.Offset)})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("no projects found for user %s", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get projects for user %s. %v", input.UserHandle, err))
	}

	projects := []models.ProjectBrief{}

	/* Get the details for each project (for now, we only give the brief output...)
	 */

	/* Build response array with brief output */
	for _, p := range projectHandles {
		projects = append(projects, models.ProjectBrief{
			Owner:         p.Owner,
			ProjectHandle: p.ProjectHandle,
			ProjectID:     int(p.ProjectID),
			PublicRead:    p.PublicRead.Bool,
			Role:          p.Role.(string),
		})
	}
	// Build the response
	response := &models.GetProjectsResponse{}
	response.Body.Projects = projects

	return response, nil
}

// Retrieve a specific project
func getProjectFunc(ctx context.Context, input *models.GetProjectRequest) (*models.GetProjectResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("database connection error: %v", err)
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}
	queries := database.New(pool)

	// - check if user exists
	if _, err := queries.RetrieveUser(ctx, input.UserHandle); err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}

	// get handle of requesting user from context (set by auth middleware)
	requestingUser := ctx.Value(auth.AuthUserKey)
	if requestingUser == nil {
		return nil, huma.Error500InternalServerError("unable to get requesting user from context")
	}

	var p database.Project
	var role pgtype.Text

	// Admin users can access any project without being in users_projects
	if requestingUser.(string) == "admin" {
		// Use the basic RetrieveProject query for admin users
		p, err = queries.RetrieveProject(ctx, database.RetrieveProjectParams{
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
		})
		if err != nil {
			if err.Error() == "no rows in result set" {
				return nil, huma.Error404NotFound(fmt.Sprintf("user %s's project %s not found", input.UserHandle, input.ProjectHandle))
			}
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get project %s for user %s. %v", input.ProjectHandle, input.UserHandle, err))
		}
		// Admin users have admin role
		role = pgtype.Text{String: "admin"}
	} else {
		// For non-admin users, use RetrieveProjectForUser which checks access permissions
		params := database.RetrieveProjectForUserParams{
			Owner:         input.UserHandle,
			ProjectHandle: input.ProjectHandle,
			UserHandle:    requestingUser.(string),
		}
		projectRow, err := queries.RetrieveProjectForUser(ctx, params)
		if err != nil {
			if err.Error() == "no rows in result set" {
				return nil, huma.Error404NotFound(fmt.Sprintf("user %s's project %s not found", input.UserHandle, input.ProjectHandle))
			}
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get project %s for user %s. %v", input.ProjectHandle, input.UserHandle, err))
		}
		// Convert RetrieveProjectForUserRow to Project
		p = database.Project{
			ProjectID:      projectRow.ProjectID,
			ProjectHandle:  projectRow.ProjectHandle,
			Owner:          projectRow.Owner,
			Description:    projectRow.Description,
			MetadataScheme: projectRow.MetadataScheme,
			PublicRead:     projectRow.PublicRead,
			InstanceID:     projectRow.InstanceID,
		}
		role = projectRow.Role
	}

	// Get the authorized reader accounts for the project (if requested by project owner)
	sharedUsers := []models.SharedUser{}
	if requestingUser.(string) == input.UserHandle {
		// If the project is publicly readable, show "*" in shared_with
		if p.PublicRead.Valid && p.PublicRead.Bool {
			sharedUsers = append(sharedUsers, models.SharedUser{UserHandle: "*", Role: "reader"})
		}
		// Iterate all shared users
		userRows, err := queries.GetUsersByProject(ctx, database.GetUsersByProjectParams{Owner: input.UserHandle, ProjectHandle: input.ProjectHandle, Limit: maxSharedUsersPerQuery, Offset: 0})
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get authorized reader accounts for %s's project %s. %v", input.UserHandle, input.ProjectHandle, err))
		}
		for _, row := range userRows {
			sharedUsers = append(sharedUsers, models.SharedUser{UserHandle: row.UserHandle, Role: row.Role})
		}
	} else {
		// If the requesting user is not the project owner, do not return the list of shared users (privacy reasons)
		sharedUsers = nil
	}

	// Get the LLM Service Instance for the project (1:1 relationship)
	instance := models.InstanceBrief{}
	llmRow, err := queries.RetrieveInstanceByID(ctx, p.InstanceID.Int32)
	if err != nil {
		if err.Error() != "no rows in result set" {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get LLM Service Instance for %s's project %s: %v", input.UserHandle, input.ProjectHandle, err))
		}
		// Project has no LLM service instance assigned yet - just don't populate response's instance field
	} else {
		// Get user's access role for the instance (if any) - to include in the response
		var accessRole string
		if llmRow.Owner == requestingUser.(string) {
			accessRole = "owner"
		} else {
			sharedUsers, err := queries.GetSharedUsersForInstance(ctx, database.GetSharedUsersForInstanceParams{Owner: llmRow.Owner, InstanceHandle: llmRow.InstanceHandle, Limit: maxSharedUsersPerQuery, Offset: 0})
			if err != nil {
				return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get shared users for LLM Service Instance %s owned by %s. %v", llmRow.InstanceHandle, llmRow.Owner, err))
			}
			for _, su := range sharedUsers {
				if su.UserHandle == requestingUser.(string) {
					accessRole = su.Role
					break
				}
			}
		}
		instance = models.InstanceBrief{
			Owner:          llmRow.Owner,
			InstanceID:     int(llmRow.InstanceID),
			InstanceHandle: llmRow.InstanceHandle,
			AccessRole:     accessRole,
		}
	}

	// Get the (number of) embeddings for the project
	count, err := queries.CountEmbeddingsByProject(ctx, database.CountEmbeddingsByProjectParams{Owner: input.UserHandle, ProjectHandle: input.ProjectHandle})
	if err != nil {
		if err.Error() == "no rows in result set" {
			count = 0
		} else {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get number of embeddings for %s's project %s. %v", input.UserHandle, input.ProjectHandle, err))
		}
	}

	// Build the response
	response := &models.GetProjectResponse{}
	response.Body = models.ProjectFull{
		ProjectID:          int(p.ProjectID),
		ProjectHandle:      p.ProjectHandle,
		Owner:              p.Owner,
		Description:        p.Description.String,
		MetadataScheme:     p.MetadataScheme.String,
		SharedWith:         sharedUsers,
		Instance:           instance,
		Role:               role.String,
		NumberOfEmbeddings: int(count),
	}

	return response, nil
}

func deleteProjectFunc(ctx context.Context, input *models.DeleteProjectRequest) (*models.DeleteProjectResponse, error) {
	// Check if user exists
	if _, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle}); err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Check if project exists
	if _, err = getProjectFunc(ctx, &models.GetProjectRequest{UserHandle: input.UserHandle, ProjectHandle: input.ProjectHandle}); err != nil {
		return nil, err
	}

	// Build the query parameters
	params := database.DeleteProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
	}

	// Execute delete operation within a transaction
	err = database.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		queries := database.New(tx)
		err := queries.DeleteProject(ctx, params)
		if err != nil {
			return fmt.Errorf("unable to delete project %s for user %s. %v", input.ProjectHandle, input.UserHandle, err)
		}
		return nil
	})

	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	// Build the response
	response := &models.DeleteProjectResponse{}

	return response, nil
}

// Share a project with another user
func shareProjectFunc(ctx context.Context, input *models.ShareProjectRequest) (*models.ShareProjectResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("database connection error: %v", err)
	}
	queries := database.New(pool)

	// Check if ShareWithUser is identical to owner (no need to share if sharing with self)
	if input.Body.ShareWithHandle == input.UserHandle {
		return nil, huma.Error400BadRequest("cannot share project with owner")
	}
	// Check if role is valid
	if input.Body.Role != "editor" && input.Body.Role != "reader" {
		return nil, huma.Error400BadRequest(fmt.Sprintf("invalid role %s. Role must be either \"editor\" or \"reader\"", input.Body.Role))
	}
	// Check if project exists
	project, err := queries.RetrieveProject(ctx, database.RetrieveProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("project %s/%s not found", input.UserHandle, input.ProjectHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve project %s/%s: %v", input.UserHandle, input.ProjectHandle, err))
	}
	// Check if project belongs to current user (only owner can share)
	if project.Owner != ctx.Value(auth.AuthUserKey).(string) {
		return nil, huma.Error401Unauthorized(fmt.Sprintf("not authorized to share project %s/%s", input.UserHandle, input.ProjectHandle))
	}
	// Check if target user exists
	_, err = getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.Body.ShareWithHandle})
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("target user %s does not exist: %v", input.Body.ShareWithHandle, err))
	}

	// Share the project
	_, err = queries.LinkProjectToUser(ctx, database.LinkProjectToUserParams{
		UserHandle: input.Body.ShareWithHandle,
		ProjectID:  project.ProjectID,
		Role:       input.Body.Role,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to share project: %v", err))
	}

	// Build response - simple acknowledgment with the shared user info
	response := &models.ShareProjectResponse{}
	response.Body.Owner = input.UserHandle
	response.Body.ProjectHandle = input.ProjectHandle
	response.Body.ProjectID = int(project.ProjectID)
	response.Body.SharedWithHandle = input.Body.ShareWithHandle
	response.Body.SharedWithRole = input.Body.Role

	return response, nil
}

// Unshare a project from a user
func unshareProjectFunc(ctx context.Context, input *models.UnshareProjectRequest) (*models.UnshareProjectResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("database connection error: %v", err)
	}
	queries := database.New(pool)

	// Check if project exists and belongs to owner
	project, err := queries.RetrieveProject(ctx, database.RetrieveProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("project %s/%s not found", input.UserHandle, input.ProjectHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve project: %v", err))
	}

	// Check if target user exists and is currently shared
	sharedUsers, err := queries.GetUsersByProject(ctx, database.GetUsersByProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
		Limit:         maxSharedUsersPerQuery,
		Offset:        0,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("project %s/%s is not shared with user %s", input.UserHandle, input.ProjectHandle, input.UnshareWithHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve shared users for project: %v", err))
	}
	for _, su := range sharedUsers {
		if su.UserHandle == input.UnshareWithHandle {
			// Unshare the project
			err = queries.UnlinkProjectFromUser(ctx, database.UnlinkProjectFromUserParams{
				UserHandle: input.UnshareWithHandle,
				ProjectID:  project.ProjectID,
			})
			if err != nil {
				return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to unshare project %s/%s from user %s: %v", input.UserHandle, input.ProjectHandle, input.UnshareWithHandle, err))
			}
			// Build response
			response := &models.UnshareProjectResponse{}
			return response, nil
		}
	}
	// If we get here, the target user exists but is not currently shared
	return nil, huma.Error404NotFound(fmt.Sprintf("project %s/%s is not shared with user %s", input.UserHandle, input.ProjectHandle, input.UnshareWithHandle))
}

// Get all users a project is shared with
func getProjectSharedUsersFunc(ctx context.Context, input *models.GetProjectSharedUsersRequest) (*models.GetProjectSharedUsersResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("database connection error: %v", err)
	}
	queries := database.New(pool)

	// Get the requesting user from context (set by auth middleware)
	requestingUser := ctx.Value(auth.AuthUserKey)
	if requestingUser == nil {
		return nil, huma.Error500InternalServerError("unable to get requesting user from context")
	}

	// Only the project owner can see the list of shared users
	if requestingUser.(string) != input.UserHandle {
		return nil, huma.Error403Forbidden("only the project owner can view shared users list")
	}

	// Get shared users
	sharedUsers, err := queries.GetUsersByProject(ctx, database.GetUsersByProjectParams{
		Owner:         input.UserHandle,
		ProjectHandle: input.ProjectHandle,
		Limit:         maxSharedUsersPerQuery,
		Offset:        0,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			// Return empty list instead of error
			response := &models.GetProjectSharedUsersResponse{}
			response.Body.SharedWith = []models.SharedUser{}
			response.Body.Owner = input.UserHandle
			response.Body.ProjectHandle = input.ProjectHandle
			return response, nil
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve shared users: %v", err))
	}

	// Build response
	users := []models.SharedUser{}
	for _, su := range sharedUsers {
		// Skip the owner - only include shared users
		if su.UserHandle == input.UserHandle {
			continue
		}
		users = append(users, models.SharedUser{
			UserHandle: su.UserHandle,
			Role:       su.Role,
		})
	}

	response := &models.GetProjectSharedUsersResponse{}
	response.Body.Owner = input.UserHandle
	response.Body.ProjectHandle = input.ProjectHandle
	response.Body.SharedWith = users

	return response, nil
}

// RegisterProjectRoutes registers all the project routes with the API
func RegisterProjectsRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	putProjectOp := huma.Operation{
		OperationID:   "putProject",
		Method:        http.MethodPut,
		Path:          "/v1/projects/{user_handle}/{project_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create or update a project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"admin", "projects"},
	}
	postProjectOp := huma.Operation{
		OperationID:   "postProject",
		Method:        http.MethodPost,
		Path:          "/v1/projects/{user_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create or update a project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"admin", "projects"},
	}
	getProjectsOp := huma.Operation{
		OperationID: "getProjects",
		Method:      http.MethodGet,
		Path:        "/v1/projects/{user_handle}",
		Summary:     "Get all projects for a specific user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"admin", "projects"},
	}
	getProjectOp := huma.Operation{
		OperationID: "getProject",
		Method:      http.MethodGet,
		Path:        "/v1/projects/{user_handle}/{project_handle}",
		Summary:     "Get a specific project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"admin", "projects"},
	}
	deleteProjectOp := huma.Operation{
		OperationID:   "deleteProject",
		Method:        http.MethodDelete,
		Path:          "/v1/projects/{user_handle}/{project_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete a specific project",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"admin", "projects"},
	}
	shareProjectOp := huma.Operation{
		OperationID:   "shareProject",
		Method:        http.MethodPost,
		Path:          "/v1/projects/{user_handle}/{project_handle}/share",
		DefaultStatus: http.StatusCreated,
		Summary:       "Share a project with another user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"projects"},
	}
	unshareProjectOp := huma.Operation{
		OperationID:   "unshareProject",
		Method:        http.MethodDelete,
		Path:          "/v1/projects/{user_handle}/{project_handle}/share/{unshare_with_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Unshare a project from a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"projects"},
	}
	getProjectSharedUsersOp := huma.Operation{
		OperationID: "getProjectSharedUsers",
		Method:      http.MethodGet,
		Path:        "/v1/projects/{user_handle}/{project_handle}/shared-with",
		Summary:     "Get all users a project is shared with",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"projects"},
	}

	huma.Register(api, putProjectOp, addPoolToContext(pool, putProjectFunc))
	huma.Register(api, postProjectOp, addPoolToContext(pool, postProjectFunc))
	huma.Register(api, getProjectsOp, addPoolToContext(pool, getProjectsFunc))
	huma.Register(api, getProjectOp, addPoolToContext(pool, getProjectFunc))
	huma.Register(api, deleteProjectOp, addPoolToContext(pool, deleteProjectFunc))
	huma.Register(api, shareProjectOp, addPoolToContext(pool, shareProjectFunc))
	huma.Register(api, unshareProjectOp, addPoolToContext(pool, unshareProjectFunc))
	huma.Register(api, getProjectSharedUsersOp, addPoolToContext(pool, getProjectSharedUsersFunc))
	return nil
}
