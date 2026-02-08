# Public Access to Embeddings

## Overview

Projects can be configured to allow unauthenticated (public) read access to embeddings and similar documents by setting the `public_read` field to `true` when creating or updating a project.

**Note**: The `shared_with` field is used for sharing projects with specific users. For public access, use the `public_read` boolean field. See the main [README.md](../README.md#project-sharing) for details on sharing with specific users.

## Usage

### Creating a Public Project

When creating or updating a project, set `public_read` to `true`:

```json
{
  "project_handle": "my-public-project",
  "description": "A publicly accessible project",
  "public_read": true
}
```

You can also combine public access with user-specific sharing:

```json
{
  "project_handle": "my-project",
  "description": "A public project with additional editors",
  "public_read": true,
  "shared_with": [
    {
      "user_handle": "bob",
      "role": "editor"
    }
  ]
}
```

### Endpoints with Public Access

When a project has public read access enabled, the following endpoints can be accessed without authentication:

- `GET /v1/projects/{user}/{project}` - Retrieve project metadata (including owner and shared_with)
- `GET /v1/embeddings/{user}/{project}` - Retrieve all embeddings for the project
- `GET /v1/embeddings/{user}/{project}/{text_id}` - Retrieve a specific embedding
- `GET /v1/similars/{user}/{project}/{text_id}` - Find similar documents

### Endpoints Requiring Authentication

Even for public projects, the following operations still require authentication:

- `POST /v1/embeddings/{user}/{project}` - Create new embeddings
- `DELETE /v1/embeddings/{user}/{project}` - Delete all embeddings
- `DELETE /v1/embeddings/{user}/{project}/{text_id}` - Delete a specific embedding

## Implementation Details

### Database Schema

A `public_read` boolean flag is stored in the `projects` table to indicate whether a project allows public access.

### Authentication Flow

1. When a request is made to a reader-protected endpoint, the middleware checks if authentication is required
2. If the project has `public_read` set to true, the request is allowed without an Authorization header
3. Unauthenticated requests are logged with the user set to "public"
4. If `public_read` is false or not set, normal authentication rules apply

### Backwards Compatibility

The `public_read` flag defaults to `false`, so existing projects continue to require authentication for read operations unless explicitly updated.

### Project Metadata Display

When a project has `public_read` enabled:
- The project is accessible without authentication for read operations
- The `public_read` flag will be set to `true` in the project metadata
- Anonymous users can view project metadata including owner and description

## Security Considerations

- Public access only applies to read operations (GET requests)
- Write operations (POST, PUT, DELETE) always require authentication
- Project metadata and ownership information is publicly visible for public projects
- The admin and owner authentication mechanisms are unaffected

## Examples

### Accessing a Public Project Without Authentication

```bash
# Get project metadata without authentication
curl http://localhost:8080/v1/projects/alice/public-project
# Returns: {"project_handle": "public-project", "owner": "alice", "public_read": true, ...}

# Get all embeddings without authentication
curl http://localhost:8080/v1/embeddings/alice/public-project

# Get a specific embedding without authentication
curl http://localhost:8080/v1/embeddings/alice/public-project/text123

# Find similar documents without authentication
curl http://localhost:8080/v1/similars/alice/public-project/text123
```

### Creating Embeddings Still Requires Authentication

```bash
# This will fail with 401 Unauthorized
curl -X POST http://localhost:8080/v1/embeddings/alice/public-project \
  -H "Content-Type: application/json" \
  -d '{"embeddings": [...]}'

# This will succeed with a valid API key
curl -X POST http://localhost:8080/v1/embeddings/alice/public-project \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"embeddings": [...]}'
```

## Migration

Existing projects are not affected. The `public_read` flag defaults to `false`, so all existing projects continue to require authentication for read operations unless explicitly updated to set `public_read: true`.
