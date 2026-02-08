package models

import "net/http"

// Project is a project that a user is a member of.
type ProjectFull struct {
	ProjectID          int           `json:"project_id" readOnly:"true" doc:"Unique project identifier"`
	ProjectHandle      string        `json:"project_handle" minLength:"3" maxLength:"20" example:"my-gpt-4" doc:"Project handle"`
	Owner              string        `json:"owner" readOnly:"true" doc:"User handle of the project owner"`
	Description        string        `json:"description,omitempty" maxLength:"255" doc:"Description of the project."`
	MetadataScheme     string        `json:"metadataScheme,omitempty" doc:"Metadata json scheme used in the project."`
	PublicRead         bool          `json:"public_read" doc:"Whether the project is public or not"`
	SharedWith         []SharedUser  `json:"shared_with,omitempty" default:"" doc:"Account names allowed to retrieve information from the project. Defaults to everyone ([\"*\"])"`
	Instance           InstanceBrief `json:"instance,omitempty" doc:"LLM Service Instance used in the project"`
	Role               string        `json:"role,omitempty" doc:"Role of the requesting user in the project (can be owner or some other role)"`
	NumberOfEmbeddings int           `json:"number_of_embeddings" readOnly:"true" doc:"Number of embeddings in the project"`
}

type ProjectBrief struct {
	Owner         string `json:"owner" readOnly:"true" doc:"User handle of the project owner"`
	ProjectHandle string `json:"project_handle" minLength:"3" maxLength:"20" example:"my-gpt-4" doc:"Project handle"`
	ProjectID     int    `json:"project_id" readOnly:"true" doc:"Unique project identifier"`
	PublicRead    bool   `json:"public_read" doc:"Whether the project is public or not"`
	Role          string `json:"role,omitempty" doc:"Role of the requesting user in the project (can be owner or some other role)"`
}

type ProjectSubmission struct {
	ProjectHandle  string       `json:"project_handle" minLength:"3" maxLength:"20" example:"my-gpt-4" doc:"Project handle"`
	Description    string       `json:"description,omitempty" maxLength:"255" doc:"Description of the project."`
	MetadataScheme string       `json:"metadataScheme,omitempty" doc:"Metadata json scheme used in the project."`
	InstanceOwner  string       `json:"instance_owner,omitempty" doc:"User handle of the owner of the LLM Service Instance used in the project."`
	InstanceHandle string       `json:"instance_handle,omitempty" doc:"Handle of the LLM Service Instance used in the project"`
	PublicRead     bool         `json:"public_read,omitempty" default:"false" doc:"Whether the project is public or not"`
	SharedWith     []SharedUser `json:"shared_with,omitempty" doc:"List of users to share the project with upon creation"`
}

// Request and Response structs for the project administration API
// The request structs must be structs with fields for the request path/query/header/cookie parameters and/or body.
// The response structs must be structs with fields for the output headers and body of the operation, if any.

// Put/post project
// PUT Path: "/v1/projects/{user_handle}/{project_handle}"

type PutProjectRequest struct {
	UserHandle    string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	ProjectHandle string `json:"project_handle" path:"project_handle" maxLength:"20" minLength:"3" example:"my-gpt-4" doc:"Project handle"`
	Body          ProjectSubmission
}

// POST Path: "/v1/projects/{user}"

type PostProjectRequest struct {
	UserHandle string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Body       ProjectSubmission
}

type UploadProjectResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   ProjectBrief  `json:"project" doc:"Information about the created or updated project"`
}

// Get all projects by user
// Path: "/v1/projects/{user_handle}"

type GetProjectsRequest struct {
	UserHandle string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	Limit      int    `json:"limit,omitempty" query:"limit" minimum:"1" maximum:"200" example:"10" default:"10" doc:"Maximum number of projects to return"`
	Offset     int    `json:"offset,omitempty" query:"offset" minimum:"0" example:"0" default:"0" doc:"Offset into the list of projects"`
}

type GetProjectsResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		// Handles []string `json:"handles" doc:"Handles of all registered projects for specified user"`
		Projects []ProjectBrief `json:"projects" doc:"Projects that the user is a member of"`
	}
}

// Get single project
// Path: "/v1/projects/{user_handle}/{project_handle}"

type GetProjectRequest struct {
	UserHandle    string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	ProjectHandle string `json:"project_handle" path:"project_handle" maxLength:"20" minLength:"3" example:"my-gpt-4" doc:"Project handle"`
}

type GetProjectResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   ProjectFull   `json:"project" doc:"Project information"`
}

// Delete project
// Path: "/v1/projects/{user_handle}/{project_handle}"

type DeleteProjectRequest struct {
	UserHandle    string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"jdoe" doc:"User handle"`
	ProjectHandle string `json:"project_handle" path:"project_handle" maxLength:"20" minLength:"3" example:"my-gpt-4" doc:"Project handle"`
}

type DeleteProjectResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
}

// Share project with another user
// - POST /v1/projects/{user_handle}/{project_handle}/share

type ShareProjectRequest struct {
	UserHandle    string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"alice" doc:"Project owner handle"`
	ProjectHandle string `json:"project_handle" path:"project_handle" maxLength:"20" minLength:"3" example:"my-openai" doc:"Project handle"`
	Body          struct {
		ShareWithHandle string `json:"share_with_handle" minLength:"3" maxLength:"20" example:"bob" doc:"User handle to share with"`
		Role            string `json:"role" enum:"reader,editor" example:"reader" doc:"Role for shared access"`
	}
}

type ShareProjectResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Owner            string `json:"owner" doc:"Project owner"`
		ProjectHandle    string `json:"project_handle" doc:"Project handle"`
		ProjectID        int    `json:"project_id" doc:"Project ID"`
		SharedWithHandle string `json:"shared_with_handle" doc:"User handle that was granted access"`
		SharedWithRole   string `json:"shared_with_role" doc:"Role granted to the user"`
	}
}

// Unshare Instance from User
// DELETE Path: "/v1/projects/{user_handle}/{project_handle}/share/{unshare_with_handle}"

type UnshareProjectRequest struct {
	UserHandle        string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"alice" doc:"Project owner handle"`
	ProjectHandle     string `json:"project_handle" path:"project_handle" maxLength:"20" minLength:"3" example:"my-project" doc:"Project handle"`
	UnshareWithHandle string `json:"unshare_with_handle" path:"unshare_with_handle" maxLength:"20" minLength:"3" example:"bob" doc:"User handle to unshare from"`
}

type UnshareProjectResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
}

// Get users a Project is shared with
// GET Path: "/v1/projects/{user_handle}/{project_handle}/shared-with"

type GetProjectSharedUsersRequest struct {
	UserHandle    string `json:"user_handle" path:"user_handle" maxLength:"20" minLength:"3" example:"alice" doc:"Project owner handle"`
	ProjectHandle string `json:"project_handle" path:"project_handle" maxLength:"20" minLength:"3" example:"my-project" doc:"Project handle"`
}

type GetProjectSharedUsersResponse struct {
	Header []http.Header `json:"header,omitempty" doc:"Response headers"`
	Body   struct {
		Owner         string       `json:"owner" doc:"Project owner"`
		ProjectHandle string       `json:"project_handle" doc:"Project handle"`
		SharedWith    []SharedUser `json:"shared_with" doc:"List of users this project is shared with"`
	}
}
