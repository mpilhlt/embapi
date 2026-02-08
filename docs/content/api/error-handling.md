---
title: "Error Handling"
weight: 9
---

# Error Handling

The API uses standard HTTP status codes and returns structured error responses in JSON format for all error conditions.

## Error Response Format

All error responses follow this structure:

```json
{
  "title": "Error Title",
  "status": 400,
  "detail": "Detailed error message explaining what went wrong"
}
```

**Fields:**

- `title` (string): Human-readable error title
- `status` (integer): HTTP status code
- `detail` (string): Detailed description of the error

## HTTP Status Codes

### 2xx Success

#### 200 OK

Request succeeded. Response body contains requested data.

**Example:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "projects": [...]
}
```

---

#### 201 Created

Resource created successfully. Response body contains the new resource.

**Example:**

```bash
curl -X POST "https://api.example.com/v1/users" \
  -H "Authorization: Bearer admin_api_key" \
  -d '{"user_handle": "bob"}'
```

**Response:**

```json
{
  "user_handle": "bob",
  "api_key": "..."
}
```

---

### 4xx Client Errors

#### 400 Bad Request

The request is invalid or contains malformed data.

**Common causes:**
- Invalid JSON syntax
- Missing required fields
- Invalid field values or types
- Dimension mismatches in embeddings
- Metadata schema violations
- Invalid query parameter values

**Examples:**

**Invalid JSON:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid JSON: unexpected end of input"
}
```

**Missing required field:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Missing required field: project_handle"
}
```

**Dimension mismatch:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch: embedding declares 3072 dimensions but LLM service 'openai-large' expects 1536 dimensions"
}
```

**Metadata validation:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc123': metadata validation failed:\n  - author: author is required"
}
```

**Invalid query parameter:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "limit must be between 1 and 200"
}
```

---

#### 401 Unauthorized

Authentication failed or credentials are missing.

**Common causes:**
- Missing `Authorization` header
- Invalid API key
- Expired API key
- Malformed `Authorization` header

**Example:**

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

**Troubleshooting:**

- Verify the `Authorization` header is present
- Check the API key is correct
- Ensure the header format is `Authorization: Bearer <api_key>`
- Verify the user still exists

---

#### 403 Forbidden

Authentication succeeded but authorization failed. The authenticated user lacks permission for the requested operation.

**Common causes:**
- User attempting to access another user's private resources
- Non-admin attempting admin-only operations
- User attempting to modify shared resources they don't own
- Attempting to access resources with insufficient role (reader vs editor)

**Examples:**

**Admin-only operation:**

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "Only admin users can create new users"
}
```

**Accessing another user's resource:**

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to access this project"
}
```

**Insufficient role:**

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to modify embeddings in this project. Editor role required."
}
```

---

#### 404 Not Found

The requested resource does not exist.

**Common causes:**
- Resource was deleted
- Incorrect resource identifier
- Typo in URL path
- Resource never existed

**Examples:**

**User not found:**

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "User 'alice' not found"
}
```

**Project not found:**

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Project 'alice/research-docs' not found"
}
```

**Embedding not found:**

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Embedding 'doc123' not found in project 'alice/research-docs'"
}
```

**LLM service not found:**

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "LLM service instance 'alice/openai-large' not found"
}
```

---

#### 409 Conflict

The request conflicts with the current state of the resource.

**Common causes:**
- Creating a resource that already exists
- Deleting a resource that is in use
- Concurrent modification conflicts

**Examples:**

**Resource already exists:**

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "User 'alice' already exists"
}
```

**Resource in use:**

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "Cannot delete LLM service instance: in use by 3 projects"
}
```

**Ownership transfer conflict:**

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "New owner already has a project with handle 'research-docs'"
}
```

---

#### 422 Unprocessable Entity

The request is syntactically correct but semantically invalid.

**Common causes:**
- Invalid JSON Schema
- Logical validation failures
- Constraint violations

**Example:**

```json
{
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "metadataScheme is not valid JSON Schema: invalid schema structure"
}
```

---

### 5xx Server Errors

#### 500 Internal Server Error

An unexpected error occurred on the server.

**Common causes:**
- Database connection failures
- Unexpected exceptions
- Configuration errors

**Example:**

```json
{
  "title": "Internal Server Error",
  "status": 500,
  "detail": "An internal error occurred. Please try again later."
}
```

**Action:** Contact support if the error persists.

---

#### 503 Service Unavailable

The service is temporarily unavailable.

**Common causes:**
- Database maintenance
- Service overload
- Network issues

**Example:**

```json
{
  "title": "Service Unavailable",
  "status": 503,
  "detail": "Service temporarily unavailable. Please try again later."
}
```

**Action:** Retry the request after a delay.

---

## Common Error Scenarios

### Authentication Errors

#### Missing Authorization Header

**Request:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice"
```

**Response:**

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

**Solution:** Include the `Authorization` header:

```bash
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer alice_api_key"
```

---

#### Invalid API Key

**Request:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer invalid_key"
```

**Response:**

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

**Solution:** Verify your API key is correct.

---

### Permission Errors

#### Non-Admin Creating Users

**Request:**

```bash
curl -X POST "https://api.example.com/v1/users" \
  -H "Authorization: Bearer alice_api_key" \
  -d '{"user_handle": "bob"}'
```

**Response:**

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "Only admin users can create new users"
}
```

**Solution:** Use an admin API key.

---

#### Accessing Another User's Project

**Request:**

```bash
curl -X GET "https://api.example.com/v1/projects/bob/private-project" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to access this project"
}
```

**Solution:** Request the project owner to share the project with you.

---

### Validation Errors

#### Invalid JSON

**Request:**

```bash
curl -X POST "https://api.example.com/v1/users" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{"user_handle": "bob"'
```

**Response:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid JSON: unexpected end of input"
}
```

**Solution:** Fix the JSON syntax.

---

#### Dimension Mismatch

**Request:**

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/research-docs" \
  -d '{
    "embeddings": [{
      "text_id": "doc123",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3],
      "vector_dim": 3
    }]
  }'
```

**Response:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch: embedding declares 3 dimensions but LLM service 'openai-large' expects 3072 dimensions"
}
```

**Solution:** Ensure vector dimensions match the LLM service configuration.

---

#### Metadata Schema Violation

**Request:**

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/research-docs" \
  -d '{
    "embeddings": [{
      "text_id": "doc123",
      "instance_handle": "openai-large",
      "vector": [...],
      "vector_dim": 3072,
      "metadata": {
        "year": 2024
      }
    }]
  }'
```

**Response (if project schema requires "author"):**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc123': metadata validation failed:\n  - author: author is required"
}
```

**Solution:** Include all required metadata fields.

---

### Resource Conflicts

#### Creating Duplicate Resource

**Request:**

```bash
curl -X POST "https://api.example.com/v1/users" \
  -H "Authorization: Bearer admin_api_key" \
  -d '{"user_handle": "alice"}'
```

**Response:**

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "User 'alice' already exists"
}
```

**Solution:** Use a different user handle or update the existing user.

---

#### Deleting Resource In Use

**Request:**

```bash
curl -X DELETE "https://api.example.com/v1/llm-services/alice/openai-large" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "Cannot delete LLM service instance: in use by 3 projects"
}
```

**Solution:** Delete or update the projects using this instance first.

---

## Error Handling Best Practices

### Client-Side Error Handling

**Python example:**

```python
import requests

response = requests.get(
    "https://api.example.com/v1/projects/alice",
    headers={"Authorization": f"Bearer {api_key}"}
)

if response.status_code == 200:
    projects = response.json()["projects"]
    print(f"Found {len(projects)} projects")
    
elif response.status_code == 401:
    print("Authentication failed - check API key")
    
elif response.status_code == 403:
    print("Permission denied")
    
elif response.status_code == 404:
    print("User not found")
    
elif response.status_code >= 500:
    print("Server error - retry later")
    
else:
    error = response.json()
    print(f"Error: {error['detail']}")
```

---

### Retry Logic

For transient errors (5xx), implement exponential backoff:

```python
import time
import requests

def api_request_with_retry(url, max_retries=3):
    for attempt in range(max_retries):
        response = requests.get(url, headers={...})
        
        if response.status_code < 500:
            return response
            
        if attempt < max_retries - 1:
            wait_time = 2 ** attempt  # Exponential backoff
            print(f"Retrying in {wait_time}s...")
            time.sleep(wait_time)
    
    return response
```

---

### Validation Before Request

Validate data locally before sending to reduce 400 errors:

```python
def create_embedding(text_id, vector, metadata):
    # Validate locally
    if not text_id:
        raise ValueError("text_id is required")
    
    if not isinstance(vector, list):
        raise ValueError("vector must be a list")
    
    if len(vector) != 3072:
        raise ValueError("vector must have 3072 dimensions")
    
    # Send request
    response = requests.post(
        "https://api.example.com/v1/embeddings/alice/research-docs",
        json={
            "embeddings": [{
                "text_id": text_id,
                "instance_handle": "openai-large",
                "vector": vector,
                "vector_dim": len(vector),
                "metadata": metadata
            }]
        },
        headers={"Authorization": f"Bearer {api_key}"}
    )
    
    return response.json()
```

---

## Troubleshooting

### Check API Documentation

Always refer to the live API documentation:

- **OpenAPI YAML:** `/openapi.yaml`
- **Interactive Docs:** `/docs`

---

### Verify Request Format

Use tools like `curl -v` to inspect the full request:

```bash
curl -v -X POST "https://api.example.com/v1/users" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{"user_handle": "bob"}'
```

---

### Check Status and Logs

For persistent 5xx errors, check service status and logs (if you have access).

---

### Contact Support

If errors persist or the cause is unclear:

1. Note the error message and status code
2. Record the request details (method, URL, headers, body)
3. Check if the issue is reproducible
4. Contact support with this information

---

## Related Documentation

- [Authentication](authentication/) - API authentication guide
- [Query Parameters](query-parameters/) - Valid parameter values
- [Endpoints](endpoints/) - Complete endpoint reference
