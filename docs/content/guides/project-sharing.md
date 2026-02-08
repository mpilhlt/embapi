---
title: "Project Sharing Guide"
weight: 2
---

# Project Sharing Guide

This guide explains how to share projects with specific users for collaborative work in embapi.

## Overview

Project sharing allows you to grant other users access to your projects with different permission levels:
- **reader**: Read-only access to embeddings and similar documents
- **editor**: Read and write access to embeddings (can add/modify/delete embeddings)

Only the project owner can manage sharing settings and delete the project.

## Sharing During Project Creation

You can specify users to share with when creating a new project using the `shared_with` field:

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/collaborative-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "collaborative-project",
    "description": "A project shared with team members",
    "instance_id": 123,
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
  }'
```

In this example:
- `bob` can read embeddings and search for similar documents
- `charlie` can read embeddings, search, and also add/modify/delete embeddings
- `alice` (the owner) has full control including managing sharing and deleting the project

## Managing Sharing After Creation

### Share a Project with a User

Add a user to an existing project:

```bash
curl -X POST "https://api.example.com/v1/projects/alice/collaborative-project/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "david",
    "role": "reader"
  }'
```

**Response:**
```json
{
  "message": "Project shared successfully",
  "user": "david",
  "role": "reader"
}
```

### Update User's Role

To change a user's role, simply share again with the new role:

```bash
curl -X POST "https://api.example.com/v1/projects/alice/collaborative-project/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "david",
    "role": "editor"
  }'
```

This updates David's access from reader to editor.

### Unshare a Project from a User

Remove a user's access to a project:

```bash
curl -X DELETE "https://api.example.com/v1/projects/alice/collaborative-project/share/david" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**
```json
{
  "message": "Project unshared successfully",
  "user": "david"
}
```

### List Shared Users

View all users a project is shared with:

```bash
curl -X GET "https://api.example.com/v1/projects/alice/collaborative-project/shared-with" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**
```json
{
  "project_handle": "collaborative-project",
  "owner": "alice",
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

**Note:** Only the project owner can view the list of shared users. Users who have been granted access cannot see which other users also have access.

## What Shared Users Can Do

### As a Reader

Bob (with reader access) can:

```bash
# View project metadata
curl -X GET "https://api.example.com/v1/projects/alice/collaborative-project" \
  -H "Authorization: Bearer bob_api_key"

# Retrieve embeddings
curl -X GET "https://api.example.com/v1/embeddings/alice/collaborative-project" \
  -H "Authorization: Bearer bob_api_key"

# Get a specific embedding
curl -X GET "https://api.example.com/v1/embeddings/alice/collaborative-project/doc123" \
  -H "Authorization: Bearer bob_api_key"

# Search for similar documents
curl -X GET "https://api.example.com/v1/similars/alice/collaborative-project/doc123?count=5" \
  -H "Authorization: Bearer bob_api_key"
```

### As an Editor

Charlie (with editor access) can do everything a reader can, plus:

```bash
# Upload new embeddings
curl -X POST "https://api.example.com/v1/embeddings/alice/collaborative-project" \
  -H "Authorization: Bearer charlie_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc456",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {"author": "Charlie"}
    }]
  }'

# Delete specific embeddings
curl -X DELETE "https://api.example.com/v1/embeddings/alice/collaborative-project/doc456" \
  -H "Authorization: Bearer charlie_api_key"

# Delete all embeddings
curl -X DELETE "https://api.example.com/v1/embeddings/alice/collaborative-project" \
  -H "Authorization: Bearer charlie_api_key"
```

## What Shared Users Cannot Do

Even with editor access, shared users **cannot**:

- Delete the project
- Change project settings (description, instance, metadata schema)
- Manage sharing (add/remove other users)
- View the list of other shared users
- Transfer project ownership

These operations require owner privileges.

## Permission Summary Table

| Operation | Owner | Editor | Reader |
|-----------|-------|--------|--------|
| View project metadata | ✅ | ✅ | ✅ |
| Retrieve embeddings | ✅ | ✅ | ✅ |
| Search similar documents | ✅ | ✅ | ✅ |
| Add embeddings | ✅ | ✅ | ❌ |
| Modify embeddings | ✅ | ✅ | ❌ |
| Delete embeddings | ✅ | ✅ | ❌ |
| Update project settings | ✅ | ❌ | ❌ |
| Delete project | ✅ | ❌ | ❌ |
| Manage sharing | ✅ | ❌ | ❌ |
| View shared users list | ✅ | ❌ | ❌ |
| Transfer ownership | ✅ | ❌ | ❌ |

## Use Cases

### Research Team Collaboration

A research team can share a project where:
- The principal investigator (PI) is the owner
- Research assistants have editor access to upload new data
- External collaborators have reader access to query the data

```bash
# PI creates and shares the project
curl -X PUT "https://api.example.com/v1/projects/pi/research-corpus" \
  -H "Authorization: Bearer pi_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "research-corpus",
    "description": "Research team corpus",
    "instance_id": 123,
    "shared_with": [
      {"user_handle": "assistant1", "role": "editor"},
      {"user_handle": "assistant2", "role": "editor"},
      {"user_handle": "external_collab", "role": "reader"}
    ]
  }'
```

### Data Pipeline with Read-Only Access

Share processed embeddings with downstream consumers:

```bash
# Data processor creates project
curl -X PUT "https://api.example.com/v1/projects/processor/processed-data" \
  -H "Authorization: Bearer processor_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "processed-data",
    "description": "Processed embeddings for consumption",
    "instance_id": 456,
    "shared_with": [
      {"user_handle": "app_backend", "role": "reader"},
      {"user_handle": "analytics_team", "role": "reader"}
    ]
  }'
```

### Temporary Access

Grant temporary access to a consultant and revoke it later:

```bash
# Grant access
curl -X POST "https://api.example.com/v1/projects/alice/sensitive-project/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"share_with_handle": "consultant", "role": "reader"}'

# Revoke access when consultation is complete
curl -X DELETE "https://api.example.com/v1/projects/alice/sensitive-project/share/consultant" \
  -H "Authorization: Bearer alice_api_key"
```

## Combining with Public Access

You can combine user-specific sharing with public access (see [Public Projects Guide](./public-projects.md)):

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/mixed-access-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "mixed-access-project",
    "description": "Public read, specific editors",
    "instance_id": 123,
    "public_read": true,
    "shared_with": [
      {"user_handle": "bob", "role": "editor"}
    ]
  }'
```

In this case:
- Anyone can read the project (no authentication required)
- `bob` can also edit (add/modify/delete embeddings)
- `alice` retains full owner privileges

## Security Considerations

1. **Choose Roles Carefully**: Only grant editor access to trusted users who need to modify data
2. **Audit Access**: Regularly review the shared users list to ensure appropriate access levels
3. **Revoke Promptly**: Remove access immediately when users no longer need it
4. **Use Reader by Default**: Start with reader access and upgrade to editor only when necessary
5. **Consider Public Access**: For truly open data, use `public_read: true` instead of sharing with many users

## Related Documentation

- [Ownership Transfer Guide](./ownership-transfer.md) - Transfer project ownership
- [Public Projects Guide](./public-projects.md) - Make projects publicly accessible
- [Instance Management Guide](./instance-management.md) - Share LLM service instances

## Troubleshooting

### Cannot Share Project

**Error:** "Only the owner can share projects"

**Solution:** Only the project owner can manage sharing. If you need to share the project, ask the owner to add you, or consider [transferring ownership](./ownership-transfer.md).

### User Not Found

**Error:** "User not found"

**Solution:** The user must exist in the system before you can share a project with them. Ask the admin to create the user first.

### Cannot View Shared Users List

**Error:** "Forbidden"

**Solution:** Only the project owner can view the list of shared users. Shared users cannot see other shared users for privacy reasons.
