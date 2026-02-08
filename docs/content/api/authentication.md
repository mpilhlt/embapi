---
title: "Authentication"
weight: 1
---

# API Authentication

All API requests (except public project read operations) require authentication using API keys passed in the `Authorization` header with a `Bearer` prefix.

## Authentication Method

Include your API key in the `Authorization` header of every request:

```
Authorization: Bearer your_api_key_here
```

## API Key Types

### Admin API Key

- Full access to all API endpoints and resources
- Can create and manage users
- Can access all projects and resources across all users
- Required for administrative operations
- Set via `ADMIN_KEY` environment variable

**Admin-only operations:**
- `POST /v1/users` - Create new users
- `GET /v1/users` - List all users
- `GET /v1/admin/*` - Admin endpoints
- Create/modify `_system` LLM service definitions
- Access any user's resources

### User API Key

- Access limited to user's own resources and shared projects
- Cannot create other users
- Cannot access admin endpoints
- Returned when creating a new user (store securely)

**User capabilities:**
- Manage own projects and LLM services
- Upload and query embeddings in owned projects
- Access projects shared with them (read or edit access)
- Delete own account

## Getting an API Key

### Admin Key

Set the admin key via environment variable when launching the service:

```bash
export ADMIN_KEY="your-secure-admin-key-here"
./embapi
```

### User Key

Create a new user using the admin API key:

```bash
curl -X POST "https://api.example.com/v1/users" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "user_handle": "alice",
    "name": "Alice Doe",
    "email": "alice@example.com"
  }'
```

**Response:**

```json
{
  "user_handle": "alice",
  "name": "Alice Doe",
  "email": "alice@example.com",
  "api_key": "024v2013621509245f2e24"
}
```

**⚠️ Important:** The API key is only returned once during user creation. Store it securely - it cannot be recovered.

## Authorization Header Format

### Correct Format

```bash
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer 024v2013621509245f2e24"
```

### Common Mistakes

❌ Missing "Bearer" prefix:
```
Authorization: 024v2013621509245f2e24
```

❌ Wrong prefix:
```
Authorization: Token 024v2013621509245f2e24
```

❌ Extra quotes:
```
Authorization: Bearer "024v2013621509245f2e24"
```

## Authentication Errors

### 401 Unauthorized

Returned when authentication credentials are missing or invalid.

**Causes:**
- Missing `Authorization` header
- Invalid API key
- Malformed header format

**Example response:**

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

### 403 Forbidden

Returned when authentication succeeds but authorization fails.

**Causes:**
- Attempting to access another user's resources
- User lacking required permissions
- Non-admin accessing admin endpoints

**Example response:**

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to access this resource"
}
```

## Public Access

Projects can be made publicly accessible by setting `public_read: true` when creating or updating the project. Public projects allow unauthenticated read access to:

- Project metadata
- Embeddings
- Similarity search

See [Public Access Documentation](/docs/PUBLIC_ACCESS.md) for details.

## Security Best Practices

1. **Never commit API keys** to version control
2. **Use environment variables** to store API keys in your applications
3. **Rotate keys regularly** by creating new users and deleting old ones
4. **Use HTTPS** in production to prevent key interception
5. **Store admin keys securely** using secrets management systems
6. **Limit admin key distribution** to essential personnel only

## Example Authenticated Request

```bash
# List user's projects
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer 024v2013621509245f2e24"

# Create a new project
curl -X POST "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer 024v2013621509245f2e24" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "my-project",
    "description": "My research project",
    "instance_owner": "alice",
    "instance_handle": "my-openai"
  }'
```

## Related Documentation

- [Error Handling](error-handling/) - Complete error response reference
- [Users Endpoint](endpoints/users/) - User management API
