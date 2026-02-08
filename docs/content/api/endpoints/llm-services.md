---
title: "LLM Services"
weight: 3
---

# LLM Services Endpoint

Manage LLM service instances and definitions. The system supports two types of LLM service resources:

1. **Definitions**: Reusable templates owned by `_system` or individual users
2. **Instances**: User-specific configurations with API keys that reference definitions or stand alone

## Architecture Overview

```
definitions (templates)
  └── owned by _system or users
  └── contain: endpoint, model, dimensions, api_standard
  └── no API keys

instances (user-specific)
  └── owned by individual users
  └── reference a definition (optional)
  └── contain encrypted API keys
  └── can be shared with other users
```

For complete details, see [LLM Service Refactoring Documentation](/docs/LLM_SERVICE_REFACTORING.md).

---

## Instance Endpoints

### List User's Instances

Get all LLM service instances owned by or shared with a user.

**Endpoint:** `GET /v1/llm-services/{username}`

**Authentication:** Admin or the user themselves

**Example:**

```bash
curl -X GET "https://api.example.com/v1/llm-services/alice" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "instances": [
    {
      "instance_id": 1,
      "instance_handle": "openai-large",
      "owner": "alice",
      "endpoint": "https://api.openai.com/v1/embeddings",
      "description": "OpenAI large embeddings",
      "api_standard": "openai",
      "model": "text-embedding-3-large",
      "dimensions": 3072,
      "definition_id": 1
    },
    {
      "instance_id": 2,
      "instance_handle": "custom-model",
      "owner": "alice",
      "endpoint": "https://custom.api.example.com/embed",
      "api_standard": "openai",
      "model": "custom-embed-v1",
      "dimensions": 1536
    }
  ]
}
```

**Note:** API keys are never returned in GET responses for security.

---

### Create or Update Instance (PUT)

Create a new LLM service instance or update an existing one.

**Endpoint:** `PUT /v1/llm-services/{username}/{instance_handle}`

**Authentication:** Admin or the user themselves

**Request Body (Standalone Instance):**

```json
{
  "instance_handle": "openai-large",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "description": "OpenAI large embeddings",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072,
  "api_key_encrypted": "sk-proj-..."
}
```

**Request Body (From Definition):**

```json
{
  "instance_handle": "my-openai",
  "definition_owner": "_system",
  "definition_handle": "openai-large",
  "api_key_encrypted": "sk-proj-..."
}
```

**Parameters:**

- `instance_handle` (string, required): Unique identifier within user's namespace
- `endpoint` (string, required for standalone): API endpoint URL
- `description` (string, optional): Instance description
- `api_standard` (string, required for standalone): Reference to API standard (e.g., "openai", "cohere")
- `model` (string, required for standalone): Model name
- `dimensions` (integer, required for standalone): Vector dimensions
- `api_key_encrypted` (string, optional): API key (encrypted if ENCRYPTION_KEY is set)
- `definition_owner` (string, optional): Owner of the definition template
- `definition_handle` (string, optional): Handle of the definition template

**Example - Standalone:**

```bash
curl -X PUT "https://api.example.com/v1/llm-services/alice/openai-large" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "openai-large",
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "api_key_encrypted": "sk-proj-..."
  }'
```

**Example - From _system Definition:**

```bash
curl -X PUT "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "my-openai",
    "definition_owner": "_system",
    "definition_handle": "openai-large",
    "api_key_encrypted": "sk-proj-..."
  }'
```

**Response:**

```json
{
  "instance_id": 1,
  "instance_handle": "openai-large",
  "owner": "alice",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072
}
```

**Security Note:** API keys are encrypted using AES-256-GCM if the `ENCRYPTION_KEY` environment variable is set. Keys are never returned in responses.

---

### Get Instance Information

Retrieve information about a specific LLM service instance.

**Endpoint:** `GET /v1/llm-services/{username}/{instance_handle}`

**Authentication:** Admin, owner, or users with shared access

**Example:**

```bash
curl -X GET "https://api.example.com/v1/llm-services/alice/openai-large" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "instance_id": 1,
  "instance_handle": "openai-large",
  "owner": "alice",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "description": "OpenAI large embeddings",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072,
  "definition_id": 1
}
```

---

### Delete Instance

Delete an LLM service instance.

**Endpoint:** `DELETE /v1/llm-services/{username}/{instance_handle}`

**Authentication:** Admin or the owner

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/llm-services/alice/openai-large" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "message": "LLM service instance alice/openai-large deleted successfully"
}
```

**⚠️ Warning:** Cannot delete instances that are in use by projects.

---

### Partial Update (PATCH)

Update specific instance fields without providing all data.

**Endpoint:** `PATCH /v1/llm-services/{username}/{instance_handle}`

**Authentication:** Admin or the owner

**Example - Update description:**

```bash
curl -X PATCH "https://api.example.com/v1/llm-services/alice/openai-large" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description"
  }'
```

See [PATCH Updates](../patch-updates/) for more details.

---

## Instance Sharing

### Share Instance

Share an LLM service instance with another user.

**Endpoint:** `POST /v1/llm-services/{owner}/{instance}/share`

**Authentication:** Admin or the instance owner

**Request Body:**

```json
{
  "share_with_handle": "bob",
  "role": "reader"
}
```

**Roles:**
- `reader`: Can use the instance but cannot see API keys
- `editor`: Can use the instance (owner can still modify)
- `owner`: Full control (cannot be granted via sharing)

**Example:**

```bash
curl -X POST "https://api.example.com/v1/llm-services/alice/openai-large/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "bob",
    "role": "reader"
  }'
```

**Important:** Shared users can USE the instance but cannot see the API key.

---

### Unshare Instance

Remove a user's access to a shared instance.

**Endpoint:** `DELETE /v1/llm-services/{owner}/{instance}/share/{user_handle}`

**Authentication:** Admin or the instance owner

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/llm-services/alice/openai-large/share/bob" \
  -H "Authorization: Bearer alice_api_key"
```

---

### List Shared Users

Get a list of users the instance is shared with.

**Endpoint:** `GET /v1/llm-services/{owner}/{instance}/shared-with`

**Authentication:** Admin or the instance owner

**Example:**

```bash
curl -X GET "https://api.example.com/v1/llm-services/alice/openai-large/shared-with" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "instance": "alice/openai-large",
  "shared_with": [
    {
      "user_handle": "bob",
      "role": "reader"
    }
  ]
}
```

---

## Definition Endpoints

### List System Definitions

Get all LLM service definitions owned by `_system`.

**Endpoint:** `GET /v1/llm-service-definitions/_system`

**Authentication:** Any authenticated user

**Example:**

```bash
curl -X GET "https://api.example.com/v1/llm-service-definitions/_system" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "definitions": [
    {
      "definition_id": 1,
      "definition_handle": "openai-large",
      "owner": "_system",
      "endpoint": "https://api.openai.com/v1/embeddings",
      "description": "OpenAI text-embedding-3-large",
      "api_standard": "openai",
      "model": "text-embedding-3-large",
      "dimensions": 3072
    },
    {
      "definition_id": 2,
      "definition_handle": "openai-small",
      "owner": "_system",
      "endpoint": "https://api.openai.com/v1/embeddings",
      "description": "OpenAI text-embedding-3-small",
      "api_standard": "openai",
      "model": "text-embedding-3-small",
      "dimensions": 1536
    }
  ]
}
```

**Pre-seeded System Definitions:**
- `openai-large`: OpenAI text-embedding-3-large (3072 dimensions)
- `openai-small`: OpenAI text-embedding-3-small (1536 dimensions)
- `cohere-v4`: Cohere embed-english-v4.0 (1536 dimensions)
- `gemini-embedding-001`: Google Gemini embedding-001 (768 dimensions)

---

### Create User Definition

Create a reusable LLM service definition template.

**Endpoint:** `POST /v1/llm-service-definitions/{username}`

**Authentication:** Admin or the user themselves

**Request Body:**

```json
{
  "definition_handle": "custom-model",
  "endpoint": "https://custom.api.example.com/embed",
  "description": "Custom embedding model",
  "api_standard": "openai",
  "model": "custom-embed-v1",
  "dimensions": 1024
}
```

**Note:** Only admin users can create `_system` definitions.

---

## Instance Properties

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `instance_handle` | string | Yes | Unique identifier within user's namespace |
| `owner` | string | Read-only | Instance owner's user handle |
| `endpoint` | string | Yes* | API endpoint URL |
| `description` | string | No | Instance description |
| `api_standard` | string | Yes* | Reference to API standard |
| `model` | string | Yes* | Model name |
| `dimensions` | integer | Yes* | Vector dimensions |
| `api_key_encrypted` | string | Write-only | API key (never returned) |
| `definition_id` | integer | Read-only | Reference to definition template |

\* Required for standalone instances; inherited from definition if using template

---

## Security Features

### API Key Encryption

- **Algorithm:** AES-256-GCM
- **Key Source:** `ENCRYPTION_KEY` environment variable
- **Storage:** Encrypted in `api_key_encrypted` column
- **Retrieval:** Never returned in API responses

### API Key Protection

API keys are write-only:
- Provided during instance creation/update
- Encrypted before storage
- Never returned in GET/list responses
- Shared users cannot see API keys

### Shared Instance Access

When an instance is shared:
- Shared users can USE the instance for projects
- Shared users CANNOT see the API key
- Shared users CANNOT modify the instance
- Only owner can manage sharing

---

## Common Errors

### 400 Bad Request

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Instance must specify either all configuration fields or reference a definition"
}
```

### 403 Forbidden

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "Only admin users can create _system definitions"
}
```

### 404 Not Found

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "LLM service instance 'alice/openai-large' not found"
}
```

### 409 Conflict

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "Cannot delete instance: in use by 3 projects"
}
```

---

## Related Documentation

- [API Standards](api-standards/) - Managing API standard definitions
- [Projects](projects/) - Projects require an LLM service instance
- [LLM Service Refactoring](/docs/LLM_SERVICE_REFACTORING.md) - Complete architecture documentation
