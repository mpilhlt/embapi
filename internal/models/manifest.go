package models

import (
	"net/http"
)

// ServiceManifest represents the service metadata and available endpoints
type ServiceManifest struct {
	Name           string                 `json:"name" doc:"Human-readable name of the service"`
	Versions       []string               `json:"versions" doc:"Supported API version numbers"`
	Description    string                 `json:"description,omitempty" doc:"Short human-readable description of the service"`
	Documentation  string                 `json:"documentation,omitempty" doc:"Link to more elaborate human-readable description of the API"`
	Logo           string                 `json:"logo,omitempty" doc:"Link to a square image which can be used as the service's logo"`
	ServiceVersion string                 `json:"serviceVersion,omitempty" doc:"Version of the software exposing this service"`
	Authentication map[string]interface{} `json:"authentication,omitempty" doc:"JSON object describing the authentication offered according to the OpenAPI spec"`
	BatchSize      int                    `json:"batchSize,omitempty" doc:"Batch size for batch operations"`
	Endpoints      []EndpointInfo         `json:"endpoints" doc:"List of available endpoints"`
}

// EndpointInfo represents information about an available endpoint
type EndpointInfo struct {
	Path        string   `json:"path" doc:"URI path or URI template (RFC6570) for the endpoint"`
	Methods     []string `json:"methods" doc:"HTTP methods supported by this endpoint"`
	Description string   `json:"description,omitempty" doc:"Brief description of the endpoint"`
	Tags        []string `json:"tags,omitempty" doc:"Tags categorizing this endpoint"`
}

// GetManifestRequest is the request for getting the service manifest
type GetManifestRequest struct {
}

// GetManifestResponse is the response containing the service manifest
type GetManifestResponse struct {
	Header http.Header     `json:"header,omitempty" doc:"Response headers"`
	Body   ServiceManifest `json:"manifest" doc:"Service manifest"`
}
