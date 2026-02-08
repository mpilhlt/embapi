---
title: "First Project"
weight: 4
---

# First Project

Step-by-step guide to creating your first complete project in dhamps-vdb.

## Overview

This guide walks you through creating a complete RAG (Retrieval Augmented Generation) workflow:

1. Set up authentication
2. Configure an LLM service
3. Create a project with metadata validation
4. Upload document embeddings
5. Search for similar documents
6. Share your project with collaborators

## Step 1: Authentication Setup

### Get Your API Key

If you're an admin, create your first user:

```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_handle": "researcher1",
    "name": "Research User",
    "email": "researcher@example.com"
  }'
```

Save the returned `vdb_key` to a variable:

```bash
export USER_KEY="your-returned-vdb-key"
```

### Verify Authentication

Test your API key:

```bash
curl -X GET http://localhost:8880/v1/users/researcher1 \
  -H "Authorization: Bearer $USER_KEY"
```

## Step 2: Configure LLM Service

### Option A: Use System Definition

List available system definitions:

```bash
curl -X GET http://localhost:8880/v1/llm-services/_system \
  -H "Authorization: Bearer $USER_KEY"
```

Create an instance from a system definition:

```bash
curl -X PUT http://localhost:8880/v1/llm-services/researcher1/my-embeddings \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "definition_owner": "_system",
    "definition_handle": "openai-large",
    "description": "My OpenAI embeddings instance",
    "api_key_encrypted": "sk-proj-your-openai-api-key"
  }'
```

### Option B: Create Custom Instance

Create a standalone instance with custom configuration:

```bash
curl -X PUT http://localhost:8880/v1/llm-services/researcher1/custom-embeddings \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-small",
    "dimensions": 1536,
    "description": "Custom OpenAI small embeddings",
    "api_key_encrypted": "sk-proj-your-api-key"
  }'
```

## Step 3: Create Project with Metadata Schema

Define a metadata schema to ensure consistent document metadata:

```bash
curl -X POST http://localhost:8880/v1/projects/researcher1 \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "literature-analysis",
    "description": "Literary texts for research analysis",
    "instance_owner": "researcher1",
    "instance_handle": "my-embeddings",
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"title\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"},\"genre\":{\"type\":\"string\",\"enum\":[\"poetry\",\"prose\",\"drama\"]},\"language\":{\"type\":\"string\"}},\"required\":[\"author\",\"title\",\"year\"]}"
  }'
```

This schema requires `author`, `title`, and `year` fields, with optional `genre` and `language` fields.

## Step 4: Upload Document Embeddings

### Prepare Your Data

Create a file `embeddings.json` with your document embeddings:

```json
{
  "embeddings": [
    {
      "text_id": "hamlet-act1-scene1",
      "instance_handle": "my-embeddings",
      "text": "Who's there? Nay, answer me: stand, and unfold yourself.",
      "vector": [0.023, -0.015, 0.087, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "William Shakespeare",
        "title": "Hamlet",
        "year": 1603,
        "genre": "drama",
        "language": "English"
      }
    },
    {
      "text_id": "paradise-lost-book1-line1",
      "instance_handle": "my-embeddings",
      "text": "Of Man's first disobedience, and the fruit...",
      "vector": [0.045, -0.032, 0.091, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "John Milton",
        "title": "Paradise Lost",
        "year": 1667,
        "genre": "poetry",
        "language": "English"
      }
    }
  ]
}
```

### Upload Embeddings

```bash
curl -X POST http://localhost:8880/v1/embeddings/researcher1/literature-analysis \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d @embeddings.json
```

### Verify Upload

List all embeddings:

```bash
curl -X GET "http://localhost:8880/v1/embeddings/researcher1/literature-analysis?limit=10" \
  -H "Authorization: Bearer $USER_KEY"
```

Get a specific embedding:

```bash
curl -X GET http://localhost:8880/v1/embeddings/researcher1/literature-analysis/hamlet-act1-scene1 \
  -H "Authorization: Bearer $USER_KEY"
```

## Step 5: Search Similar Documents

### Basic Similarity Search

Find passages similar to Hamlet Act 1:

```bash
curl -X GET "http://localhost:8880/v1/similars/researcher1/literature-analysis/hamlet-act1-scene1?count=5&threshold=0.7" \
  -H "Authorization: Bearer $USER_KEY"
```

**Response:**

```json
{
  "user_handle": "researcher1",
  "project_handle": "literature-analysis",
  "results": [
    {
      "id": "hamlet-act2-scene1",
      "similarity": 0.89
    },
    {
      "id": "macbeth-act1-scene3",
      "similarity": 0.82
    },
    {
      "id": "othello-act3-scene3",
      "similarity": 0.76
    }
  ]
}
```

### Search with Metadata Filtering

Exclude passages from the same work:

```bash
curl -X GET "http://localhost:8880/v1/similars/researcher1/literature-analysis/hamlet-act1-scene1?count=5&metadata_path=title&metadata_value=Hamlet" \
  -H "Authorization: Bearer $USER_KEY"
```

This excludes all documents where `metadata.title` equals "Hamlet".

### Search with Raw Embeddings

Search using a new embedding without storing it:

```bash
curl -X POST "http://localhost:8880/v1/similars/researcher1/literature-analysis?count=5&threshold=0.7" \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.034, -0.021, 0.092, ...]
  }'
```

## Step 6: Share Your Project

### Share with Collaborators

Grant read-only access to another user:

```bash
curl -X POST http://localhost:8880/v1/projects/researcher1/literature-analysis/share \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "colleague1",
    "role": "reader"
  }'
```

Grant edit access:

```bash
curl -X POST http://localhost:8880/v1/projects/researcher1/literature-analysis/share \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "colleague2",
    "role": "editor"
  }'
```

### Make Project Public

Enable public read access (no authentication required):

```bash
curl -X PATCH http://localhost:8880/v1/projects/researcher1/literature-analysis \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "public_read": true
  }'
```

Now anyone can read embeddings and search without authentication:

```bash
# No Authorization header needed
curl -X GET http://localhost:8880/v1/embeddings/researcher1/literature-analysis/hamlet-act1-scene1
```

### View Shared Users

List all users with access to your project:

```bash
curl -X GET http://localhost:8880/v1/projects/researcher1/literature-analysis/shared-with \
  -H "Authorization: Bearer $USER_KEY"
```

## Step 7: Manage Your Project

### Update Project Description

```bash
curl -X PATCH http://localhost:8880/v1/projects/researcher1/literature-analysis \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated: Shakespearean and Renaissance literature analysis"
  }'
```

### Update Metadata Schema

```bash
curl -X PATCH http://localhost:8880/v1/projects/researcher1/literature-analysis \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"title\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"},\"genre\":{\"type\":\"string\"},\"language\":{\"type\":\"string\"},\"act\":{\"type\":\"integer\"},\"scene\":{\"type\":\"integer\"}},\"required\":[\"author\",\"title\",\"year\"]}"
  }'
```

### Delete Specific Embeddings

```bash
curl -X DELETE http://localhost:8880/v1/embeddings/researcher1/literature-analysis/hamlet-act1-scene1 \
  -H "Authorization: Bearer $USER_KEY"
```

### Delete All Embeddings

```bash
curl -X DELETE http://localhost:8880/v1/embeddings/researcher1/literature-analysis \
  -H "Authorization: Bearer $USER_KEY"
```

## Common Patterns

### Batch Upload Script

```bash
#!/bin/bash

USER_KEY="your-vdb-key"
PROJECT="researcher1/literature-analysis"
API_URL="http://localhost:8880"

# Process multiple files
for file in data/*.json; do
  echo "Uploading $file..."
  curl -X POST "$API_URL/v1/embeddings/$PROJECT" \
    -H "Authorization: Bearer $USER_KEY" \
    -H "Content-Type: application/json" \
    -d @"$file"
done
```

### Search and Filter Workflow

```bash
# 1. Find similar documents
SIMILAR=$(curl -s -X GET "$API_URL/v1/similars/$PROJECT/doc1?count=20" \
  -H "Authorization: Bearer $USER_KEY")

# 2. Extract IDs
IDS=$(echo $SIMILAR | jq -r '.results[].id')

# 3. Retrieve full embeddings for similar documents
for id in $IDS; do
  curl -X GET "$API_URL/v1/embeddings/$PROJECT/$id" \
    -H "Authorization: Bearer $USER_KEY"
done
```

## Troubleshooting

### Validation Errors

If metadata validation fails:

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc1': year is required"
}
```

Check your metadata schema and ensure all required fields are present.

### Dimension Mismatches

If vector dimensions don't match:

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: expected 3072 dimensions, got 1536"
}
```

Verify your LLM service configuration and embedding dimensions.

### Authentication Errors

If you get 401 Unauthorized:

- Check your API key is correct
- Ensure `Authorization: Bearer` prefix is included
- Verify the user owns the resource or has been granted access

## Next Steps

- [Learn about metadata validation](../guides/metadata-validation/)
- [Explore batch operations](../guides/batch-operations/)
- [Understand similarity search](../concepts/similarity-search/)
- [Review API documentation](../api/)
