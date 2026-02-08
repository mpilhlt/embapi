---
title: "Projects"
weight: 2
---

# Projects Endpoint

Manage vector database projects. Each project contains embeddings and must be associated with an LLM service instance.

## Endpoints

### List User's Projects

Get all projects owned by a user.

**Endpoint:** `GET /v1/projects/{username}`

**Authentication:** Admin, the user themselves, or users with shared access

**Example:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "projects": [
    {
      "project_id": 1,
      "project_handle": "research-docs",
      "owner": "alice",
      "description": "Research document embeddings",
      "instance_id": 5,
      "instance_owner": "alice",
      "instance_handle": "openai-large",
      "public_read": false,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### Create Project

Register a new project for a user.

**Endpoint:** `POST /v1/projects/{username}`

**Authentication:** Admin or the user themselves

**Request Body:**

```json
{
  "project_handle": "research-docs",
  "description": "Research document embeddings",
  "instance_owner": "alice",
  "instance_handle": "openai-large",
  "public_read": false,
  "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"}}}",
  "shared_with": [
    {
      "user_handle": "bob",
      "role": "reader"
    }
  ]
}
```

**Parameters:**

- `project_handle` (string, required): Unique project identifier within the user's namespace
- `description` (string, optional): Project description
- `instance_owner` (string, required): Owner of the LLM service instance
- `instance_handle` (string, required): Handle of the LLM service instance to use
- `public_read` (boolean, optional): Allow unauthenticated read access (default: false)
- `metadataScheme` (string, optional): JSON Schema for validating embedding metadata
- `shared_with` (array, optional): List of users to share the project with

**Example:**

```bash
curl -X POST "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "research-docs",
    "description": "Research document embeddings",
    "instance_owner": "alice",
    "instance_handle": "openai-large"
  }'
```

**Response:**

```json
{
  "project_id": 1,
  "project_handle": "research-docs",
  "owner": "alice",
  "description": "Research document embeddings",
  "instance_id": 5,
  "instance_owner": "alice",
  "instance_handle": "openai-large",
  "public_read": false
}
```

---

### Get Project Information

Retrieve information about a specific project.

**Endpoint:** `GET /v1/projects/{username}/{projectname}`

**Authentication:** Admin, owner, authorized readers, or public if `public_read` is enabled

**Example:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "project_id": 1,
  "project_handle": "research-docs",
  "owner": "alice",
  "description": "Research document embeddings",
  "instance_id": 5,
  "instance_owner": "alice",
  "instance_handle": "openai-large",
  "public_read": false,
  "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"}}}",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Update Project (PUT)

Create or update a project with the specified handle.

**Endpoint:** `PUT /v1/projects/{username}/{projectname}`

**Authentication:** Admin or the owner

**Request Body:** Same as POST endpoint

**Example:**

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "research-docs",
    "description": "Updated description",
    "instance_owner": "alice",
    "instance_handle": "openai-large"
  }'
```

---

### Delete Project

Delete a project and all its embeddings.

**Endpoint:** `DELETE /v1/projects/{username}/{projectname}`

**Authentication:** Admin or the owner

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "message": "Project alice/research-docs deleted successfully"
}
```

**⚠️ Warning:** This operation is irreversible and will delete all embeddings in the project.

---

### Partial Update (PATCH)

Update specific project fields without providing all data.

**Endpoint:** `PATCH /v1/projects/{username}/{projectname}`

**Authentication:** Admin or the owner

**Example - Enable public read access:**

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "public_read": true
  }'
```

**Example - Update description:**

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated project description"
  }'
```

See [PATCH Updates](../patch-updates/) for more details.

---

## Project Sharing

### Share Project

Share a project with another user, granting them read or edit access.

**Endpoint:** `POST /v1/projects/{owner}/{project}/share`

**Authentication:** Admin or the project owner

**Request Body:**

```json
{
  "share_with_handle": "bob",
  "role": "reader"
}
```

**Roles:**
- `reader`: Read-only access to embeddings and similarity search
- `editor`: Read and write access to embeddings

**Example:**

```bash
curl -X POST "https://api.example.com/v1/projects/alice/research-docs/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "bob",
    "role": "reader"
  }'
```

**Response:**

```json
{
  "message": "Project shared successfully",
  "project": "alice/research-docs",
  "shared_with": "bob",
  "role": "reader"
}
```

---

### Unshare Project

Remove a user's access to a shared project.

**Endpoint:** `DELETE /v1/projects/{owner}/{project}/share/{user_handle}`

**Authentication:** Admin or the project owner

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/projects/alice/research-docs/share/bob" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "message": "Project unshared successfully",
  "project": "alice/research-docs",
  "removed_user": "bob"
}
```

---

### List Shared Users

Get a list of users the project is shared with.

**Endpoint:** `GET /v1/projects/{owner}/{project}/shared-with`

**Authentication:** Admin or the project owner

**Example:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice/research-docs/shared-with" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "project": "alice/research-docs",
  "shared_with": [
    {
      "user_handle": "bob",
      "role": "reader"
    },
    {
      "user_handle": "charlie",
      "role": "editor"
    }
  ]
}
```

**Note:** Only the project owner can view this list. Users with shared access cannot see who else has access.

---

## Project Ownership Transfer

### Transfer Ownership

Transfer ownership of a project to another user.

**Endpoint:** `POST /v1/projects/{owner}/{project}/transfer-ownership`

**Authentication:** Admin or the project owner

**Request Body:**

```json
{
  "new_owner_handle": "bob"
}
```

**Example:**

```bash
curl -X POST "https://api.example.com/v1/projects/alice/research-docs/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "new_owner_handle": "bob"
  }'
```

**Response:**

```json
{
  "message": "Project ownership transferred successfully",
  "old_owner": "alice",
  "new_owner": "bob",
  "project_handle": "research-docs",
  "new_path": "/v1/projects/bob/research-docs"
}
```

**Important Notes:**
- Only the current owner can transfer ownership
- The new owner must be an existing user
- The new owner cannot already have a project with the same handle
- After transfer, the old owner loses all access to the project
- All embeddings and project data remain intact

---

## Query Parameters

### List Projects

When listing projects, the following query parameters are available:

- `limit` (integer, default: 10, max: 200): Maximum number of results to return
- `offset` (integer, default: 0): Pagination offset

**Example:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice?limit=20&offset=40" \
  -H "Authorization: Bearer alice_api_key"
```

---

## Project Properties

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_handle` | string | Yes | Unique identifier within user's namespace |
| `owner` | string | Read-only | Project owner's user handle |
| `description` | string | No | Project description |
| `instance_owner` | string | Yes | Owner of the LLM service instance |
| `instance_handle` | string | Yes | Handle of the LLM service instance |
| `instance_id` | integer | Read-only | Internal ID of the LLM service instance |
| `public_read` | boolean | No | Allow unauthenticated read access |
| `metadataScheme` | string | No | JSON Schema for metadata validation |
| `created_at` | timestamp | Read-only | Project creation timestamp |

---

## Metadata Schema Validation

Projects can define a JSON Schema to validate metadata attached to embeddings. See the main README for examples.

**Example Schema:**

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

When uploading embeddings, metadata will be validated against this schema. Invalid metadata will result in a `400 Bad Request` error.

---

## Common Errors

### 400 Bad Request

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Project must have instance_id"
}
```

### 403 Forbidden

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to modify this project"
}
```

### 404 Not Found

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Project 'alice/research-docs' not found"
}
```

### 409 Conflict

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "Project 'alice/research-docs' already exists"
}
```

---

## Related Documentation

- [LLM Services](llm-services/) - Managing LLM service instances
- [Embeddings](embeddings/) - Adding and retrieving embeddings
- [Similars](similars/) - Similarity search
- [Public Access](/docs/PUBLIC_ACCESS.md) - Public project configuration
