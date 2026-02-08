---
title: "Projects"
weight: 3
---

# Projects

Projects organize embeddings and define their configuration, including LLM service instances and optional metadata validation.

## What is a Project?

A project is a collection of document embeddings that share:

- A single LLM service instance (embedding configuration)
- Optional metadata schema for validation
- Access control (ownership and sharing)
- Consistent vector dimensions

## Project Properties

### Core Fields

- **project_handle**: Unique identifier within owner's namespace (3-20 characters)
- **owner**: User who owns the project
- **description**: Human-readable project description
- **instance_id**: Reference to LLM service instance (required, 1:1 relationship)
- **metadataScheme**: Optional JSON Schema for metadata validation
- **public_read**: Boolean flag for public read access
- **created_at**: Creation timestamp
- **updated_at**: Last modification timestamp

### Unique Constraints

Projects are uniquely identified by `(owner, project_handle)`:
- User "alice" can have project "research"
- User "bob" can also have project "research"
- Same user cannot have two projects with same handle

## Creating Projects

### Basic Project

```bash
POST /v1/projects/alice

{
  "project_handle": "literature-study",
  "description": "Literary text analysis",
  "instance_owner": "alice",
  "instance_handle": "my-openai"
}
```

### Project with Metadata Schema

```bash
POST /v1/projects/alice

{
  "project_handle": "research-papers",
  "description": "Academic papers with structured metadata",
  "instance_owner": "alice",
  "instance_handle": "my-embeddings",
  "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"},\"doi\":{\"type\":\"string\"}},\"required\":[\"author\",\"year\"]}"
}
```

### Public Project

```bash
POST /v1/projects/alice

{
  "project_handle": "open-dataset",
  "description": "Publicly accessible research data",
  "instance_owner": "alice",
  "instance_handle": "my-embeddings",
  "public_read": true
}
```

### Shared Project

```bash
POST /v1/projects/alice

{
  "project_handle": "collaborative",
  "description": "Team collaboration project",
  "instance_owner": "alice",
  "instance_handle": "my-embeddings",
  "shared_with": [
    {
      "user_handle": "bob",
      "role": "editor"
    },
    {
      "user_handle": "charlie",
      "role": "reader"
    }
  ]
}
```

## Project-Instance Relationship

### One-to-One Constraint

Each project references exactly one LLM service instance:

```
Project → Instance (1:1)
  ├── Defines vector dimensions
  ├── Specifies embedding model
  └── Contains API configuration
```

**Why 1:1?**
- Ensures consistent dimensions across all embeddings
- Prevents dimension mismatches in similarity searches
- Simplifies validation and error handling

### Specifying Instance

Use owner and handle to reference an instance:

```json
{
  "instance_owner": "alice",
  "instance_handle": "my-openai"
}
```

The instance can be:
- Owned by project owner: `"instance_owner": "alice"`
- Shared with project owner: `"instance_owner": "bob"` (if bob shared with alice)

## Metadata Schemas

### Purpose

Metadata schemas ensure consistent, structured metadata across all embeddings in a project.

### Schema Format

Use JSON Schema (draft-07 or later):

```json
{
  "type": "object",
  "properties": {
    "author": {"type": "string"},
    "title": {"type": "string"},
    "year": {
      "type": "integer",
      "minimum": 1000,
      "maximum": 2100
    },
    "genre": {
      "type": "string",
      "enum": ["fiction", "non-fiction", "poetry"]
    }
  },
  "required": ["author", "title", "year"]
}
```

### Validation Behavior

- **With schema**: All embeddings validated on upload
- **Without schema**: Any JSON metadata accepted
- **Validation failure**: Upload rejected with detailed error
- **Schema updates**: Only apply to new/updated embeddings

### Example Schemas

See [Metadata Validation Guide](../guides/metadata-validation/) for detailed examples.

## Access Control

### Ownership

- **Owner**: User who created the project
- **Full control**: Read, write, delete, share, transfer
- **Cannot be removed**: Owner always has access

### Sharing

Projects can be shared with specific users:

**Reader Role**
- View embeddings
- Search for similar documents
- View project metadata
- Cannot modify anything

**Editor Role**
- All reader permissions
- Add embeddings
- Modify embeddings
- Delete embeddings
- Cannot delete project or change settings

**Managing Sharing:**

```bash
# Share with user
POST /v1/projects/alice/my-project/share
{
  "share_with_handle": "bob",
  "role": "reader"
}

# Unshare from user
DELETE /v1/projects/alice/my-project/share/bob

# List shared users
GET /v1/projects/alice/my-project/shared-with
```

### Public Access

Projects can allow unauthenticated read access:

```bash
PATCH /v1/projects/alice/my-project
{
  "public_read": true
}
```

With `public_read: true`:
- Anyone can view embeddings (no authentication)
- Anyone can search for similar documents
- Write operations still require authentication

See [Public Projects Guide](../guides/public-projects/) for details.

## Project Operations

### List Projects

List all projects owned by a user:

```bash
GET /v1/projects/alice
Authorization: Bearer alice_embapi_key
```

Returns array of project objects.

### Get Project Details

```bash
GET /v1/projects/alice/research
Authorization: Bearer alice_embapi_key
```

Returns full project object including:
- Configuration
- Instance reference
- Metadata schema
- Sharing information (owner only)

### Update Project

Use PATCH for partial updates:

```bash
PATCH /v1/projects/alice/research
{
  "description": "Updated description"
}
```

Use PUT for full replacement:

```bash
PUT /v1/projects/alice/research
{
  "project_handle": "research",
  "description": "Complete project configuration",
  "instance_owner": "alice",
  "instance_handle": "my-embeddings",
  "metadataScheme": "{...}"
}
```

### Delete Project

```bash
DELETE /v1/projects/alice/research
Authorization: Bearer alice_embapi_key
```

**Cascading deletion:**
- All embeddings in project deleted
- All sharing grants removed
- Project metadata removed

## Ownership Transfer

Transfer project to another user:

```bash
POST /v1/projects/alice/research/transfer-ownership
{
  "new_owner_handle": "bob"
}
```

**Effects:**
- Project owner changes to bob
- Project URL changes: `/v1/projects/bob/research`
- Alice loses all access (unless re-shared)
- All embeddings transferred
- Bob cannot already have project with same handle

See [Ownership Transfer Guide](../guides/ownership-transfer/) for details.

## Project Limits

### Current Implementation

No enforced limits on:
- Number of embeddings per project
- Project storage size
- Number of shared users

### Recommended Practices

For large projects:
- Use pagination when listing embeddings
- Batch upload embeddings
- Monitor database size
- Consider archiving old projects

## Common Patterns

### Research Project Workflow

```bash
# 1. Create project
POST /v1/projects/alice
{
  "project_handle": "study-2024",
  "description": "2024 Research Study",
  "instance_owner": "alice",
  "instance_handle": "my-embeddings"
}

# 2. Upload data
POST /v1/embeddings/alice/study-2024
{ ... embeddings ... }

# 3. Share with team
POST /v1/projects/alice/study-2024/share
{"share_with_handle": "bob", "role": "reader"}

# 4. Make public when published
PATCH /v1/projects/alice/study-2024
{"public_read": true}
```

### Multi-Project Organization

```bash
# Development project
POST /v1/projects/alice
{
  "project_handle": "dev-experiments",
  "instance_owner": "alice",
  "instance_handle": "dev-embeddings"
}

# Production project
POST /v1/projects/alice
{
  "project_handle": "prod-dataset",
  "instance_owner": "alice",
  "instance_handle": "prod-embeddings",
  "metadataScheme": "{...}"
}

# Archive project
POST /v1/projects/alice
{
  "project_handle": "archive-2023",
  "instance_owner": "alice",
  "instance_handle": "archive-embeddings",
  "public_read": true
}
```

## Troubleshooting

### Cannot Create Project

**Possible causes:**
- Project handle already exists for this user
- Invalid project handle format
- Instance doesn't exist or not accessible
- Missing required fields

**Solutions:**
- Choose different project handle
- Verify instance exists: `GET /v1/llm-services/owner`
- Check instance is owned or shared with you
- Include all required fields (instance_owner, instance_handle)

### Metadata Validation Fails

**Possible causes:**
- Metadata doesn't match schema
- Invalid JSON Schema format
- Schema too restrictive

**Solutions:**
- Test schema with online validator
- Verify embedding metadata matches schema
- Update schema or metadata as needed

### Cannot Share Project

**Possible causes:**
- Not project owner
- Target user doesn't exist
- Invalid role specified

**Solutions:**
- Only owner can share projects
- Verify user exists: `GET /v1/users/target`
- Use valid role: "reader" or "editor"

## Next Steps

- [Learn about embeddings](embeddings/)
- [Explore metadata validation](../guides/metadata-validation/)
- [Understand project sharing](../guides/project-sharing/)
