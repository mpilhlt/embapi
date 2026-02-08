---
title: "Public Projects Guide"
weight: 3
---

# Public Projects Guide

This guide explains how to make projects publicly accessible, allowing anyone to read embeddings and search for similar documents without authentication.

## Overview

Projects can be configured to allow unauthenticated (public) read access by setting the `public_read` field to `true`. This is useful for:
- Open datasets and research data
- Public APIs and services
- Shared knowledge bases
- Educational resources

**Important:** Public access only applies to read operations. Write operations (creating, updating, or deleting embeddings) always require authentication.

## Creating a Public Project

Set `public_read` to `true` when creating a project:

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "public-knowledge",
    "description": "Publicly accessible knowledge base",
    "instance_id": 123,
    "public_read": true
  }'
```

**Response:**
```json
{
  "project_handle": "public-knowledge",
  "owner": "alice",
  "description": "Publicly accessible knowledge base",
  "instance_id": 123,
  "public_read": true
}
```

## Making an Existing Project Public

Update an existing project using PATCH:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"public_read": true}'
```

## Making a Public Project Private

To disable public access:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"public_read": false}'
```

## Accessing Public Projects Without Authentication

Once a project has `public_read` enabled, anyone can access it without providing an API key.

### Get Project Metadata

```bash
curl -X GET "https://api.example.com/v1/projects/alice/public-knowledge"
```

**Response:**
```json
{
  "project_handle": "public-knowledge",
  "owner": "alice",
  "description": "Publicly accessible knowledge base",
  "instance_id": 123,
  "public_read": true
}
```

### Retrieve All Embeddings

```bash
curl -X GET "https://api.example.com/v1/embeddings/alice/public-knowledge?limit=100"
```

**Response:**
```json
{
  "embeddings": [
    {
      "text_id": "doc001",
      "text": "Public document content",
      "metadata": {"category": "science"},
      "vector_dim": 3072
    },
    {
      "text_id": "doc002",
      "text": "Another public document",
      "metadata": {"category": "history"},
      "vector_dim": 3072
    }
  ]
}
```

### Get a Specific Embedding

```bash
curl -X GET "https://api.example.com/v1/embeddings/alice/public-knowledge/doc001"
```

**Response:**
```json
{
  "text_id": "doc001",
  "text": "Public document content",
  "metadata": {"category": "science"},
  "vector_dim": 3072,
  "vector": [0.021, -0.015, 0.043, ...]
}
```

### Search for Similar Documents

```bash
# Search by existing document ID
curl -X GET "https://api.example.com/v1/similars/alice/public-knowledge/doc001?count=5&threshold=0.7"
```

**Response:**
```json
{
  "user_handle": "alice",
  "project_handle": "public-knowledge",
  "results": [
    {"id": "doc002", "similarity": 0.92},
    {"id": "doc003", "similarity": 0.85},
    {"id": "doc004", "similarity": 0.78}
  ]
}
```

### Search with Raw Embeddings

```bash
curl -X POST "https://api.example.com/v1/similars/alice/public-knowledge?count=5" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.032, -0.018, 0.056, ...]
  }'
```

## Operations Still Requiring Authentication

Even for public projects, these operations require authentication:

### Creating Embeddings (Requires Auth)

```bash
# This will fail with 401 Unauthorized
curl -X POST "https://api.example.com/v1/embeddings/alice/public-knowledge" \
  -H "Content-Type: application/json" \
  -d '{"embeddings": [...]}'

# This succeeds with valid API key
curl -X POST "https://api.example.com/v1/embeddings/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc123",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072
    }]
  }'
```

### Deleting Embeddings (Requires Auth)

```bash
# Delete specific embedding (requires auth)
curl -X DELETE "https://api.example.com/v1/embeddings/alice/public-knowledge/doc001" \
  -H "Authorization: Bearer alice_api_key"

# Delete all embeddings (requires auth)
curl -X DELETE "https://api.example.com/v1/embeddings/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key"
```

### Modifying Project Settings (Requires Auth)

```bash
# Update project description (requires auth)
curl -X PATCH "https://api.example.com/v1/projects/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated description"}'
```

### Deleting Project (Requires Auth)

```bash
curl -X DELETE "https://api.example.com/v1/projects/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key"
```

## Combining Public Access with User Sharing

You can combine public read access with user-specific editor permissions:

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/collaborative-public" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "collaborative-public",
    "description": "Public read, restricted write",
    "instance_id": 123,
    "public_read": true,
    "shared_with": [
      {
        "user_handle": "bob",
        "role": "editor"
      },
      {
        "user_handle": "charlie",
        "role": "editor"
      }
    ]
  }'
```

In this configuration:
- **Anyone** can read embeddings and search (no auth required)
- **bob** and **charlie** can add/modify/delete embeddings (with auth)
- **alice** (owner) has full control (with auth)

## Use Cases

### Open Research Dataset

Share research data publicly while maintaining write control:

```bash
curl -X PUT "https://api.example.com/v1/projects/university/research-corpus" \
  -H "Authorization: Bearer university_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "research-corpus",
    "description": "Open research corpus for academic use",
    "instance_id": 456,
    "public_read": true,
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"doi\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"doi\"]}"
  }'
```

Researchers worldwide can access the data without credentials, but only authorized users can add new data.

### Public API Backend

Build a public search API on top of embapi:

```python
import requests

def public_search_api(query_vector, count=10):
    """Public search function requiring no authentication"""
    response = requests.post(
        "https://api.example.com/v1/similars/company/product-docs",
        json={"vector": query_vector},
        params={"count": count, "threshold": 0.6}
    )
    return response.json()

# No API key needed for public projects!
results = public_search_api(query_embedding)
```

### Educational Resources

Share educational content publicly:

```bash
curl -X PUT "https://api.example.com/v1/projects/edu/learning-materials" \
  -H "Authorization: Bearer edu_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "learning-materials",
    "description": "Free educational content embeddings",
    "instance_id": 789,
    "public_read": true
  }'
```

Students and educators can access the materials without creating accounts.

### Community-Driven Knowledge Base

Open knowledge base with restricted editors:

```bash
curl -X PUT "https://api.example.com/v1/projects/community/wiki-embeddings" \
  -H "Authorization: Bearer community_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "wiki-embeddings",
    "description": "Community wiki embeddings",
    "instance_id": 321,
    "public_read": true,
    "shared_with": [
      {"user_handle": "moderator1", "role": "editor"},
      {"user_handle": "moderator2", "role": "editor"}
    ]
  }'
```

## Security Considerations

### What is Publicly Visible

When `public_read` is enabled:
- ✅ Project metadata (name, description, owner)
- ✅ All embedding vectors and text content
- ✅ All embedding metadata
- ✅ Vector dimensions and instance references
- ❌ API keys (never exposed)
- ❌ User passwords or credentials

### Best Practices

1. **Review Content First**: Ensure no sensitive information is in embeddings or metadata before enabling public access
2. **Use Metadata Schemas**: Enforce consistent metadata structure with validation
3. **Monitor Usage**: Track access patterns to your public projects
4. **Set Clear Descriptions**: Provide clear project descriptions explaining the data's purpose and licensing
5. **Consider Rate Limiting**: For high-traffic public APIs, implement rate limiting at the application level

### What to Avoid

❌ **Don't** make projects public that contain:
- Personal identifiable information (PII)
- Proprietary or confidential data
- Sensitive research data not yet published
- Internal company information

✅ **Do** make projects public that contain:
- Already-published research data
- Open educational resources
- Public domain content
- Creative Commons licensed materials

## Disabling Public Access

If you need to make a public project private again:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/public-knowledge" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"public_read": false}'
```

After this change:
- All read operations require authentication
- Existing anonymous access is immediately revoked
- No data is deleted, just access is restricted

## Checking if a Project is Public

View project metadata to check the `public_read` flag:

```bash
curl -X GET "https://api.example.com/v1/projects/alice/public-knowledge"
```

Look for `"public_read": true` in the response.

## Related Documentation

- [Project Sharing Guide](./project-sharing.md) - Share with specific users
- [RAG Workflow Guide](./rag-workflow.md) - Complete RAG implementation
- [Metadata Validation Guide](./metadata-validation.md) - Enforce data quality

## Troubleshooting

### Public Access Not Working

**Symptom:** Still getting 401 Unauthorized for public project

**Solutions:**
1. Verify `public_read: true` is set:
   ```bash
   curl -X GET "https://api.example.com/v1/projects/alice/my-project"
   ```
2. Check you're using GET/POST for similars (not other methods)
3. Ensure the project exists and handle is correct

### Accidentally Made Project Public

**Solution:** Immediately disable public access:
```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"public_read": false}'
```

### Want to Track Public Usage

**Solution:** Anonymous requests are logged with user set to "public". Review server logs to monitor public access patterns.
