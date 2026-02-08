---
title: "Users and Authentication"
weight: 2
---

# Users and Authentication

dhamps-vdb uses token-based authentication with API keys for all operations.

## User Model

### User Properties

- **user_handle**: Unique identifier (3-20 characters, alphanumeric + underscore)
- **name**: Full name (optional)
- **email**: Email address (unique, required)
- **vdb_key**: API key (SHA-256 hash, 64 characters)
- **created_at**: Timestamp of creation
- **updated_at**: Timestamp of last update

### Special Users

**`_system` User**

- Created automatically during database migration
- Owns system-wide LLM service definitions
- Cannot be used for authentication
- Provides default configurations for all users

## Authentication Flow

### API Key Authentication

All requests (except public endpoints) require authentication:

```http
GET /v1/projects/alice/my-project
Authorization: Bearer 024v2013621509245f2e24...
```

### Authentication Process

1. Client sends API key in `Authorization` header with `Bearer` prefix
2. Server extracts key and looks up user in database
3. If user found, request proceeds with user context
4. If not found or missing, returns `401 Unauthorized`

### Admin Authentication

Administrative operations require the admin API key:

```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_handle":"alice","email":"alice@example.com"}'
```

Admin key is set via `SERVICE_ADMINKEY` environment variable.

## User Creation

### By Admin

Only admin users can create new users:

```bash
POST /v1/users
Authorization: Bearer ADMIN_KEY

{
  "user_handle": "researcher1",
  "name": "Research User",
  "email": "researcher@example.com"
}
```

**Response:**

```json
{
  "user_handle": "researcher1",
  "name": "Research User",
  "email": "researcher@example.com",
  "vdb_key": "024v2013621509245f2e24abcdef...",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Important:** Save the `vdb_key` immediately - it cannot be recovered later.

### User Handle Restrictions

- Must be 3-20 characters
- Alphanumeric characters and underscores only
- Must be unique
- Cannot be `_system`

## User Management

### Retrieve User Information

Users can view their own information:

```bash
GET /v1/users/alice
Authorization: Bearer alice_vdb_key
```

Admins can view any user:

```bash
GET /v1/users/alice
Authorization: Bearer ADMIN_KEY
```

### List All Users

Only admins can list all users:

```bash
GET /v1/users
Authorization: Bearer ADMIN_KEY
```

Returns array of user handles (not full user objects).

### Update User

Users can update their own information:

```bash
PATCH /v1/users/alice
Authorization: Bearer alice_vdb_key

{
  "name": "Alice Smith-Jones",
  "email": "alice.smith@example.com"
}
```

### Delete User

Users can delete their own account:

```bash
DELETE /v1/users/alice
Authorization: Bearer alice_vdb_key
```

Admins can delete any user:

```bash
DELETE /v1/users/alice
Authorization: Bearer ADMIN_KEY
```

**Cascading Deletion:**
- All user's projects are deleted
- All user's LLM service instances are deleted
- All embeddings in user's projects are deleted
- Sharing grants from this user to others are removed

## Authorization Model

### Resource Ownership

Users own three types of resources:

1. **Projects**: Collections of embeddings
2. **LLM Service Instances**: Embedding configurations with API keys
3. **LLM Service Definitions**: Reusable configuration templates (optional)

### Access Levels

**Owner**
- Full control over resource
- Can read, write, delete
- Can share with others
- Can transfer ownership (projects only)

**Editor** (via sharing)
- Read and write access
- Cannot delete resource
- Cannot modify sharing
- Cannot change project settings

**Reader** (via sharing)
- Read-only access
- Can view embeddings
- Can search for similar documents
- Cannot modify anything

**Public** (if project.public_read = true)
- Unauthenticated read access
- Can view embeddings
- Can search for similar documents
- Cannot write or modify

## Security Best Practices

### API Key Management

**Storage**
- Store API keys securely (e.g., environment variables, secret managers)
- Never commit API keys to version control
- Use different keys for development and production

**Rotation**
- Currently, API keys cannot be rotated
- To change a key, delete and recreate the user
- Plan key rotation strategy before production deployment

**Transmission**
- Always use HTTPS in production
- API keys are transmitted in Authorization header
- Never pass API keys in URL query parameters

### User Key vs. LLM API Keys

dhamps-vdb handles two types of keys:

1. **User API keys** (`vdb_key`): Authenticate users to dhamps-vdb
   - Stored as SHA-256 hash in database
   - Never encrypted (one-way hash)
   - Returned only once on user creation

2. **LLM API keys** (`api_key_encrypted`): Authenticate to LLM services
   - Stored encrypted (AES-256-GCM) in database
   - Never returned in API responses
   - Used internally for LLM processing

## Multi-User Workflows

### Collaboration Pattern

1. **Admin** creates user accounts for team members
2. **Project Owner** creates project with embeddings
3. **Owner** shares project with collaborators
4. **Readers** can search and view embeddings
5. **Editors** can add/modify embeddings

### Organization Pattern

1. **Admin** creates organizational users
2. **Each user** creates LLM service instances with their own API keys
3. **Users** create projects using their instances
4. **Projects** shared within organization as needed

### Public Access Pattern

1. **User** creates project with research data
2. **User** sets `public_read: true`
3. **Anyone** can access embeddings and search without authentication
4. **Only owner** can modify project

## User Limits

### Current Implementation

No enforced limits on:
- Number of projects per user
- Number of embeddings per project
- Number of LLM service instances per user
- Storage size per user

### Recommended Limits

For production deployments, consider implementing:
- Rate limiting per API key
- Storage quotas per user
- Maximum project count per user

## Example Workflows

### Create and Use User Account

```bash
# Admin creates user
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_handle": "researcher1",
    "email": "researcher@example.com",
    "name": "Research User"
  }'

# Save returned vdb_key
export USER_KEY="returned-vdb-key"

# User verifies access
curl -X GET http://localhost:8880/v1/users/researcher1 \
  -H "Authorization: Bearer $USER_KEY"

# User creates project
curl -X POST http://localhost:8880/v1/projects/researcher1 \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "my-project",
    "description": "My research project"
  }'
```

### Share Resources

```bash
# User shares project with colleague
curl -X POST http://localhost:8880/v1/projects/researcher1/my-project/share \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "colleague1",
    "role": "reader"
  }'

# Colleague accesses shared project
curl -X GET http://localhost:8880/v1/projects/researcher1/my-project \
  -H "Authorization: Bearer $COLLEAGUE_KEY"
```

## Troubleshooting

### 401 Unauthorized

**Possible causes:**
- Missing Authorization header
- Incorrect API key
- Expired or invalid key
- Using user key instead of admin key (or vice versa)

**Solution:**
- Verify API key is correct
- Check Authorization header format: `Bearer KEY`
- Ensure operation matches key type (admin vs. user)

### 403 Forbidden

**Possible causes:**
- User doesn't own resource
- User not granted access to shared resource
- Insufficient permissions (reader trying to edit)

**Solution:**
- Verify resource ownership
- Check sharing grants
- Ensure user has required role (editor for writes)

### User Creation Failed

**Possible causes:**
- User handle already exists
- Email already registered
- Invalid user handle format
- Not using admin key

**Solution:**
- Choose different user handle
- Use unique email address
- Check user handle format (3-20 chars, alphanumeric + underscore)
- Verify using admin API key

## Next Steps

- [Learn about projects](projects/)
- [Understand LLM services](llm-services/)
- [Explore project sharing](../guides/project-sharing/)
