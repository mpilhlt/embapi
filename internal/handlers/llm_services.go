package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"

	"github.com/mpilhlt/dhamps-vdb/internal/auth"
	"github.com/mpilhlt/dhamps-vdb/internal/crypto"
	"github.com/mpilhlt/dhamps-vdb/internal/database"
	"github.com/mpilhlt/dhamps-vdb/internal/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// getEncryptionKey retrieves the encryption key, returns nil if not set (optional encryption)
func getEncryptionKey() *crypto.EncryptionKey {
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		return nil
	}
	return crypto.NewEncryptionKey(keyStr)
}

// === Sharing LLM Service Definitions ===

func putDefinitionFunc(ctx context.Context, input *models.PutDefinitionRequest) (*models.UploadDefinitionResponse, error) {
	if input.DefinitionHandle != input.Body.DefinitionHandle {
		return nil, huma.Error400BadRequest(fmt.Sprintf("definition handle in URL (\"%s\") does not match definition handle in body (\"%s\")", input.DefinitionHandle, input.Body.DefinitionHandle))
	}

	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Execute all database operations within a transaction
	var owner string
	var definitionHandle string
	var definitionID int32
	var isPublic bool

	err = database.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		queries := database.New(tx)

		// 1. Upsert LLM service definition
		llm, err := queries.UpsertDefinition(ctx, database.UpsertDefinitionParams{
			Owner:            input.UserHandle,
			DefinitionHandle: input.DefinitionHandle,
			Endpoint:         input.Body.Endpoint,
			Description:      pgtype.Text{String: input.Body.Description, Valid: true},
			APIStandard:      input.Body.APIStandard,
			Model:            input.Body.Model,
			Dimensions:       int32(input.Body.Dimensions),
			ContextLimit:     int32(input.Body.ContextLimit),
			IsPublic:         input.Body.IsPublic,
		})
		if err != nil {
			return fmt.Errorf("unable to upload llm service definition: %v", err)
		}

		owner = llm.Owner
		definitionHandle = llm.DefinitionHandle
		definitionID = llm.DefinitionID
		isPublic = llm.IsPublic

		// Ownership is tracked via the owner column in instances
		// No need for separate linking table

		return nil
	})

	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	// Build response
	response := &models.UploadDefinitionResponse{}
	response.Body.Owner = owner
	response.Body.DefinitionHandle = definitionHandle
	response.Body.DefinitionID = int(definitionID)
	response.Body.IsPublic = isPublic

	return response, nil
}

func postDefinitionFunc(ctx context.Context, input *models.PostDefinitionRequest) (*models.UploadDefinitionResponse, error) {
	return putDefinitionFunc(ctx, &models.PutDefinitionRequest{UserHandle: input.UserHandle, DefinitionHandle: input.Body.DefinitionHandle, Body: input.Body})
}

func getDefinitionFunc(ctx context.Context, input *models.GetDefinitionRequest) (*models.GetDefinitionResponse, error) {

	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("user handle in retrieved record does not match retrieve handle %s", input.UserHandle))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Unable to access database pool")
	}

	// Run the query
	queries := database.New(pool)
	def, err := queries.RetrieveDefinition(ctx, database.RetrieveDefinitionParams{
		Owner:            input.UserHandle,
		DefinitionHandle: input.DefinitionHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("llm definition %s/%s not found", input.UserHandle, input.DefinitionHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve llm definition %s/%s: %v", input.UserHandle, input.DefinitionHandle, err))
	}
	if def.DefinitionHandle != input.DefinitionHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("llm definition handle in retrieved record does not match retrieving handle %s/%s", input.UserHandle, input.DefinitionHandle))
	}
	if !def.IsPublic {
		authorized := false
		if authUserHandle, ok := ctx.Value(auth.AuthUserKey).(string); ok {
			acessibleDefinitions, err := queries.GetAccessibleDefinitionsByUser(ctx, database.GetAccessibleDefinitionsByUserParams{
				Owner:  authUserHandle,
				Limit:  999,
				Offset: 0,
			})
			if err != nil && err != pgx.ErrNoRows {
				return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve accessible definitions for user %s: %v", authUserHandle, err))
			} else if err == pgx.ErrNoRows {
				return nil, huma.Error403Forbidden(fmt.Sprintf("user %s does not have access to llm definition %s/%s", authUserHandle, input.UserHandle, input.DefinitionHandle))
			}
			for _, d := range acessibleDefinitions {
				if d.DefinitionID == def.DefinitionID {
					authorized = true
					break
				}
			}
		}
		if !authorized {
			return nil, huma.Error403Forbidden(fmt.Sprintf("user does not have access to llm definition %s/%s", input.UserHandle, input.DefinitionHandle))
		}
	}

	// Build response
	ls := models.DefinitionFull{
		Owner:            def.Owner,
		DefinitionHandle: def.DefinitionHandle,
		DefinitionID:     int(def.DefinitionID),
		Endpoint:         def.Endpoint,
		Description:      def.Description.String,
		APIStandard:      def.APIStandard,
		Model:            def.Model,
		Dimensions:       def.Dimensions,
		ContextLimit:     def.ContextLimit,
		IsPublic:         def.IsPublic,
	}
	response := &models.GetDefinitionResponse{}
	response.Body = ls

	return response, nil
}

func getUserDefinitionsFunc(ctx context.Context, input *models.GetUserDefinitionsRequest) (*models.GetUserDefinitionsResponse, error) {

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

	// Run the query - get all accessible instances (own + shared)
	def, err := queries.GetAccessibleDefinitionsByUser(ctx, database.GetAccessibleDefinitionsByUserParams{
		Owner:  input.UserHandle,
		Limit:  int32(input.Limit),
		Offset: int32(input.Offset),
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			// Return empty list instead of error
			response := &models.GetUserDefinitionsResponse{}
			response.Body.Definitions = []models.DefinitionBrief{}
			return response, nil
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve llm service definitions for user %s: %v", input.UserHandle, err))
	}

	// Build response
	ls := []models.DefinitionBrief{}
	for _, d := range def {
		ls = append(ls, models.DefinitionBrief{
			Owner:            d.Owner,
			DefinitionHandle: d.DefinitionHandle,
			DefinitionID:     int(d.DefinitionID),
			IsPublic:         d.IsPublic,
		})
	}
	response := &models.GetUserDefinitionsResponse{}
	response.Body.Definitions = ls

	return response, nil
}

func deleteDefinitionFunc(ctx context.Context, input *models.DeleteDefinitionRequest) (*models.DeleteDefinitionResponse, error) {

	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("user handle of retrieved record does not match retrieving handle %s", input.UserHandle))
	}

	// Check if llm service definition exists
	_, err = getDefinitionFunc(ctx, &models.GetDefinitionRequest{UserHandle: input.UserHandle, DefinitionHandle: input.DefinitionHandle})
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("definition %s/%s not found", input.UserHandle, input.DefinitionHandle))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}
	queries := database.New(pool)

	// Run the query
	err = queries.DeleteDefinition(ctx, database.DeleteDefinitionParams{
		Owner:            input.UserHandle,
		DefinitionHandle: input.DefinitionHandle,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete llm service definition %s for user %s: %v", input.DefinitionHandle, input.UserHandle, err))
	}

	// Build response
	response := &models.DeleteDefinitionResponse{}

	return response, nil
}

// share/unshare LLM Service Definitions

func shareDefinitionFunc(ctx context.Context, input *models.ShareDefinitionRequest) (*models.ShareDefinitionResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error accessing database connection: %v", err)
	}
	queries := database.New(pool)

	// Check if definition exists and belongs to owner
	definition, err := queries.RetrieveDefinition(ctx, database.RetrieveDefinitionParams{
		Owner:            input.UserHandle,
		DefinitionHandle: input.DefinitionHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("definition %s/%s not found", input.UserHandle, input.DefinitionHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve definition: %v", err))
	}
	if definition.Owner != ctx.Value(auth.AuthUserKey).(string) {
		return nil, huma.Error401Unauthorized(fmt.Sprintf("Not authorized to share definition %s/%s", input.UserHandle, input.DefinitionHandle))
	}

	// Check if target user exists
	_, err = getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.Body.ShareWithHandle})
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("target user %s does not exist: %v", input.Body.ShareWithHandle, err))
	}

	// Share the definition
	err = queries.LinkDefinitionToUser(ctx, database.LinkDefinitionToUserParams{
		UserHandle:   input.Body.ShareWithHandle,
		DefinitionID: definition.DefinitionID,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to share definition: %v", err))
	}

	// Build response
	sharedUsers := []string{}
	sharedUsers = append(sharedUsers, input.Body.ShareWithHandle)
	response := &models.ShareDefinitionResponse{}
	response.Body.Owner = input.UserHandle
	response.Body.DefinitionHandle = input.DefinitionHandle
	response.Body.SharedWith = sharedUsers

	return response, nil
}

func unshareDefinitionFunc(ctx context.Context, input *models.UnshareDefinitionRequest) (*models.UnshareDefinitionResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error accessing database connection: %v", err)
	}
	queries := database.New(pool)

	// Check if definition exists and belongs to owner
	definition, err := queries.RetrieveDefinition(ctx, database.RetrieveDefinitionParams{
		Owner:            input.UserHandle,
		DefinitionHandle: input.DefinitionHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("definition %s/%s not found", input.UserHandle, input.DefinitionHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve definition: %v", err))
	}
	if definition.Owner != ctx.Value(auth.AuthUserKey).(string) {
		return nil, huma.Error401Unauthorized(fmt.Sprintf("Not authorized to share definition %s/%s", input.UserHandle, input.DefinitionHandle))
	}
	fmt.Printf("Definition retrieved: %s/%s (id %d)\n", definition.Owner, definition.DefinitionHandle, definition.DefinitionID)
	fmt.Printf("Attempting to unshare with %s\n", input.UnshareWithHandle)

	// Unshare the definition
	err = queries.UnlinkDefinition(ctx, database.UnlinkDefinitionParams{
		UserHandle:   input.UnshareWithHandle,
		DefinitionID: definition.DefinitionID,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to unshare definition: %v", err))
	}

	// Build response
	response := &models.UnshareDefinitionResponse{}

	return response, nil
}

func getDefinitionSharedUsersFunc(ctx context.Context, input *models.GetDefinitionSharedUsersRequest) (*models.GetDefinitionSharedUsersResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error accessing database connection: %v", err)
	}
	queries := database.New(pool)

	// Get shared users - use input parameters for user-facing pagination
	sharedUsers, err := queries.GetSharedUsersForDefinition(ctx, database.GetSharedUsersForDefinitionParams{
		Owner:            input.UserHandle,
		DefinitionHandle: input.DefinitionHandle,
		Limit:            int32(input.Limit),
		Offset:           int32(input.Offset),
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			// Return empty list instead of error
			response := &models.GetDefinitionSharedUsersResponse{}
			response.Body = nil
			return response, nil
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve shared users: %v", err))
	}

	response := &models.GetDefinitionSharedUsersResponse{}
	response.Body = sharedUsers

	return response, nil
}

// === LLM Service Instances ===

// Create a llm service instance (with a handle being present in the URL)
func putInstanceFunc(ctx context.Context, input *models.PutInstanceRequest) (*models.UploadInstanceResponse, error) {
	if input.InstanceHandle != input.Body.InstanceHandle {
		return nil, huma.Error400BadRequest(fmt.Sprintf("instance handle in URL (\"%s\") does not match instance handle in body (\"%s\")", input.InstanceHandle, input.Body.InstanceHandle))
	}

	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Get encryption key if available
	encKey := getEncryptionKey()

	// Execute all database operations within a transaction
	var instanceID int32
	var instanceHandle string
	var owner string

	err = database.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		queries := database.New(tx)

		// Prepare API key encryption
		var APIKeyEncrypted []byte
		if input.Body.APIKey != "" && encKey != nil {
			APIKeyEncrypted, err = encKey.Encrypt(input.Body.APIKey)
			if err != nil {
				return huma.Error500InternalServerError(fmt.Sprintf("unable to encrypt API key: %v", err))
			}
		}

		// 1. Upsert LLM service instance
		llm, err := queries.UpsertInstance(ctx, database.UpsertInstanceParams{
			Owner:           input.UserHandle,
			InstanceHandle:  input.InstanceHandle,
			DefinitionID:    pgtype.Int4{Valid: false}, // Standalone instance (no definition reference)
			Endpoint:        input.Body.Endpoint,
			Description:     pgtype.Text{String: input.Body.Description, Valid: true},
			APIKeyEncrypted: APIKeyEncrypted,
			APIStandard:     input.Body.APIStandard,
			Model:           input.Body.Model,
			Dimensions:      int32(input.Body.Dimensions),
		})
		if err != nil {
			return huma.Error500InternalServerError(fmt.Sprintf("unable to upload llm service instance: %v", err))
		}

		instanceID = llm.InstanceID
		instanceHandle = llm.InstanceHandle
		owner = llm.Owner

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build response
	response := &models.UploadInstanceResponse{}
	response.Body.Owner = owner
	response.Body.InstanceHandle = instanceHandle
	response.Body.InstanceID = int(instanceID)

	return response, nil
}

// Create a llm service (without a handle being present in the URL)
func postInstanceFunc(ctx context.Context, input *models.PostInstanceRequest) (*models.UploadInstanceResponse, error) {
	return putInstanceFunc(ctx, &models.PutInstanceRequest{UserHandle: input.UserHandle, InstanceHandle: input.Body.InstanceHandle, Body: input.Body})
}

// Create a llm service instance based on a definition
func postInstanceFromDefinitionFunc(ctx context.Context, input *models.PostInstanceFromDefinitionRequest) (*models.UploadInstanceResponse, error) {
	if input.UserHandle != input.Body.UserHandle {
		return nil, huma.Error400BadRequest(fmt.Sprintf("user handle in URL (\"%s\") does not match user handle in body (\"%s\")", input.UserHandle, input.Body.UserHandle))
	}

	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to access user %s. %v", input.UserHandle, err))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Get encryption key if available
	encKey := getEncryptionKey()

	// Execute all database operations within a transaction
	var instanceID int32
	var instanceHandle string
	var owner string

	err = database.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		queries := database.New(tx)

		// Prepare API key encryption
		var APIKeyEncrypted []byte
		if input.Body.APIKey != "" && encKey != nil {
			APIKeyEncrypted, err = encKey.Encrypt(input.Body.APIKey)
			if err != nil {
				return fmt.Errorf("unable to encrypt API key: %v", err)
			}
		}

		// Get definition to base instance on
		definition, err := queries.RetrieveDefinition(ctx, database.RetrieveDefinitionParams{
			Owner:            input.Body.DefinitionOwner,
			DefinitionHandle: input.Body.DefinitionHandle,
		})
		if err != nil {
			if err.Error() == "no rows in result set" {
				return huma.Error404NotFound(fmt.Sprintf("definition %s/%s not found", input.Body.DefinitionOwner, input.Body.DefinitionHandle))
			}
			return huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve definition %s/%s: %v", input.Body.DefinitionOwner, input.Body.DefinitionHandle, err))
		}
		// Check if user has access to the definition (either owner or shared)
		if !definition.IsPublic && definition.Owner != ctx.Value(auth.AuthUserKey).(string) {
			hasAccess := false
			// Check if shared with user - use maxSharedUsersPerQuery for authorization check
			sharedUsers, err := queries.GetSharedUsersForDefinition(ctx, database.GetSharedUsersForDefinitionParams{
				Owner:            input.Body.DefinitionOwner,
				DefinitionHandle: input.Body.DefinitionHandle,
				Limit:            maxSharedUsersPerQuery,
				Offset:           0,
			})
			if err != nil && err.Error() != "no rows in result set" {
				return huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve shared users for definition %s/%s: %v", input.Body.DefinitionOwner, input.Body.DefinitionHandle, err))
			}
			if slices.Contains(sharedUsers, ctx.Value(auth.AuthUserKey).(string)) {
				hasAccess = true
			}
			if !hasAccess {
				return huma.Error401Unauthorized(fmt.Sprintf("user does not have access to definition %s/%s", input.Body.DefinitionOwner, input.Body.DefinitionHandle))
			}
		}

		// merge definition values with instance overrides from request body
		if input.Body.Endpoint == "" {
			input.Body.Endpoint = definition.Endpoint
		}
		if input.Body.Description == "" {
			input.Body.Description = definition.Description.String
		}
		if input.Body.APIStandard == "" {
			input.Body.APIStandard = definition.APIStandard
		}
		if input.Body.Model == "" {
			input.Body.Model = definition.Model
		}
		if input.Body.Dimensions == 0 {
			input.Body.Dimensions = definition.Dimensions
		}
		if input.Body.ContextLimit == 0 {
			input.Body.ContextLimit = definition.ContextLimit
		}

		// 1. Upsert LLM service instance
		llm, err := queries.UpsertInstance(ctx, database.UpsertInstanceParams{
			Owner:           input.UserHandle,
			InstanceHandle:  input.Body.InstanceHandle,
			DefinitionID:    pgtype.Int4{Int32: int32(definition.DefinitionID), Valid: true}, // Standalone instance (no definition reference)
			Endpoint:        input.Body.Endpoint,
			Description:     pgtype.Text{String: input.Body.Description, Valid: true},
			APIKeyEncrypted: APIKeyEncrypted,
			APIStandard:     input.Body.APIStandard,
			Model:           input.Body.Model,
			Dimensions:      int32(input.Body.Dimensions),
			ContextLimit:    int32(input.Body.ContextLimit),
		})
		if err != nil {
			return fmt.Errorf("unable to upload llm service instance: %v", err)
		}

		instanceID = llm.InstanceID
		instanceHandle = llm.InstanceHandle
		owner = llm.Owner

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build response
	response := &models.UploadInstanceResponse{}
	response.Body.Owner = owner
	response.Body.InstanceHandle = instanceHandle
	response.Body.InstanceID = int(instanceID)

	return response, nil
}

func getInstanceFunc(ctx context.Context, input *models.GetInstanceRequest) (*models.GetInstanceResponse, error) {
	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("user handle in retrieved record does not match retrieve handle %s", input.UserHandle))
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Unable to access database pool")
	}

	// Run the query
	queries := database.New(pool)
	llm, err := queries.RetrieveInstance(ctx, database.RetrieveInstanceParams{
		Owner:          input.UserHandle,
		InstanceHandle: input.InstanceHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("llm service %s for user %s not found", input.InstanceHandle, input.UserHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve llm service %s for user %s: %v", input.InstanceHandle, input.UserHandle, err))
	}
	if llm.InstanceHandle != input.InstanceHandle {
		return nil, huma.Error404NotFound(fmt.Sprintf("llm service %s for user %s not found", input.InstanceHandle, input.UserHandle))
	}

	// Retrieve the authenticated user's access role
	accessRole := ""
	if authUserHandle, ok := ctx.Value(auth.AuthUserKey).(string); ok {
		acessibleInstances, err := queries.GetAccessibleInstancesByUser(ctx, database.GetAccessibleInstancesByUserParams{
			Owner:  authUserHandle,
			Limit:  maxSharedUsersPerQuery,
			Offset: 0,
		})
		if err != nil && err != pgx.ErrNoRows {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve accessible instances for user %s: %v", authUserHandle, err))
		} else if err == pgx.ErrNoRows {
			return nil, huma.Error403Forbidden(fmt.Sprintf("user %s does not have access to llm service %s/%s", authUserHandle, input.UserHandle, input.InstanceHandle))
		}
		found := false
		for _, inst := range acessibleInstances {
			if inst.InstanceID == llm.InstanceID {
				found = true
				accessRole = inst.Role.(string)
				break
			}
		}
		if !found {
			return nil, huma.Error403Forbidden(fmt.Sprintf("user %s does not have access to llm service %s/%s", authUserHandle, input.UserHandle, input.InstanceHandle))
		}
	} else {
		// No authenticated user in context, only possible if public access is allowed (not implemented)
		return nil, huma.Error403Forbidden("no authenticated user in context")
	}

	defID := int32(0)
	if llm.DefinitionID.Valid {
		defID = llm.DefinitionID.Int32
	}

	// Build response (never return API key in plaintext)
	ls := models.InstanceFull{
		Owner:            llm.Owner,
		InstanceHandle:   llm.InstanceHandle,
		InstanceID:       int(llm.InstanceID),
		AccessRole:       accessRole,
		DefinitionID:     int(defID),
		DefinitionOwner:  llm.DefinitionOwner.String,
		DefinitionHandle: llm.DefinitionHandle.String,
		Endpoint:         llm.Endpoint,
		Description:      llm.Description.String,
		HasAPIKey:        llm.HasAPIKey,
		// APIKey:         "", // Never return API key
		APIStandard:  llm.APIStandard,
		Model:        llm.Model,
		Dimensions:   llm.Dimensions,
		ContextLimit: llm.ContextLimit,
	}
	response := &models.GetInstanceResponse{}
	response.Body = ls

	return response, nil
}

func getUserInstancesFunc(ctx context.Context, input *models.GetUserInstancesRequest) (*models.GetUserInstancesResponse, error) {

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

	// Run the query - get all accessible instances (own + shared)
	llms, err := queries.GetAccessibleInstancesByUser(ctx, database.GetAccessibleInstancesByUserParams{
		Owner:  input.UserHandle,
		Limit:  int32(input.Limit),
		Offset: int32(input.Offset),
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			// Return empty list instead of error
			response := &models.GetUserInstancesResponse{}
			response.Body.Instances = []models.InstanceBrief{}
			return response, nil
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve llm service instances: %v", err))
	}

	// Build response (hide API keys for shared instances)
	ls := []models.InstanceBrief{}
	for _, llm := range llms {
		ls = append(ls, models.InstanceBrief{
			Owner:          llm.Owner,
			InstanceHandle: llm.InstanceHandle,
			InstanceID:     int(llm.InstanceID),
		})
	}
	response := &models.GetUserInstancesResponse{}
	response.Body.Instances = ls

	return response, nil
}

func deleteInstanceFunc(ctx context.Context, input *models.DeleteInstanceRequest) (*models.DeleteInstanceResponse, error) {
	// Check if user exists
	u, err := getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.UserHandle})
	if err != nil {
		return nil, err
	}
	if u.Body.UserHandle != input.UserHandle {
		return nil, huma.Error404NotFound(fmt.Sprintf("user %s not found", input.UserHandle))
	}

	// Check if llm service instance exists
	_, err = getInstanceFunc(ctx, &models.GetInstanceRequest{UserHandle: input.UserHandle, InstanceHandle: input.InstanceHandle})
	if err != nil {
		return nil, err
	}

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}

	// Run the query
	queries := database.New(pool)
	err = queries.DeleteInstance(ctx, database.DeleteInstanceParams{
		Owner:          input.UserHandle,
		InstanceHandle: input.InstanceHandle,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to delete llm service %s for user %s: %v", input.InstanceHandle, input.UserHandle, err))
	}

	// Build response
	response := &models.DeleteInstanceResponse{}

	return response, nil
}

// share/unshare LLM Service Instances

func shareInstanceFunc(ctx context.Context, input *models.ShareInstanceRequest) (*models.ShareInstanceResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}
	queries := database.New(pool)

	// Check if ShareWithUser is identical to owner (no need to share if sharing with self)
	if input.Body.ShareWithHandle == input.UserHandle {
		return nil, huma.Error400BadRequest("cannot share instance with owner")
	}
	// Check if role is valid
	if input.Body.Role != "editor" && input.Body.Role != "reader" {
		return nil, huma.Error400BadRequest(fmt.Sprintf("invalid role %s. Role must be either \"editor\" or \"reader\"", input.Body.Role))
	}
	// Check if instance exists
	instance, err := queries.RetrieveInstance(ctx, database.RetrieveInstanceParams{
		Owner:          input.UserHandle,
		InstanceHandle: input.InstanceHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("instance %s/%s not found", input.UserHandle, input.InstanceHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve instance %s/%s: %v", input.UserHandle, input.InstanceHandle, err))
	}
	// Check if instance belongs to current user (only owner can share)
	if instance.Owner != ctx.Value(auth.AuthUserKey).(string) {
		return nil, huma.Error401Unauthorized(fmt.Sprintf("Not authorized to share instance %s/%s", input.UserHandle, input.InstanceHandle))
	}
	// Check if target user exists
	_, err = getUserFunc(ctx, &models.GetUserRequest{UserHandle: input.Body.ShareWithHandle})
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("target user %s does not exist: %v", input.Body.ShareWithHandle, err))
	}

	// Share the instance
	err = queries.LinkInstanceToUser(ctx, database.LinkInstanceToUserParams{
		UserHandle: input.Body.ShareWithHandle,
		InstanceID: instance.InstanceID,
		Role:       input.Body.Role,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to share instance: %v", err))
	}

	// Build response
	// (only the instance owner can share, so we know they have sent the request,
	//  meaning we can show all shared users)
	// TODO: validate: retrieve shared users from database instead of just returning the input values
	sharedUsers := []models.SharedUser{}
	sharedUsers = append(sharedUsers, models.SharedUser{
		UserHandle: input.Body.ShareWithHandle,
		Role:       input.Body.Role,
	})
	response := &models.ShareInstanceResponse{}
	response.Body.Owner = input.UserHandle
	response.Body.InstanceHandle = input.InstanceHandle
	response.Body.SharedWith = sharedUsers

	return response, nil
}

func unshareInstanceFunc(ctx context.Context, input *models.UnshareInstanceRequest) (*models.UnshareInstanceResponse, error) {
	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}
	queries := database.New(pool)

	// Check if instance exists and belongs to owner
	instance, err := queries.RetrieveInstance(ctx, database.RetrieveInstanceParams{
		Owner:          input.UserHandle,
		InstanceHandle: input.InstanceHandle,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("instance %s/%s not found", input.UserHandle, input.InstanceHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve instance: %v", err))
	}

	// Check if target user exists and is currently shared
	// Use maxSharedUsersPerQuery for internal check to ensure we scan all shared users
	sharedUsers, err := queries.GetSharedUsersForInstance(ctx, database.GetSharedUsersForInstanceParams{
		Owner:          input.UserHandle,
		InstanceHandle: input.InstanceHandle,
		Limit:          maxSharedUsersPerQuery,
		Offset:         0,
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, huma.Error404NotFound(fmt.Sprintf("instance %s/%s is not shared with user %s", input.UserHandle, input.InstanceHandle, input.UnshareWithHandle))
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve shared users for instance: %v", err))
	}
	for _, su := range sharedUsers {
		if su.UserHandle == input.UnshareWithHandle {
			// Unshare the instance
			err = queries.UnlinkInstance(ctx, database.UnlinkInstanceParams{
				UserHandle: input.UnshareWithHandle,
				InstanceID: instance.InstanceID,
			})
			if err != nil {
				return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to unshare instance %s/%s from user %s: %v", input.UserHandle, input.InstanceHandle, input.UnshareWithHandle, err))
			}
			// Build response
			response := &models.UnshareInstanceResponse{}
			return response, nil
		}
	}
	// If we get here, the target user exists but is not currently shared
	return nil, huma.Error404NotFound(fmt.Sprintf("instance %s/%s is not shared with user %s", input.UserHandle, input.InstanceHandle, input.UnshareWithHandle))
}

func getInstanceSharedUsersFunc(ctx context.Context, input *models.GetInstanceSharedUsersRequest) (*models.GetInstanceSharedUsersResponse, error) {

	// Get the database connection pool from the context
	pool, err := GetDBPool(ctx)
	if err != nil {
		return nil, err
	}
	queries := database.New(pool)

	// Get shared users - use input parameters for user-facing pagination
	sharedUsers, err := queries.GetSharedUsersForInstance(ctx, database.GetSharedUsersForInstanceParams{
		Owner:          input.UserHandle,
		InstanceHandle: input.InstanceHandle,
		Limit:          int32(input.Limit),
		Offset:         int32(input.Offset),
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			// Return empty list instead of error
			response := &models.GetInstanceSharedUsersResponse{}
			response.Body.SharedWith = []models.SharedUser{}
			return response, nil
		}
		return nil, huma.Error500InternalServerError(fmt.Sprintf("unable to retrieve shared users: %v", err))
	}

	// Build response
	users := []models.SharedUser{}
	for _, su := range sharedUsers {
		users = append(users, models.SharedUser{
			UserHandle: su.UserHandle,
			Role:       su.Role,
		})
	}

	response := &models.GetInstanceSharedUsersResponse{}
	response.Body.Owner = input.UserHandle
	response.Body.InstanceHandle = input.InstanceHandle
	response.Body.SharedWith = users

	return response, nil
}

// === Registration of Routes ===

// RegisterDefinitionsRoutes registers the routes for the management of LLM service definitions
func RegisterDefinitionsRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	putDefinitionOp := huma.Operation{
		OperationID:   "putDefinition",
		Method:        http.MethodPut,
		Path:          "/v1/llm-definitions/{user_handle}/{definition_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create or update an llm service definition",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}
	postDefinitionOp := huma.Operation{
		OperationID:   "postDefinition",
		Method:        http.MethodPost,
		Path:          "/v1/llm-definitions/{user_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create an llm service definition (with auto-generated handle)",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}
	getDefinitionOp := huma.Operation{
		OperationID: "getDefinition",
		Method:      http.MethodGet,
		Path:        "/v1/llm-definitions/{user_handle}/{definition_handle}",
		Summary:     "Get an llm service definition",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"llm-definitions"},
	}
	getDefinitionsOp := huma.Operation{
		OperationID: "getDefinitions",
		Method:      http.MethodGet,
		Path:        "/v1/llm-definitions/{user_handle}",
		Summary:     "Get all llm service definitions for a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}
	deleteDefinitionOp := huma.Operation{
		OperationID:   "deleteDefinition",
		Method:        http.MethodDelete,
		Path:          "/v1/llm-definitions/{user_handle}/{definition_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete an llm service definition",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}
	shareDefinitionOp := huma.Operation{
		OperationID:   "shareDefinition",
		Method:        http.MethodPost,
		Path:          "/v1/llm-definitions/{user_handle}/{definition_handle}/share",
		DefaultStatus: http.StatusCreated,
		Summary:       "Share an llm service definition with another user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}
	unshareDefinitionOp := huma.Operation{
		OperationID:   "unshareDefinition",
		Method:        http.MethodDelete,
		Path:          "/v1/llm-definitions/{user_handle}/{definition_handle}/share/{unshare_with_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Unshare an llm service definition from a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}
	getDefinitionSharedUsersOp := huma.Operation{
		OperationID: "getDefinitionSharedUsers",
		Method:      http.MethodGet,
		Path:        "/v1/llm-definitions/{user_handle}/{definition_handle}/shared-with",
		Summary:     "Get list of users a definition is shared with",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-definitions"},
	}

	huma.Register(api, putDefinitionOp, addPoolToContext(pool, putDefinitionFunc))
	huma.Register(api, postDefinitionOp, addPoolToContext(pool, postDefinitionFunc))
	huma.Register(api, getDefinitionOp, addPoolToContext(pool, getDefinitionFunc))
	huma.Register(api, getDefinitionsOp, addPoolToContext(pool, getUserDefinitionsFunc))
	huma.Register(api, deleteDefinitionOp, addPoolToContext(pool, deleteDefinitionFunc))
	huma.Register(api, shareDefinitionOp, addPoolToContext(pool, shareDefinitionFunc))
	huma.Register(api, unshareDefinitionOp, addPoolToContext(pool, unshareDefinitionFunc))
	huma.Register(api, getDefinitionSharedUsersOp, addPoolToContext(pool, getDefinitionSharedUsersFunc))
	return nil
}

// RegisterInstancesRoutes registers the routes for the management of LLM service instances
func RegisterInstancesRoutes(pool *pgxpool.Pool, api huma.API) error {
	// Define huma.Operations for each route
	postInstanceOp := huma.Operation{
		OperationID:   "postInstance",
		Method:        http.MethodPost,
		Path:          "/v1/llm-instances/{user_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create llm service instance",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}
	putInstanceOp := huma.Operation{
		OperationID:   "putInstance",
		Method:        http.MethodPut,
		Path:          "/v1/llm-instances/{user_handle}/{instance_handle}",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create or update llm service instance",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}
	postInstanceFromDefinitionOp := huma.Operation{
		OperationID: "postInstanceFromDefinition",
		Method:      http.MethodPost,
		Path:        "/v1/llm-instances/{user_handle}/from-definition",
		Summary:     "Create an llm service instance based on a definition",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}
	getUserInstancesOp := huma.Operation{
		OperationID: "getUserInstances",
		Method:      http.MethodGet,
		Path:        "/v1/llm-instances/{user_handle}",
		Summary:     "Get all llm service instances for a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"llm-instances"},
	}
	getInstanceOp := huma.Operation{
		OperationID: "getInstance",
		Method:      http.MethodGet,
		Path:        "/v1/llm-instances/{user_handle}/{instance_handle}",
		Summary:     "Get a specific llm service instance for a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
			{"readerAuth": []string{"reader"}},
		},
		Tags: []string{"llm-instances"},
	}
	deleteInstanceOp := huma.Operation{
		OperationID:   "deleteInstance",
		Method:        http.MethodDelete,
		Path:          "/v1/llm-instances/{user_handle}/{instance_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Delete a user's llm service instance and all embeddings associated to it",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}
	shareInstanceOp := huma.Operation{
		OperationID:   "shareInstance",
		Method:        http.MethodPost,
		Path:          "/v1/llm-instances/{user_handle}/{instance_handle}/share",
		DefaultStatus: http.StatusCreated,
		Summary:       "Share an llm service instance with another user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}
	unshareInstanceOp := huma.Operation{
		OperationID:   "unshareInstance",
		Method:        http.MethodDelete,
		Path:          "/v1/llm-instances/{user_handle}/{instance_handle}/share/{unshare_with_handle}",
		DefaultStatus: http.StatusNoContent,
		Summary:       "Unshare an llm service instance from a user",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}
	getInstanceSharedUsersOp := huma.Operation{
		OperationID: "getInstanceSharedUsers",
		Method:      http.MethodGet,
		Path:        "/v1/llm-instances/{user_handle}/{instance_handle}/shared-with",
		Summary:     "Get list of users an instance is shared with",
		Security: []map[string][]string{
			{"adminAuth": []string{"admin"}},
			{"ownerAuth": []string{"owner"}},
		},
		Tags: []string{"llm-instances"},
	}

	huma.Register(api, postInstanceOp, addPoolToContext(pool, postInstanceFunc))
	huma.Register(api, putInstanceOp, addPoolToContext(pool, putInstanceFunc))
	huma.Register(api, postInstanceFromDefinitionOp, addPoolToContext(pool, postInstanceFromDefinitionFunc))
	huma.Register(api, getUserInstancesOp, addPoolToContext(pool, getUserInstancesFunc))
	huma.Register(api, getInstanceOp, addPoolToContext(pool, getInstanceFunc))
	huma.Register(api, deleteInstanceOp, addPoolToContext(pool, deleteInstanceFunc))
	huma.Register(api, shareInstanceOp, addPoolToContext(pool, shareInstanceFunc))
	huma.Register(api, unshareInstanceOp, addPoolToContext(pool, unshareInstanceFunc))
	huma.Register(api, getInstanceSharedUsersOp, addPoolToContext(pool, getInstanceSharedUsersFunc))
	return nil
}
