---
title: "Embeddings"
weight: 5
---

# Embeddings Endpoint

Store and retrieve vector embeddings with associated text identifiers and metadata. Embeddings are organized within projects and validated against the project's LLM service instance dimensions.

## Endpoints

### List Embeddings

Get all embeddings for a project with pagination support.

**Endpoint:** `GET /v1/embeddings/{username}/{projectname}`

**Authentication:** Admin, owner, authorized readers, or public if `public_read` is enabled

**Query Parameters:**

- `limit` (integer, default: 10, max: 200): Maximum number of results to return
- `offset` (integer, default: 0): Pagination offset

**Example:**

```bash
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs?limit=20&offset=0" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "embeddings": [
    {
      "text_id": "doc123",
      "user_handle": "alice",
      "project_handle": "research-docs",
      "instance_handle": "openai-large",
      "text": "This is a research document about machine learning.",
      "vector": [-0.020850, 0.018522, 0.053270, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "Alice Doe",
        "year": 2024
      }
    }
  ],
  "total_count": 150,
  "limit": 20,
  "offset": 0
}
```

---

### Upload Embeddings (Batch)

Upload one or more embeddings to a project. Supports batch upload for efficiency.

**Endpoint:** `POST /v1/embeddings/{username}/{projectname}`

**Authentication:** Admin, owner, or authorized editors

**Request Body:**

```json
{
  "embeddings": [
    {
      "text_id": "doc123",
      "instance_handle": "openai-large",
      "text": "This is a research document.",
      "vector": [-0.020850, 0.018522, 0.053270, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "Alice Doe",
        "year": 2024
      }
    },
    {
      "text_id": "doc124",
      "instance_handle": "openai-large",
      "text": "Another research document.",
      "vector": [0.012345, -0.054321, 0.098765, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "Bob Smith",
        "year": 2024
      }
    }
  ]
}
```

**Parameters:**

- `embeddings` (array, required): Array of embedding objects
  - `text_id` (string, required): Unique identifier for the text (URL-encoded recommended)
  - `instance_handle` (string, required): Handle of the LLM service instance used
  - `vector` (array of floats, required): Embedding vector
  - `vector_dim` (integer, required): Number of dimensions in the vector
  - `text` (string, optional): The original text
  - `metadata` (object, optional): Additional metadata (validated against project schema if defined)

**Example:**

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [
      {
        "text_id": "doc123",
        "instance_handle": "openai-large",
        "text": "Machine learning research document.",
        "vector": [-0.020850, 0.018522, 0.053270],
        "vector_dim": 3,
        "metadata": {
          "author": "Alice Doe",
          "year": 2024
        }
      }
    ]
  }'
```

**Response:**

```json
{
  "message": "1 embedding(s) uploaded successfully",
  "uploaded": ["doc123"]
}
```

---

### Get Single Embedding

Retrieve information about a specific embedding by its text identifier.

**Endpoint:** `GET /v1/embeddings/{username}/{projectname}/{identifier}`

**Authentication:** Admin, owner, authorized readers, or public if `public_read` is enabled

**Example:**

```bash
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs/doc123" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "text_id": "doc123",
  "user_handle": "alice",
  "project_handle": "research-docs",
  "project_id": 1,
  "instance_handle": "openai-large",
  "text": "Machine learning research document.",
  "vector": [-0.020850, 0.018522, 0.053270, ...],
  "vector_dim": 3072,
  "metadata": {
    "author": "Alice Doe",
    "year": 2024
  }
}
```

**Note:** URL-encode the identifier if it contains special characters (e.g., URLs).

**Example with URL identifier:**

```bash
# URL-encoded identifier
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs/https%3A%2F%2Fexample.com%2Fdoc123" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Delete Single Embedding

Delete a specific embedding by its text identifier.

**Endpoint:** `DELETE /v1/embeddings/{username}/{projectname}/{identifier}`

**Authentication:** Admin, owner, or authorized editors

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/embeddings/alice/research-docs/doc123" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "message": "Embedding 'doc123' deleted successfully"
}
```

---

### Delete All Embeddings

Delete all embeddings in a project.

**Endpoint:** `DELETE /v1/embeddings/{username}/{projectname}`

**Authentication:** Admin or owner

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/embeddings/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "message": "All embeddings deleted from project alice/research-docs",
  "deleted_count": 150
}
```

**⚠️ Warning:** This operation is irreversible.

---

## Request Format Details

### Text Identifiers

Text identifiers (`text_id`) should be:
- **Unique** within a project
- **URL-encoded** if they contain special characters
- **Descriptive** for easier retrieval

**Examples:**
- Simple: `"doc123"`, `"article-456"`
- URL-based: `"https://example.com/docs/123"` (URL-encode in requests)
- Path-based: `"books/chapter1/section2"`

### Vector Format

Vectors must be:
- **Arrays of floats** (float32 or float64)
- **Matching dimensions** specified in `vector_dim`
- **Consistent dimensions** with the project's LLM service instance

**Example:**

```json
{
  "vector": [-0.020850, 0.018522, 0.053270, 0.071384, 0.020003],
  "vector_dim": 5
}
```

### Metadata Format

Metadata is a flexible JSON object that can contain any valid JSON data:

**Simple metadata:**

```json
{
  "metadata": {
    "author": "Alice Doe",
    "year": 2024,
    "category": "research"
  }
}
```

**Nested metadata:**

```json
{
  "metadata": {
    "author": {
      "name": "Alice Doe",
      "id": "author-123"
    },
    "publication": {
      "year": 2024,
      "title": "Machine Learning Research"
    },
    "tags": ["AI", "ML", "embeddings"]
  }
}
```

---

## Validation

### Dimension Validation

The API automatically validates that:

1. **Vector dimension consistency**: The `vector_dim` field must match the dimensions configured in the LLM service instance
2. **Vector length verification**: The actual number of elements in the `vector` array must match the declared `vector_dim`

**Error example:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch: embedding declares 3072 dimensions but LLM service 'openai-large' expects 1536 dimensions"
}
```

### Metadata Schema Validation

If the project has a `metadataScheme` defined, all uploaded embeddings' metadata will be validated against it.

**Example project schema:**

```json
{
  "type": "object",
  "properties": {
    "author": {"type": "string"},
    "year": {"type": "integer"}
  },
  "required": ["author"]
}
```

**Valid metadata:**

```json
{
  "author": "Alice Doe",
  "year": 2024
}
```

**Invalid metadata (missing required field):**

```json
{
  "year": 2024
}
```

**Error response:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc123': metadata validation failed:\n  - author: author is required"
}
```

---

## Response Format

### List Response

```json
{
  "embeddings": [...],
  "total_count": 150,
  "limit": 20,
  "offset": 0
}
```

### Single Embedding Response

```json
{
  "text_id": "doc123",
  "user_handle": "alice",
  "project_handle": "research-docs",
  "project_id": 1,
  "instance_handle": "openai-large",
  "text": "...",
  "vector": [...],
  "vector_dim": 3072,
  "metadata": {...}
}
```

### Upload Response

```json
{
  "message": "3 embedding(s) uploaded successfully",
  "uploaded": ["doc123", "doc124", "doc125"]
}
```

---

## Common Errors

### 400 Bad Request - Dimension Mismatch

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch: embedding declares 3072 dimensions but LLM service 'openai-large' expects 1536 dimensions"
}
```

### 400 Bad Request - Invalid Vector

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "vector length (1536) does not match declared vector_dim (3072)"
}
```

### 400 Bad Request - Metadata Schema Violation

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc123': metadata validation failed:\n  - author: author is required"
}
```

### 403 Forbidden

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to upload embeddings to this project"
}
```

### 404 Not Found - Project

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Project 'alice/research-docs' not found"
}
```

### 404 Not Found - Embedding

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Embedding 'doc123' not found in project 'alice/research-docs'"
}
```

### 409 Conflict

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "Embedding 'doc123' already exists in project 'alice/research-docs'"
}
```

---

## Best Practices

### Batch Uploads

Upload multiple embeddings in a single request for better performance:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [
      {...},
      {...},
      {...}
    ]
  }'
```

### URL-Encoded Identifiers

Use URL encoding for identifiers with special characters:

```python
import urllib.parse

text_id = "https://example.com/docs/123"
encoded_id = urllib.parse.quote(text_id, safe='')
# Result: "https%3A%2F%2Fexample.com%2Fdocs%2F123"
```

### Metadata Design

Design metadata schemas that:
- Include commonly queried fields
- Use consistent data types
- Validate required fields
- Support your similarity search filtering needs

---

## Related Documentation

- [Projects](projects/) - Project management and configuration
- [Similars](similars/) - Similarity search using embeddings
- [Query Parameters](../query-parameters/) - Pagination and filtering options
- [Error Handling](../error-handling/) - Complete error reference
