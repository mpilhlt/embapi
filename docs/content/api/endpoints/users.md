---
title: "Users"
weight: 1
---

# Users Endpoint

Manage user accounts and API keys. User creation is admin-only, but users can manage their own account information.

## Endpoints

### List All Users

Get a list of all registered user handles.

**Endpoint:** `GET /v1/users`

**Authentication:** Admin only

**Example:**

```bash
curl -X GET "https://api.example.com/v1/users" \
  -H "Authorization: Bearer admin_api_key"
```

**Response:**

```json
{
  "users": ["alice", "bob", "charlie"]
}
```

---

### Create User

Register a new user and generate their API key.

**Endpoint:** `POST /v1/users`

**Authentication:** Admin only

**Request Body:**

```json
{
  "user_handle": "alice",
  "name": "Alice Doe",
  "email": "alice@example.com"
}
```

**Parameters:**

- `user_handle` (string, required): Unique identifier for the user (alphanumeric, hyphens, underscores)
- `name` (string, optional): User's full name
- `email` (string, optional): User's email address

**Example:**

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

**⚠️ Important:** The `api_key` is only returned once during user creation. Store it securely - it cannot be recovered later.

**Error Responses:**

- `400 Bad Request`: Invalid user handle or missing required fields
- `409 Conflict`: User handle already exists

---

### Get User Information

Retrieve information about a specific user.

**Endpoint:** `GET /v1/users/{username}`

**Authentication:** Admin or the user themselves

**Example:**

```bash
curl -X GET "https://api.example.com/v1/users/alice" \
  -H "Authorization: Bearer alice_or_admin_api_key"
```

**Response:**

```json
{
  "user_handle": "alice",
  "name": "Alice Doe",
  "email": "alice@example.com"
}
```

**Note:** API key is never returned in GET responses for security reasons.

---

### Create or Update User (PUT)

Register a new user with a specific handle or update existing user information.

**Endpoint:** `PUT /v1/users/{username}`

**Authentication:** Admin only

**Request Body:**

```json
{
  "user_handle": "alice",
  "name": "Alice Doe",
  "email": "alice@example.com"
}
```

**Example:**

```bash
curl -X PUT "https://api.example.com/v1/users/alice" \
  -H "Authorization: Bearer admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "user_handle": "alice",
    "name": "Alice Smith",
    "email": "alice.smith@example.com"
  }'
```

**Response:**

```json
{
  "user_handle": "alice",
  "name": "Alice Smith",
  "email": "alice.smith@example.com",
  "api_key": "024v2013621509245f2e24"
}
```

---

### Delete User

Delete a user and all their associated resources (projects, LLM services, embeddings).

**Endpoint:** `DELETE /v1/users/{username}`

**Authentication:** Admin or the user themselves

**Example:**

```bash
curl -X DELETE "https://api.example.com/v1/users/alice" \
  -H "Authorization: Bearer alice_or_admin_api_key"
```

**Response:**

```json
{
  "message": "User alice deleted successfully"
}
```

**⚠️ Warning:** This operation is irreversible and will delete:
- All projects owned by the user
- All LLM service instances owned by the user
- All embeddings in the user's projects
- All sharing relationships

---

### Partial Update (PATCH)

Update specific user fields without providing all user data.

**Endpoint:** `PATCH /v1/users/{username}`

**Authentication:** Admin or the user themselves

**Request Body:**

```json
{
  "name": "Alice Smith"
}
```

**Example:**

```bash
curl -X PATCH "https://api.example.com/v1/users/alice" \
  -H "Authorization: Bearer alice_or_admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newemail@example.com"
  }'
```

See [PATCH Updates](../patch-updates/) for more details.

---

## User Properties

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `user_handle` | string | Yes | Unique identifier (alphanumeric, hyphens, underscores) |
| `name` | string | No | User's full name |
| `email` | string | No | User's email address |
| `api_key` | string | Read-only | Generated API key (only returned on creation) |

## Special User: _system

The `_system` user is a special internal user that:
- Owns global LLM service definitions
- Cannot be used for authentication
- Cannot be deleted
- Is created automatically during database migrations

Users can create LLM service instances from `_system` definitions.

## Common Errors

### 400 Bad Request

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid user_handle: must be alphanumeric with hyphens or underscores"
}
```

### 401 Unauthorized

```json
{
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or missing authorization credentials"
}
```

### 403 Forbidden

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "Only admin users can create new users"
}
```

### 404 Not Found

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "User 'alice' not found"
}
```

### 409 Conflict

```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "User 'alice' already exists"
}
```

## Related Documentation

- [Authentication](../authentication/) - API key authentication details
- [Projects](projects/) - Project management
- [LLM Services](llm-services/) - LLM service instances
