package models

import (
	"net/http"
)

/*
  LLM Services are manage via LLM Service Definitions and LLM Service Instances.

  While the Definitions serve as templates and a couple of them are provided by
	the "_system" account for all users to use, the Instances provide fully
	specified connectionInstance details, including personal or project API keys for
	Embedding service providers and can - as soon as the
	respective function is implemented - be used to have EmbAPI forward texts to
	the embedding platform. This can be useful either to create the embeddings to
	store in EmbAPI in the first place, or to encode unseen data that
	similarities of stored embeddings can then be calculated against.

	Both Definitions and Instances can be shared with other users. API keys are
	recorded only for Instances, saved only in an encrypted way and never
	displayed in any output of EmbAPI. (Thus, make sure to keep your own backup
	copy in some secure location, don't rely on EmbAPI to be able to tell you
	your API key in case you forget it.)

	With regard to terminology, the following models involve "owner" and "user_handle":
	in most cases, both refer to the same entity: the owner of the resource.
	We use "user_handle" preferably in the API paths (i.e. in Input structs)
	and "owner" in the data model (and output JSON).
*/

// === I. LLM Service Definitions ===

// Definition represents a template for LLM service configurations
// Definitions can be owned by _system (global templates) or individual users
type DefinitionFull struct {
	DefinitionID     int    `json:"definition_id,omitempty" readOnly:"true" doc:"Unique LLM Service Definition identifier" example:"42"`
	DefinitionHandle string `json:"definition_handle" minLength:"3" maxLength:"20" example:"openai-large" doc:"LLM Service Definition handle"`
	Owner            string `json:"owner" readOnly:"true" doc:"User handle of the LLM Service Definition owner (_system for global)" example:"_system"`
	Endpoint         string `json:"endpoint,omitempty" example:"https://api.openai.com/v1/embeddings" doc:"LLM Service endpoint"`
	Description      string `json:"description,omitempty" doc:"LLM Service description"`
	APIStandard      string `json:"api_standard" example:"openai" doc:"Standard of the API"`
	Model            string `json:"model,omitempty" example:"text-embedding-3-large" doc:"Embedding model name"`
	Dimensions       int32  `json:"dimensions,omitempty" example:"3072" doc:"Number of dimensions in the embeddings"`
	ContextLimit     int32  `json:"context_limit,omitempty" example:"8192" doc:"Context limit of the LLM service"`
	IsPublic         bool   `json:"is_public,omitempty" default:"false" doc:"Whether the definition is public (shared with all users)"`
}

type DefinitionBrief struct {
	DefinitionID      int    `json:"definition_id" readOnly:"true" doc:"Unique LLM Service Definition identifier" example:"42"`
	DefinitionHandle  string `json:"definition_handle" minLength:"3" maxLength:"20" example:"openai-large" doc:"LLM Service Definition handle"`
	Owner             string `json:"owner" readOnly:"true" doc:"User handle of the LLM Service Definition owner (_system for global)" example:"_system"`
	IsPublic          bool   `json:"is_public" doc:"Whether the definition is public (shared with all users)"`
	NumberOfInstances int    `json:"number_of_instances" readOnly:"true" doc:"Number of instances using this definition"`
}

// Request and Response structs for the LLM Service Instance administration API
// The huma framework requires that:
// - request structs are structs with fields for the request path/query/header/cookie parameters and/or body.
// - response structs are structs with fields for the output headers and body of the operation, if any.

// Create/update llm-definition
// PUT Path: "/v1/llm-definitions/{user_handle}/{definition_handle}"

type PutDefinitionRequest struct {
	UserHandle       string         `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	DefinitionHandle string         `json:"definition_handle" path:"definition_handle" maxLength:"20" minLength:"3" example:"openai-large" doc:"LLM Service Definition handle"`
	Body             DefinitionFull `json:"body" doc:"LLM Service Definition to create or update"`
}

// POST Path: "/v1/llm-definitions/{user_handle}"
type PostDefinitionRequest struct {
	UserHandle string         `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Body       DefinitionFull `json:"body" doc:"LLM Service Definition to create"`
}

type UploadDefinitionResponse struct {
	Header []http.Header   `json:"header,omitempty" doc:"Response headers"`
	Body   DefinitionBrief `json:"definition" doc:"Information about the created LLM Service Definition"`
}

// Get single LLM Service Definition
// Path: "/v1/llm-definitions/{user_handle}/{definition_handle}"

type GetDefinitionRequest struct {
	UserHandle       string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	DefinitionHandle string `json:"definition_handle" path:"definition_handle" maxLength:"20" minLength:"3" example:"openai-large" doc:"LLM Service Definition handle"`
}

type GetDefinitionResponse struct {
	Header []http.Header  `json:"header,omitempty" doc:"Response headers"`
	Body   DefinitionFull `json:"definition" doc:"Information about the LLM Service Definition"`
}

// Get all LLM Service Definitions by user (i.e. owner)
// Path: "/v1/llm-definitions/{user_handle}"

type GetUserDefinitionsRequest struct {
	UserHandle string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Limit      int    `json:"limit,omitempty" query:"limit" minimum:"1" maximum:"200" example:"10" default:"20" doc:"Maximum number of embeddings to return"`
	Offset     int    `json:"offset,omitempty" query:"offset" minimum:"0" example:"0" default:"0" doc:"Offset into the list of embeddings"`
}

type GetUserDefinitionsResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   DefinitionsResponse
}

type DefinitionsResponse struct {
	Definitions []DefinitionBrief `json:"definitions" doc:"List of LLM Service Definitions owned by the user"`
}

// Delete LLM Service Definition
// Path: "/v1/llm-definitions/{user_handle}/{definition_handle}"

type DeleteDefinitionRequest struct {
	UserHandle       string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	DefinitionHandle string `json:"definition_handle" path:"definition_handle" maxLength:"20" minLength:"3" example:"openai-large" doc:"LLM Service Definition handle"`
}

type DeleteDefinitionResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
}

// Share Definition with User
// POST Path: "/v1/llm-definitions/{user_handle}/{definition_handle}/share"

type ShareDefinitionRequest struct {
	UserHandle       string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"_system" doc:"Definition owner handle"`
	DefinitionHandle string `json:"definition_handle" path:"definition_handle" maxLength:"20" minLength:"3" example:"openai-large" doc:"Definition handle"`
	Body             struct {
		ShareWithHandle string `json:"share_with_handle" minLength:"3" maxLength:"20" example:"bob" doc:"User handle to share with"`
	}
}

type ShareDefinitionResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Owner            string   `json:"owner" doc:"Definition owner"`
		DefinitionHandle string   `json:"definition_handle" doc:"Definition handle"`
		SharedWith       []string `json:"shared_with" doc:"List of users this definition is shared with"`
	}
}

// Unshare Definition from User
// DELETE Path: "/v1/llm-definitions/{user_handle}/{definition_handle}/share/{unshare_with_handle}"

type UnshareDefinitionRequest struct {
	UserHandle        string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"_system" doc:"Definition owner handle"`
	DefinitionHandle  string `json:"definition_handle" path:"definition_handle" maxLength:"20" minLength:"3" example:"openai-large" doc:"Definition handle"`
	UnshareWithHandle string `json:"unshare_with_handle" path:"unshare_with_handle" maxLength:"20" minLength:"3" example:"bob" doc:"User handle to unshare from"`
}

type UnshareDefinitionResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
}

// Get users a Definition is shared with (only for Definition owner)
// GET Path: "/v1/llm-definitions/{user_handle}/{definition_handle}/shared-with"

type GetDefinitionSharedUsersRequest struct {
	UserHandle       string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"_system" doc:"Definition owner handle"`
	DefinitionHandle string `json:"definition_handle" path:"definition_handle" maxLength:"20" minLength:"3" example:"openai-large" doc:"Definition handle"`
	Limit            int    `json:"limit,omitempty" query:"limit" minimum:"1" maximum:"200" example:"20" default:"100" doc:"Maximum number of users to return"`
	Offset           int    `json:"offset,omitempty" query:"offset" minimum:"0" example:"0" default:"0" doc:"Offset into the list of users"`
}

type GetDefinitionSharedUsersResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   []string      `json:"shared_with" doc:"List of users this definition is shared with"`
}

// === II. LLM Service Instances ===

// Instance represents a user-specific instance of an LLM service
// Instances can be based on a definition or standalone
type Instance struct {
	Owner            string       `json:"owner" readOnly:"true" doc:"User handle of the LLM Service Instance owner"`
	InstanceHandle   string       `json:"instance_handle" minLength:"3" maxLength:"20" example:"my-openai-large" doc:"LLM Service Instance handle"`
	InstanceID       int          `json:"instance_id,omitempty" readOnly:"true" doc:"Unique LLM Service Instance identifier" example:"153"`
	DefinitionID     *int         `json:"definition_id,omitempty" doc:"Reference to LLM Service Definition handle (if based on one)"`
	DefinitionOwner  string       `json:"definition_owner,omitempty" readOnly:"true" doc:"User handle of the LLM Service Definition owner (if based on one)"`
	DefinitionHandle string       `json:"definition_handle,omitempty" readOnly:"true" doc:"Handle of the LLM Service Definition (if based on one)"`
	Endpoint         string       `json:"endpoint,omitempty" example:"https://api.openai.com/v1/embeddings" doc:"LLM Service endpoint"`
	Description      string       `json:"description,omitempty" doc:"LLM Service description"`
	APIKeyEncrypted  string       `json:"api_key_encrypted,omitempty" writeOnly:"true" doc:"Authentication token (write-only, never returned)"`
	HasAPIKey        bool         `json:"has_api_key,omitempty" readOnly:"true" doc:"Indicates if Instance has an API key configured"`
	APIStandard      string       `json:"api_standard,omitempty" example:"openai" doc:"Standard of the API"`
	Model            string       `json:"model,omitempty" example:"text-embedding-3-large" doc:"Embedding model name"`
	Dimensions       int32        `json:"dimensions,omitempty" example:"3072" doc:"Number of dimensions in the embeddings"`
	ContextLimit     int32        `json:"context_limit,omitempty" example:"8192" doc:"Context limit of the LLM service"`
	IsPublic         bool         `json:"is_public,omitempty" default:"false" doc:"Whether the instance is public (shared with all users)"`
	SharedWith       []SharedUser `json:"shared_with,omitempty" readOnly:"true" doc:"Users this LLM Service Instance is shared with"`
	// RateLimits			 []RateLimit `json:"rate_limits,omitempty" readOnly:"true" doc:"Rate limits configured for this LLM Service Instance"``
	// ContextData      string `json:"contextData,omitempty" doc:"Context data that can be fed to the LLM service. Available in the request template as contextData variable."`
	// SystemPrompt     string `json:"systemPrompt,omitempty" example:"Return the embeddings for the following text:" doc:"System prompt for requests to the service. Available in the request template as systemPrompt variable."`
	// RequestTemplate  string `json:"requestTemplate,omitempty" doc:"Request template for the service. Can use input, contextData, and systemPrompt variables." example:"{\"input\": \"{{ input }}\", \"model\": \"text-embedding-3-small\"}"`
	// RespFieldName    string `json:"respFieldName,omitempty" default:"embedding" example:"embedding" doc:"Field name of the service response containing the embeddings. Supported is a top-level key of a json object."`
}

type InstanceInput struct {
	UserHandle       string `json:"user_handle,omitempty" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle of LLM Service Instance owner"`
	InstanceHandle   string `json:"instance_handle" minLength:"3" maxLength:"20" example:"GPT-4 API" doc:"LLM Service Instance handle"`
	InstanceID       int    `json:"instance_id,omitempty" doc:"Unique LLM Service Instance identifier" example:"153"`
	DefinitionOwner  string `json:"definition_owner,omitempty" readOnly:"true" doc:"User handle of the LLM Service Definition owner (if based on one)"`
	DefinitionHandle string `json:"definition_handle,omitempty" readOnly:"true" doc:"Handle of the LLM Service Definition (if based on one)"`
	Endpoint         string `json:"endpoint,omitempty" example:"https://api.openai.com/v1/embeddings" doc:"LLM Service endpoint"`
	Description      string `json:"description,omitempty" doc:"LLM Service Instance description"`
	APIStandard      string `json:"api_standard,omitempty" default:"openai" example:"openai" doc:"Standard of the API"`
	Model            string `json:"model,omitempty" example:"text-embedding-3-large" doc:"Embedding model name"`
	Dimensions       int32  `json:"dimensions,omitempty" example:"3072" doc:"Number of dimensions in the embeddings"`
	ContextLimit     int32  `json:"context_limit,omitempty" example:"8192" doc:"Context limit of the LLM service"`
	APIKey           string `json:"api_key,omitempty" example:"12345678901234567890123456789012" doc:"Authentication token for the service (will be saved in encrypted form only)"`
	// RateLimits			 []RateLimit `json:"rate_limits,omitempty" readOnly:"true" doc:"Rate limits configured for this LLM Service Instance"``
	// ContextData      string `json:"contextData,omitempty" doc:"Context data that can be fed to the LLM service. Available in the request template as contextData variable."`
	// SystemPrompt     string `json:"systemPrompt,omitempty" example:"Return the embeddings for the following text:" doc:"System prompt for requests to the service. Available in the request template as systemPrompt variable."`
	// RequestTemplate  string `json:"requestTemplate,omitempty" doc:"Request template for the service. Can use input, contextData, and systemPrompt variables." example:"{\"input\": \"{{ input }}\", \"model\": \"text-embedding-3-small\"}"`
	// RespFieldName    string `json:"respFieldName,omitempty" default:"embedding" example:"embedding" doc:"Field name of the service response containing the embeddings. Supported is a top-level key of a json object."`
}

type InstanceBrief struct {
	Owner               string `json:"owner" readOnly:"true" doc:"User handle of the LLM Service Instance owner"`
	InstanceHandle      string `json:"instance_handle" minLength:"3" maxLength:"20" example:"my-openai-large" doc:"LLM Service Instance handle"`
	InstanceID          int    `json:"instance_id" readOnly:"true" doc:"Unique LLM Service Instance identifier" example:"153"`
	AccessRole          string `json:"access_role,omitempty" readOnly:"true" doc:"Access role of the requesting user for this instance (owner, editor, reader)"`
	NumberOfProjects    int    `json:"number_of_projects" readOnly:"true" doc:"Number of projects using this instance"`
	NumberOfSharedUsers int    `json:"number_of_shared_users" readOnly:"true" doc:"Number of users this instance is shared with"`
}

// In Output, never return the API key
type InstanceFull struct {
	Owner            string       `json:"owner" readOnly:"true" doc:"User handle of the LLM Service Instance owner"`
	InstanceHandle   string       `json:"instance_handle" minLength:"3" maxLength:"20" example:"my-openai-large" doc:"LLM Service Instance handle"`
	InstanceID       int          `json:"instance_id" readOnly:"true" doc:"Unique LLM Service Instance identifier" example:"153"`
	AccessRole       string       `json:"access_role,omitempty" readOnly:"true" doc:"Access role of the requesting user for this instance (owner, editor, reader)"`
	DefinitionID     int          `json:"definition_id,omitempty" doc:"Reference to LLM Service Definition (if based on one)"`
	DefinitionOwner  string       `json:"definition_owner,omitempty" readOnly:"true" doc:"User handle of the LLM Service Definition owner (if based on one)"`
	DefinitionHandle string       `json:"definition_handle,omitempty" readOnly:"true" doc:"Handle of the LLM Service Definition (if based on one)"`
	Endpoint         string       `json:"endpoint,omitempty" example:"https://api.openai.com/v1/embeddings" doc:"LLM Service endpoint"`
	Description      string       `json:"description,omitempty" doc:"LLM Service Instance description"`
	HasAPIKey        bool         `json:"has_api_key,omitempty" readOnly:"true" doc:"Indicates if the LLM Service Instance has an API key configured"`
	APIStandard      string       `json:"api_standard,omitempty" example:"openai" doc:"Standard of the API"`
	Model            string       `json:"model,omitempty" example:"text-embedding-3-large" doc:"Embedding model name"`
	Dimensions       int32        `json:"dimensions,omitempty" example:"3072" doc:"Number of dimensions in the embeddings"`
	ContextLimit     int32        `json:"context_limit,omitempty" example:"8192" doc:"Maximum context length supported by the model"`
	SharedWith       []SharedUser `json:"shared_with,omitempty" readOnly:"true" doc:"Users this instance is shared with"` // This should only be reported when the request comes from the Instance owner
	IsShared         bool         `json:"is_shared,omitempty" readOnly:"true" doc:"Indicates if this is a shared instance (not owned by requesting user)"`
	// RateLimits       []RateLimit `json:"rate_limits,omitempty" readOnly:"true" doc:"Rate limits configured for this LLM Service Instance"``
	// ContextData      string `json:"contextData,omitempty" doc:"Context data that can be fed to the LLM service. Available in the request template as contextData variable."`
	// SystemPrompt     string `json:"systemPrompt,omitempty" example:"Return the embeddings for the following text:" doc:"System prompt for requests to the service. Available in the request template as systemPrompt variable."`
	// RequestTemplate  string `json:"requestTemplate,omitempty" doc:"Request template for the service. Can use input, contextData, and systemPrompt variables." example:"{\"input\": \"{{ input }}\", \"model\": \"text-embedding-3-small\"}"`
	// RespFieldName    string `json:"respFieldName,omitempty" default:"embedding" example:"embedding" doc:"Field name of the service response containing the embeddings. Supported is a top-level key of a json object."`
}

// Request and Response structs for the LLM Service Instance administration API
// The huma framework requires that:
// - request structs are structs with fields for the request path/query/header/cookie parameters and/or body.
// - response structs are structs with fields for the output headers and body of the operation, if any.

// Create/Update LLM Service Instance

// PUT Path: "/v1/llm-instances/{user_handle}/{instance_handle}"

type PutInstanceRequest struct {
	UserHandle     string        `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	InstanceHandle string        `json:"instance_handle" path:"instance_handle" maxLength:"20" minLength:"3" example:"my-gpt-4" doc:"LLM Service Instance handle"`
	Body           InstanceInput `json:"instance" doc:"LLM Service Instance to create or update"`
}

// POST Path: "/v1/llm-instances/{user_handle}"

type PostInstanceRequest struct {
	UserHandle string        `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Body       InstanceInput `json:"instance" doc:"LLM Service Instance to create or update"`
}

// PostInstanceFromDefinitionRequest is for creating an instance based on a definition
// POST Path: "/v1/llm-instances/{user_handle}/from-definition"
type PostInstanceFromDefinitionRequest struct {
	UserHandle string        `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Body       InstanceInput `json:"instance" doc:"LLM Service Instance to create, based on the specified definition. The instance_handle field is required, other fields will override the values from the definition if provided."`
}

type UploadInstanceResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Owner          string `json:"owner" doc:"User handle of the LLM Service Instance owner"`
		InstanceHandle string `json:"instance_handle" doc:"Handle of created or updated LLM Service Instance"`
		InstanceID     int    `json:"instance_id" doc:"System identifier of created or updated LLM Service Instance"`
	}
}

// Get single LLM Service Instance
// Path: "/v1/llm-instances/{user_handle}/{instance_handle}"

type GetInstanceRequest struct {
	UserHandle     string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	InstanceHandle string `json:"instance_handle" path:"instance_handle" maxLength:"20" minLength:"3" example:"my-gpt-4" doc:"LLM Service Instance handle"`
	Limit          int    `json:"limit,omitempty" query:"limit" minimum:"1" maximum:"200" example:"10" default:"20" doc:"Maximum number of instances to return"`
	Offset         int    `json:"offset,omitempty" query:"offset" minimum:"0" example:"0" default:"0" doc:"Offset into the list of instances"`
}

type GetInstanceResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   InstanceFull  `json:"instance" doc:"LLM Service Instance"`
}

// Get all LLM Service Instances by user (i.e. owner or shared with)
// Path: "/v1/llm-instances/{user_handle}"

type GetUserInstancesRequest struct {
	UserHandle string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Limit      int    `json:"limit,omitempty" query:"limit" minimum:"1" maximum:"200" example:"10" default:"20" doc:"Maximum number of embeddings to return"`
	Offset     int    `json:"offset,omitempty" query:"offset" minimum:"0" example:"0" default:"0" doc:"Offset into the list of embeddings"`
}

type GetUserInstancesResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Instances []InstanceBrief `json:"instances" doc:"List of LLM Service Instances"`
	}
}

// Delete LLM Service Instance
// Path: "/v1/llm-instances/{user_handle}/{instance_handle}"

type DeleteInstanceRequest struct {
	UserHandle     string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	InstanceHandle string `json:"instance_handle" path:"instance_handle" maxLength:"20" minLength:"3" example:"my-gpt-4" doc:"LLM Service Instance handle"`
}

type DeleteInstanceResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
}

// Share Instance with User
// POST Path: "/v1/llm-instances/{user_handle}/{instance_handle}/share"

type ShareInstanceRequest struct {
	UserHandle     string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"alice" doc:"Instance owner handle"`
	InstanceHandle string `json:"instance_handle" path:"instance_handle" maxLength:"20" minLength:"3" example:"my-openai" doc:"Instance handle"`
	Body           struct {
		ShareWithHandle string `json:"share_with_handle" minLength:"3" maxLength:"20" example:"bob" doc:"User handle to share with"`
		Role            string `json:"role" enum:"reader,editor" example:"reader" doc:"Role for shared access"`
	}
}

type ShareInstanceResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Owner          string       `json:"owner" doc:"Instance owner"`
		InstanceHandle string       `json:"instance_handle" doc:"Instance handle"`
		SharedWith     []SharedUser `json:"shared_with" doc:"Users this instance is shared with"`
	}
}

// Unshare Instance from User
// DELETE Path: "/v1/llm-instances/{user_handle}/{instance_handle}/share/{unshare_with_handle}"

type UnshareInstanceRequest struct {
	UserHandle        string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"alice" doc:"Instance owner handle"`
	InstanceHandle    string `json:"instance_handle" path:"instance_handle" maxLength:"20" minLength:"3" example:"my-openai" doc:"Instance handle"`
	UnshareWithHandle string `json:"unshare_with_handle" path:"unshare_with_handle" maxLength:"20" minLength:"3" example:"bob" doc:"User handle to unshare from"`
}

type UnshareInstanceResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
}

// Get users an Instance is shared with
// GET Path: "/v1/llm-instances/{user_handle}/{instance_handle}/shared-with"

type GetInstanceSharedUsersRequest struct {
	UserHandle     string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"alice" doc:"Instance owner handle"`
	InstanceHandle string `json:"instance_handle" path:"instance_handle" maxLength:"20" minLength:"3" example:"my-openai" doc:"Instance handle"`
	Limit          int    `json:"limit,omitempty" query:"limit" minimum:"1" maximum:"200" example:"20" default:"100" doc:"Maximum number of users to return"`
	Offset         int    `json:"offset,omitempty" query:"offset" minimum:"0" example:"0" default:"0" doc:"Offset into the list of users"`
}

type GetInstanceSharedUsersResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Owner          string       `json:"owner" doc:"Instance owner"`
		InstanceHandle string       `json:"instance_handle" doc:"Instance handle"`
		SharedWith     []SharedUser `json:"shared_with" doc:"List of users this instance is shared with"`
	}
}
