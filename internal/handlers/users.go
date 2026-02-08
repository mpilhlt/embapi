package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/mpilhlt/dhamps-vdb/internal/database"
	"github.com/mpilhlt/dhamps-vdb/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// putUserFunc creates or updates a user
func putUserFunc(ctx context.Context, input *models.PutUserRequest) (*models.UploadUserResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	if input.UserHandle != input.Body.UserHandle {
		return nil, huma.Error400BadRequest(fmt.Sprintf("user handle in URL (%s) does not match user handle in body (%v).", input.UserHandle, input.Body.UserHandle))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Get the API key generator from the context
	keyGen, err := GetKeyGen(ctx)
	if err != nil {
		return nil, err
	}

	// Build query parameters (user - eventually with new API key)
	// Check if user already exists
	queries := database.New(pool)
	u, err := queries.RetrieveUser(ctx, input.UserHandle)
	if err != nil && err.Error() != "no rows in result set" {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to check if user %s already exists. %v", input.UserHandle, err))
	}

	// Create API key if user does not exist
	// storeKey := make([]byte, 64)
	var storeKey string
	VDBKey := ""
	if u.UserHandle == input.UserHandle {
		// User exists, so don't create API key
		storeKey = u.VDBKey
		fmt.Printf("        User %s already exists, stored key hash is %s.\n", input.UserHandle, storeKey)
		// fmt.Printf("        User %s already exists: %v.\n", input.UserHandle, u)
		// fmt.Printf("        User %s. Stored key hash: '%s'.\n", input.UserHandle, u.VDBKey)
		VDBKey = "not changed"
	} else {
		// User does not exist, so create a new API key
		VDBKey, err = keyGen.RandomKey(32)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to create API key for user %s. %v", input.UserHandle, err))
		}
		hash := sha256.Sum256([]byte(VDBKey))
		storeKey = hex.EncodeToString(hash[:])
		// fmt.Printf("        Created user %s: API key %s (store hash: %s)\n", input.UserHandle, APIKey, storeKey)
	}
	user := database.UpsertUserParams{
		UserHandle: input.UserHandle,
		Name:       pgtype.Text{String: input.Body.Name, Valid: true},
		Email:      input.Body.Email,
		VDBKey:     storeKey,
	}

	// Run the query
	s, err := queries.UpsertUser(ctx, user)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to upload user. %v", err))
	}
	u, err = queries.RetrieveUser(ctx, s)
	if err != nil && err.Error() != "no rows in result set" {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to verify that user %s exists now. %v", s, err))
	}

	// Build the response
	response := &models.UploadUserResponse{}
	response.Body.UserHandle = u.UserHandle
	// Return the actual API key only if it was just created
	// When updating an existing user, don't include the VDB key in the response
	if VDBKey != "not changed" {
		response.Body.VDBKey = VDBKey
	} else {
		response.Body.VDBKey = "not changed"
	}

	return response, nil
}

// Create a user (without a handle being present in the URL)
func postUserFunc(ctx context.Context, input *models.PostUserRequest) (*models.UploadUserResponse, error) {
	u, err := putUserFunc(ctx, &models.PutUserRequest{UserHandle: input.Body.UserHandle, Body: input.Body})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Get all users (handles only)
func getUsersFunc(ctx context.Context, input *models.GetUsersRequest) (*models.GetUsersResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Run the query
	queries := database.New(pool)
	allUsers, err := queries.GetAllUsers(ctx, database.GetAllUsersParams{Limit: int32(input.Limit), Offset: int32(input.Offset)})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound("no users found.")
		} else {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get list of users. %v", err))
		}
	}
	if len(allUsers) == 0 {
		return nil, huma.Error404NotFound("no users found.")
	}

	// Build the response
	response := &models.GetUsersResponse{}
	response.Body = allUsers

	return response, nil
}

// Get a specific user
func getUserFunc(ctx context.Context, input *models.GetUserRequest) (*models.GetUserResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Run the query
	queries := database.New(pool)
	u, err := queries.RetrieveUser(ctx, input.UserHandle)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		} else {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to get user data for user %s. %v", input.UserHandle, err))
			// return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found. %v", input.UserHandle, err))
		}
	}
	if u.UserHandle != input.UserHandle {
		return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
	}

	// Get projects the user is a member of
	projects := models.ProjectMemberships{}
	ps, err := queries.GetProjectsByUser(ctx, database.GetProjectsByUserParams{
		UserHandle: input.UserHandle,
		Limit:      maxSharedUsersPerQuery,
		Offset:     0,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			fmt.Printf("Warning: No LLM Services registered for user %s.", input.UserHandle)
		} else {
			fmt.Printf("Warning: Unable to get list of LLM Services for user %s. %v", input.UserHandle, err)
		}
	}
	for _, project := range ps {
		projects = append(projects, models.ProjectMembership{
			ProjectHandle: project.ProjectHandle,
			ProjectOwner:  project.Owner,
			Role:          project.Role,
		})
	}

	// Get LLM service instances the user is a member of
	imemberships := models.InstanceMemberships{}
	instances, err := queries.GetAccessibleInstancesByUser(ctx, database.GetAccessibleInstancesByUserParams{
		Owner:  input.UserHandle,
		Limit:  maxSharedUsersPerQuery,
		Offset: 0,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			fmt.Printf("Warning: No LLM service instances registered for user %s.", input.UserHandle)
		} else {
			fmt.Printf("Warning: Unable to get list of LLM service instances for user %s: %v", input.UserHandle, err)
		}
	}
	for _, i := range instances {
		instance, err := queries.RetrieveInstance(ctx, database.RetrieveInstanceParams{
			Owner:          i.Owner,
			InstanceHandle: i.InstanceHandle,
		})
		if err != nil {
			fmt.Printf("Warning: Unable to get details of LLM service instance %s for user %s: %v", i.InstanceHandle, input.UserHandle, err)
			continue
		}
		// Handle the case where Role might be nil (when instance is owned by user)
		role := "owner"
		if i.Role != nil {
			if r, ok := i.Role.(string); ok {
				role = r
			}
		}
		imemberships = append(imemberships, models.InstanceMembership{
			InstanceHandle: instance.InstanceHandle,
			InstanceOwner:  instance.Owner,
			Role:           role,
		})
	}

	// Build the response
	returnUser := &models.User{
		UserHandle: u.UserHandle,
		Name:       u.Name.String,
		Email:      u.Email,
		VDBKey:     u.VDBKey,
		Projects:   projects,
		Instances:  imemberships,
	}
	response := &models.GetUserResponse{}
	response.Body = *returnUser

	return response, nil
}

// Delete a specific user
func deleteUserFunc(ctx context.Context, input *models.DeleteUserRequest) (*models.DeleteUserResponse, error) {
	// Validate that _system user cannot send requests
	if err := ValidateNotSystemUser(input.UserHandle); err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	} else if pool == nil {
		return nil, huma.Error500InternalServerError("database connection pool is nil")
	}

	// Check if user exists
	if _, err = getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle}); err != nil {
		return nil, err
	}

	// Run the query
	queries := database.New(pool)
	err = queries.DeleteUser(ctx, input.UserHandle)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete user %s. %v", input.UserHandle, err))
	}

	// Build the response
	response := &models.DeleteUserResponse{}
	return response, nil
}

// RegisterUsersRoutes registers all the admin routes with the API
func RegisterUsersRoutes(pool *pgxpool.Pool, keyGen RandomKeyGenerator, api huma.API) error {
	// Define huma.Operations for each route
	putUserOp := huma.Operation{
		OperationID:   "putUser",
		Method:        http.MethodPut,
		Path:          "/v1/users/{user_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create or update a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		// MaxBodyBytes int64 `yaml:"-"` // Max size of the request body in bytes (-1 for unlimited)
		// BodyReadTimeout time.Duration `yaml:"-" // Time to wait for the request body to be read (-1 for unlimited)
		// Middlewares Middlewares `yaml:"-"` // Middleware to run before the operation, useful for logging, etc.
		Tags: []string{"admin", "users"},
	}
	postUserOp := huma.Operation{
		OperationID:   "postUser",
		Method:        http.MethodPost,
		Path:          "/v1/users",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		Tags: []string{"admin", "users"},
	}
	getUsersOp := huma.Operation{
		OperationID: "getUsers",
		Method:      http.MethodGet,
		Path:        "/v1/users",
		Summary:     "Get information about all users",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
		},
		Tags: []string{"admin", "users"},
	}
	getUserOp := huma.Operation{
		OperationID: "getUser",
		Method:      http.MethodGet,
		Path:        "/v1/users/{user_handle}",
		Summary:     "Get information about a specific user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"admin", "users"},
	}
	deleteUserOp := huma.Operation{
		OperationID:   "deleteUser",
		Method:        http.MethodDelete,
		Path:          "/v1/users/{user_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete a specific user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"admin", "users"},
	}

	// Register the routes with middleware
	huma.Register(api, putUserOp, addPoolToContext(pool, addKeyGenToContext(keyGen, putUserFunc)))
	huma.Register(api, postUserOp, addPoolToContext(pool, addKeyGenToContext(keyGen, postUserFunc)))
	huma.Register(api, getUsersOp, addPoolToContext(pool, getUsersFunc))
	huma.Register(api, getUserOp, addPoolToContext(pool, getUserFunc))
	huma.Register(api, deleteUserOp, addPoolToContext(pool, deleteUserFunc))
	return nil
}
