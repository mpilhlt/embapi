---
title: "API Standards"
weight: 4
---

# API Standards Endpoint

Manage API standard definitions that specify how to authenticate with different LLM service providers. API standards define the authentication mechanism (Bearer token, API key header, etc.) used by LLM service instances.

## Overview

API standards are referenced by LLM service instances to determine how to authenticate API requests. Examples include:

- **OpenAI**: Bearer token in `Authorization` header
- **Cohere**: API key in `Authorization` header with `Bearer` prefix
- **Google Gemini**: API key as query parameter
- **Ollama**: No authentication required

Pre-seeded standards are available for common providers. See [testdata/valid_api_standard_*.json](https://github.com/mpilhlt/dhamps-vdb/tree/main/testdata) for examples.

---

## Endpoints

### List All API Standards

Get all defined API standards. This endpoint is publicly accessible (no authentication required).

**Endpoint:** `GET /v1/api-standards`

**Authentication:** Public (no authentication required)

**Example:**

```bash
curl -X GET "https://api.example.com/v1/api-standards"
```

**Response:**

```json
{
  "standards": [
    {
      "api_standard_handle": "openai",
      "description": "OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings",
      "key_method": "auth_bearer",
      "key_field": "Authorization"
    },
    {
      "api_standard_handle": "cohere",
      "description": "Cohere Embed API, Version 2, as documented in https://docs.cohere.com/reference/embed",
      "key_method": "auth_bearer",
      "key_field": "Authorization"
    },
    {
      "api_standard_handle": "gemini",
      "description": "Google Gemini Embedding API as documented in https://ai.google.dev/gemini-api/docs/embeddings",
      "key_method": "query_param",
      "key_field": "key"
    }
  ]
}
```

---

### Get API Standard

Retrieve information about a specific API standard. This endpoint is publicly accessible.

**Endpoint:** `GET /v1/api-standards/{standardname}`

**Authentication:** Public (no authentication required)

**Example:**

```bash
curl -X GET "https://api.example.com/v1/api-standards/openai"
```

**Response:**

```json
{
  "api_standard_handle": "openai",
  "description": "OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings",
  "key_method": "auth_bearer",
  "key_field": "Authorization"
}
```

---

### Create API Standard

Register a new API standard definition. Admin-only operation.

**Endpoint:** `POST /v1/api-standards`

**Authentication:** Admin only

**Request Body:**

```json
{
  "api_standard_handle": "custom-provider",
  "description": "Custom provider embedding API",
  "key_method": "auth_bearer",
  "key_field": "Authorization"
}
```

**Parameters:**

- `api_standard_handle` (string, required): Unique identifier for the standard
- `description` (string, required): Description including API documentation URL
- `key_method` (string, required): Authentication method
  - `auth_bearer`: Bearer token in header
  - `auth_apikey`: API key in header
  - `query_param`: API key as query parameter
  - `none`: No authentication
- `key_field` (string, required): Field name for the API key
  - For headers: typically `"Authorization"` or `"X-API-Key"`
  - For query params: parameter name like `"key"` or `"api_key"`

**Example:**

```bash
curl -X POST "https://api.example.com/v1/api-standards" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "api_standard_handle": "ollama",
    "description": "Ollama local embedding API, no authentication required",
    "key_method": "none",
    "key_field": ""
  }'
```

**Response:**

```json
{
  "api_standard_handle": "ollama",
  "description": "Ollama local embedding API, no authentication required",
  "key_method": "none",
  "key_field": ""
}
```

---

### Update API Standard (PUT)

Create or update an API standard with a specific handle. Admin-only operation.

**Endpoint:** `PUT /v1/api-standards/{standardname}`

**Authentication:** Admin only

**Request Body:** Same as POST endpoint

**Example:**

```bash
curl -X PUT "https://api.example.com/v1/api-standards/custom-provider" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "api_standard_handle": "custom-provider",
    "description": "Updated description for custom provider",
    "key_method": "auth_bearer",
    "key_field": "Authorization"
  }'
```

---

### Delete API Standard

Delete an API standard definition. Admin-only operation.

**Endpoint:** `DELETE /v1/api-standards/{standardname}`

**Authentication:** Admin only

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/api-standards/custom-provider" \
  -H "Authorization: Bearer admin_api_key"
```

**Response:**

```json
{
  "message": "API standard 'custom-provider' deleted successfully"
}
```

**⚠️ Warning:** Cannot delete API standards that are referenced by LLM service instances.

---

### Partial Update (PATCH)

Update specific API standard fields without providing all data. Admin-only operation.

**Endpoint:** `PATCH /v1/api-standards/{standardname}`

**Authentication:** Admin only

**Example - Update description:**

```bash
curl -X PATCH "https://api.example.com/v1/api-standards/openai" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "OpenAI Embeddings API v1 - Updated documentation link"
  }'
```

See [PATCH Updates](../patch-updates/) for more details.

---

## API Standard Properties

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_standard_handle` | string | Yes | Unique identifier (e.g., "openai", "cohere") |
| `description` | string | Yes | Description with API documentation URL |
| `key_method` | string | Yes | Authentication method |
| `key_field` | string | Yes | Field name for API key/token |

---

## Authentication Methods

### auth_bearer

Bearer token authentication in the `Authorization` header.

**Example:**
```json
{
  "key_method": "auth_bearer",
  "key_field": "Authorization"
}
```

**HTTP Request:**
```
Authorization: Bearer sk-proj-abc123...
```

**Used by:** OpenAI, Cohere, Anthropic

---

### auth_apikey

API key in a custom header field.

**Example:**
```json
{
  "key_method": "auth_apikey",
  "key_field": "X-API-Key"
}
```

**HTTP Request:**
```
X-API-Key: abc123...
```

**Used by:** Some custom API providers

---

### query_param

API key passed as a URL query parameter.

**Example:**
```json
{
  "key_method": "query_param",
  "key_field": "key"
}
```

**HTTP Request:**
```
GET https://api.example.com/embed?key=abc123...
```

**Used by:** Google Gemini, some older APIs

---

### none

No authentication required.

**Example:**
```json
{
  "key_method": "none",
  "key_field": ""
}
```

**Used by:** Ollama (local), self-hosted models without auth

---

## Pre-seeded API Standards

The following API standards are created during database migration:

### openai

```json
{
  "api_standard_handle": "openai",
  "description": "OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings",
  "key_method": "auth_bearer",
  "key_field": "Authorization"
}
```

### cohere

```json
{
  "api_standard_handle": "cohere",
  "description": "Cohere Embed API, Version 2, as documented in https://docs.cohere.com/reference/embed",
  "key_method": "auth_bearer",
  "key_field": "Authorization"
}
```

### gemini

```json
{
  "api_standard_handle": "gemini",
  "description": "Google Gemini Embedding API as documented in https://ai.google.dev/gemini-api/docs/embeddings",
  "key_method": "query_param",
  "key_field": "key"
}
```

---

## Use Cases

### Creating a Custom API Standard

For self-hosted or custom LLM services:

```bash
curl -X POST "https://api.example.com/v1/api-standards" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "api_standard_handle": "vllm-local",
    "description": "vLLM local deployment with custom auth",
    "key_method": "auth_apikey",
    "key_field": "X-API-Key"
  }'
```

### Referencing in LLM Service Instance

Once created, reference the API standard in your LLM service instance:

```bash
curl -X PUT "https://api.example.com/v1/llm-services/alice/my-vllm" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "my-vllm",
    "endpoint": "https://vllm.local/v1/embeddings",
    "api_standard": "vllm-local",
    "model": "custom-embed",
    "dimensions": 768,
    "api_key_encrypted": "my-secret-key"
  }'
```

---

## Common Errors

### 400 Bad Request

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid key_method: must be one of auth_bearer, auth_apikey, query_param, none"
}
```

### 401 Unauthorized (Admin Operations)

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

### 403 Forbidden (Admin Operations)

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "Only admin users can create/modify/delete API standards"
}
```

### 404 Not Found

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "API standard 'custom-provider' not found"
}
```

### 409 Conflict

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "Cannot delete API standard: referenced by 5 LLM service instances"
}
```

---

## Related Documentation

- [LLM Services](llm-services/) - LLM service instances reference API standards
- [Authentication](../authentication/) - User authentication methods
- [Testdata Examples](https://github.com/mpilhlt/dhamps-vdb/tree/main/testdata) - Example API standard definitions
