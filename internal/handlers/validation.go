package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mpilhlt/embapi/internal/models"
	"github.com/xeipuuv/gojsonschema"
)

// ValidateEmbeddingDimensions checks if the embeddings vector dimensions match the LLM service dimensions
func ValidateEmbeddingDimensions(embedding models.EmbeddingsInput, llmDimensions int32) error {
	// Check if text_id is not empty
	if embedding.TextID == "" {
		return fmt.Errorf("text_id cannot be empty")
	}

	// Check if vector is not empty
	if len(embedding.Vector) == 0 {
		return fmt.Errorf("vector cannot be empty for text_id '%s'", embedding.TextID)
	}

	// Check if declared vector_dim matches LLM service dimensions
	if embedding.VectorDim != llmDimensions {
		return fmt.Errorf("vector dimension mismatch: embedding declares %d dimensions but LLM service instance '%s' expects %d dimensions", 
			embedding.VectorDim, embedding.InstanceHandle, llmDimensions)
	}

	// Check if actual vector length matches declared vector_dim
	actualLength := int32(len(embedding.Vector))
	if actualLength != embedding.VectorDim {
		return fmt.Errorf("vector length mismatch for text_id '%s': actual vector has %d elements but vector_dim declares %d", 
			embedding.TextID, actualLength, embedding.VectorDim)
	}

	return nil
}

// ValidateMetadataAgainstSchema validates the metadata against a JSON schema if provided
func ValidateMetadataAgainstSchema(metadata json.RawMessage, schemaStr string, isUpdate bool, existingMetadata json.RawMessage) error {
	// If no schema is provided, skip validation
	if schemaStr == "" {
		return nil
	}

	// If no metadata is provided but schema exists
	if len(metadata) == 0 || string(metadata) == "null" {
		// For updates, if we have existing metadata, that's okay - we're not changing it
		if isUpdate && len(existingMetadata) > 0 && string(existingMetadata) != "null" {
			return nil
		}
		// For new records or updates without existing metadata, metadata is required
		return fmt.Errorf("metadata is required when project has a metadata schema defined")
	}

	// Parse the schema
	schemaLoader := gojsonschema.NewStringLoader(schemaStr)
	
	// Parse the metadata
	metadataLoader := gojsonschema.NewBytesLoader(metadata)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, metadataLoader)
	if err != nil {
		return fmt.Errorf("failed to validate metadata against schema: %v", err)
	}

	if !result.Valid() {
		// Build a helpful error message with all validation errors
		errMsg := "metadata validation failed:\n"
		for i, desc := range result.Errors() {
			if i > 0 {
				errMsg += "\n"
			}
			errMsg += fmt.Sprintf("  - %s", desc.String())
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

// ValidateEmbeddingAgainstLLMDimension validates that an embedding's dimensions match the LLM service
func ValidateEmbeddingAgainstLLMDimension(vectorDim int32, llmDimensions int32, textID string) error {
	if vectorDim != llmDimensions {
		return fmt.Errorf("text_id '%s': vector dimension %d does not match LLM service dimension %d", 
			textID, vectorDim, llmDimensions)
	}
	return nil
}

// ValidateEmbeddingMetadataAgainstProjectSchema validates an embedding's metadata against project schema
func ValidateEmbeddingMetadataAgainstProjectSchema(metadata json.RawMessage, schemaStr string, textID string, isUpdate bool, existingMetadata json.RawMessage) error {
	err := ValidateMetadataAgainstSchema(metadata, schemaStr, isUpdate, existingMetadata)
	if err != nil {
		return fmt.Errorf("text_id '%s': %v", textID, err)
	}
	return nil
}

// ValidateNotSystemUser checks if a user handle is "_system" and returns an error if so
// The _system user is read-only and cannot send requests
func ValidateNotSystemUser(userHandle string) error {
	if userHandle == "_system" {
		return huma.Error403Forbidden("_system user cannot send requests - this is a read-only account")
	}
	return nil
}
