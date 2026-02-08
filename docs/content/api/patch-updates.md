---
title: "PATCH Updates"
weight: 8
---

# PATCH Method for Partial Updates

The API supports PATCH requests for partial updates of resources. Instead of providing all resource fields (as required by PUT), you only need to include the fields you want to change.

## Overview

PATCH is automatically available for resources that support both GET and PUT operations. The PATCH endpoint:

1. Retrieves the current resource state via GET
2. Merges your changes with the existing data
3. Applies the update via PUT

This approach simplifies updates by eliminating the need to fetch, modify, and submit complete resource objects.

## Supported Resources

PATCH is available for:

- **Users:** `/v1/users/{username}`
- **Projects:** `/v1/projects/{username}/{projectname}`
- **LLM Services:** `/v1/llm-services/{username}/{llm_servicename}`
- **API Standards:** `/v1/api-standards/{standardname}`

## Request Format

**Endpoint:** `PATCH {resource_url}`

**Content-Type:** `application/json`

**Body:** JSON object containing only the fields to update

## Examples

### Update Project Description

Change only the project description without affecting other fields.

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated project description"
  }'
```

**Response:**

```json
{
  "project_id": 1,
  "project_handle": "research-docs",
  "owner": "alice",
  "description": "Updated project description",
  "instance_id": 5,
  "instance_owner": "alice",
  "instance_handle": "openai-large",
  "public_read": false
}
```

---

### Enable Public Read Access

Make a project publicly accessible without changing other settings.

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "public_read": true
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
  "public_read": true
}
```

---

### Update User Email

Change a user's email address without affecting other user data.

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/users/alice" \
  -H "Authorization: Bearer alice_or_admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice.new@example.com"
  }'
```

**Response:**

```json
{
  "user_handle": "alice",
  "name": "Alice Doe",
  "email": "alice.new@example.com"
}
```

---

### Update LLM Service Description

Change the description of an LLM service instance.

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/llm-services/alice/openai-large" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Production OpenAI embeddings service"
  }'
```

**Response:**

```json
{
  "instance_id": 1,
  "instance_handle": "openai-large",
  "owner": "alice",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "description": "Production OpenAI embeddings service",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072
}
```

---

### Update API Standard Documentation

Update the description of an API standard (admin only).

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/api-standards/openai" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "OpenAI Embeddings API, Version 1 - Updated 2024"
  }'
```

---

### Update Multiple Fields

You can update multiple fields in a single PATCH request.

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "public_read": true
  }'
```

---

### Add Project Metadata Schema

Add or update a project's metadata validation schema.

**Request:**

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"author\"]}"
  }'
```

---

## Use Cases

### Configuration Changes

Update configuration settings without rebuilding entire resource objects:

```bash
# Enable/disable public access
curl -X PATCH ".../projects/alice/research-docs" \
  -d '{"public_read": true}'

# Update instance dimensions
curl -X PATCH ".../llm-services/alice/custom-model" \
  -d '{"dimensions": 1024}'
```

---

### Metadata Management

Update descriptive metadata:

```bash
# Update project description
curl -X PATCH ".../projects/alice/research-docs" \
  -d '{"description": "New description"}'

# Update user name
curl -X PATCH ".../users/alice" \
  -d '{"name": "Alice Smith"}'
```

---

### Schema Evolution

Add or update validation schemas:

```bash
curl -X PATCH ".../projects/alice/research-docs" \
  -d '{
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"category\":{\"type\":\"string\"}}}"
  }'
```

---

## Authentication

PATCH requests require the same authentication as PUT requests for the resource:

| Resource | Who Can PATCH |
|----------|---------------|
| Users | Admin or the user themselves |
| Projects | Admin or project owner |
| LLM Services | Admin or instance owner |
| API Standards | Admin only |

---

## Behavior Details

### Merge Strategy

PATCH uses a **shallow merge** strategy:

- Top-level fields you specify **replace** the existing values
- Nested objects are replaced entirely (not deep-merged)
- Fields you don't specify remain unchanged

**Example:**

Existing project:
```json
{
  "description": "Old description",
  "public_read": false,
  "metadataScheme": "{...old schema...}"
}
```

PATCH request:
```json
{
  "description": "New description"
}
```

Result:
```json
{
  "description": "New description",
  "public_read": false,
  "metadataScheme": "{...old schema...}"
}
```

---

### Validation

All field values are validated according to the resource's schema:

- Field types must be correct
- Required fields (if specified) must be valid
- Constraints (e.g., string length, enum values) are enforced

---

### Atomic Operations

PATCH operations are atomic:
- Either all changes succeed, or none are applied
- If validation fails, the resource remains unchanged

---

## Comparison: PATCH vs PUT

### PUT (Complete Replacement)

**Requires:** All fields (except read-only ones)

```bash
curl -X PUT ".../projects/alice/research-docs" \
  -d '{
    "project_handle": "research-docs",
    "description": "Updated description",
    "instance_owner": "alice",
    "instance_handle": "openai-large",
    "public_read": false
  }'
```

**Use when:** Creating or completely replacing a resource

---

### PATCH (Partial Update)

**Requires:** Only fields to change

```bash
curl -X PATCH ".../projects/alice/research-docs" \
  -d '{
    "description": "Updated description"
  }'
```

**Use when:** Modifying one or a few fields of an existing resource

---

## Error Handling

### 400 Bad Request

Invalid field values or types:

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid value for field 'public_read': expected boolean, got string"
}
```

---

### 401 Unauthorized

Missing or invalid authentication:

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

---

### 403 Forbidden

Insufficient permissions:

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to modify this resource"
}
```

---

### 404 Not Found

Resource doesn't exist:

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Project 'alice/research-docs' not found"
}
```

---

### 422 Unprocessable Entity

Validation failed:

```json
{
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "metadataScheme is not valid JSON Schema"
}
```

---

## Best Practices

### Use PATCH for Single-Field Updates

**Do:**
```bash
curl -X PATCH ".../projects/alice/research-docs" \
  -d '{"public_read": true}'
```

**Don't:**
```bash
# Unnecessarily complex
curl -X PUT ".../projects/alice/research-docs" \
  -d '{
    "project_handle": "research-docs",
    "description": "Research docs",
    "instance_owner": "alice",
    "instance_handle": "openai-large",
    "public_read": true
  }'
```

---

### Group Related Changes

Update multiple related fields in one request:

```bash
curl -X PATCH ".../projects/alice/research-docs" \
  -d '{
    "description": "Updated project",
    "public_read": true,
    "metadataScheme": "{...new schema...}"
  }'
```

---

### Validate Before Patching

When possible, validate changes locally before submitting:

```python
# Python example
def update_project_description(project_path, new_description):
    if not new_description or len(new_description) > 500:
        raise ValueError("Invalid description")
    
    response = requests.patch(
        f"{API_BASE}/projects/{project_path}",
        json={"description": new_description},
        headers={"Authorization": f"Bearer {API_KEY}"}
    )
    return response.json()
```

---

### Handle Errors Gracefully

```python
response = requests.patch(
    f"{API_BASE}/projects/alice/research-docs",
    json={"public_read": True},
    headers={"Authorization": f"Bearer {API_KEY}"}
)

if response.status_code == 200:
    print("Updated successfully")
elif response.status_code == 403:
    print("Permission denied")
elif response.status_code == 404:
    print("Project not found")
else:
    print(f"Error: {response.json()['detail']}")
```

---

## Limitations

### Not Available For

PATCH is **not** available for:
- Endpoints that don't support GET and PUT
- List endpoints (e.g., `GET /v1/projects/alice`)
- Action endpoints (e.g., `/share`, `/transfer-ownership`)

### Cannot Change Identifiers

You cannot use PATCH to change resource identifiers:
- `user_handle`
- `project_handle`
- `instance_handle`
- `api_standard_handle`

To rename a resource, you must create a new resource and delete the old one.

---

## Related Documentation

- [Users](endpoints/users/) - User management endpoints
- [Projects](endpoints/projects/) - Project management endpoints
- [LLM Services](endpoints/llm-services/) - LLM service instance endpoints
- [API Standards](endpoints/api-standards/) - API standard endpoints
- [Error Handling](error-handling/) - Error response reference
