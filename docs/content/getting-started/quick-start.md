---
title: "Quick Start"
weight: 3
---

# Quick Start

Complete walkthrough from installation to searching similar documents using curl.

## Prerequisites

- dhamps-vdb installed and running
- Admin API key configured
- PostgreSQL with pgvector ready

## 1. Create a User

Create a new user with the admin API key:

```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_handle": "alice",
    "name": "Alice Smith",
    "email": "alice@example.com"
  }'
```

**Response:**

```json
{
  "user_handle": "alice",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "vdb_key": "024v2013621509245f2e24...",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Save the `vdb_key`** - it cannot be recovered later.

## 2. Create an LLM Service Instance

Create an LLM service configuration:

```bash
curl -X PUT http://localhost:8880/v1/llm-services/alice/my-openai \
  -H "Authorization: Bearer alice_vdb_key" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "description": "OpenAI large embeddings",
    "api_key_encrypted": "sk-proj-your-openai-key"
  }'
```

**Response:**

```json
{
  "instance_id": 1,
  "instance_handle": "my-openai",
  "owner": "alice",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "model": "text-embedding-3-large",
  "dimensions": 3072
}
```

## 3. Create a Project

Create a project to organize your embeddings:

```bash
curl -X POST http://localhost:8880/v1/projects/alice \
  -H "Authorization: Bearer alice_vdb_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "research-docs",
    "description": "Research document embeddings",
    "instance_owner": "alice",
    "instance_handle": "my-openai"
  }'
```

**Response:**

```json
{
  "project_id": 1,
  "project_handle": "research-docs",
  "owner": "alice",
  "description": "Research document embeddings",
  "instance_id": 1,
  "created_at": "2024-01-15T10:35:00Z"
}
```

## 4. Upload Embeddings

Upload document embeddings to your project:

```bash
curl -X POST http://localhost:8880/v1/embeddings/alice/research-docs \
  -H "Authorization: Bearer alice_vdb_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [
      {
        "text_id": "doc1",
        "instance_handle": "my-openai",
        "text": "Introduction to machine learning",
        "vector": [0.1, 0.2, 0.3, ..., 0.5],
        "vector_dim": 3072,
        "metadata": {
          "title": "ML Intro",
          "author": "Alice",
          "year": 2024
        }
      },
      {
        "text_id": "doc2",
        "instance_handle": "my-openai",
        "text": "Deep learning fundamentals",
        "vector": [0.15, 0.25, 0.35, ..., 0.55],
        "vector_dim": 3072,
        "metadata": {
          "title": "DL Fundamentals",
          "author": "Bob",
          "year": 2024
        }
      }
    ]
  }'
```

**Response:**

```json
{
  "message": "2 embeddings uploaded successfully"
}
```

## 5. Search for Similar Documents

### Option A: Search Using Stored Document

Find documents similar to an already-stored document:

```bash
curl -X GET "http://localhost:8880/v1/similars/alice/research-docs/doc1?count=5&threshold=0.7" \
  -H "Authorization: Bearer alice_vdb_key"
```

**Response:**

```json
{
  "user_handle": "alice",
  "project_handle": "research-docs",
  "results": [
    {
      "id": "doc2",
      "similarity": 0.92
    },
    {
      "id": "doc5",
      "similarity": 0.85
    }
  ]
}
```

### Option B: Search Using Raw Embeddings

Search without storing the query embedding:

```bash
curl -X POST "http://localhost:8880/v1/similars/alice/research-docs?count=5&threshold=0.7" \
  -H "Authorization: Bearer alice_vdb_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.12, 0.22, 0.32, ..., 0.52]
  }'
```

## 6. Filter by Metadata

Exclude documents from a specific author when searching:

```bash
curl -X GET "http://localhost:8880/v1/similars/alice/research-docs/doc1?count=5&metadata_path=author&metadata_value=Alice" \
  -H "Authorization: Bearer alice_vdb_key"
```

This excludes all documents where `metadata.author` equals "Alice".

## 7. Retrieve Embeddings

Get all embeddings in your project:

```bash
curl -X GET "http://localhost:8880/v1/embeddings/alice/research-docs?limit=10&offset=0" \
  -H "Authorization: Bearer alice_vdb_key"
```

Get a specific embedding:

```bash
curl -X GET http://localhost:8880/v1/embeddings/alice/research-docs/doc1 \
  -H "Authorization: Bearer alice_vdb_key"
```

## Complete Workflow Example

Here's a complete script to get started:

```bash
#!/bin/bash

# Configuration
API_URL="http://localhost:8880"
ADMIN_KEY="your-admin-key"

# 1. Create user
USER_RESPONSE=$(curl -s -X POST "$API_URL/v1/users" \
  -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_handle":"alice","name":"Alice Smith","email":"alice@example.com"}')

USER_KEY=$(echo $USER_RESPONSE | jq -r '.vdb_key')
echo "User created with key: $USER_KEY"

# 2. Create LLM service instance
curl -X PUT "$API_URL/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "api_key_encrypted": "sk-your-key"
  }'

# 3. Create project
curl -X POST "$API_URL/v1/projects/alice" \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "research-docs",
    "description": "Research documents",
    "instance_owner": "alice",
    "instance_handle": "my-openai"
  }'

# 4. Upload embeddings
curl -X POST "$API_URL/v1/embeddings/alice/research-docs" \
  -H "Authorization: Bearer $USER_KEY" \
  -H "Content-Type: application/json" \
  -d @embeddings.json

# 5. Search similar
curl -X GET "$API_URL/v1/similars/alice/research-docs/doc1?count=5" \
  -H "Authorization: Bearer $USER_KEY"

echo "Setup complete!"
```

## API Documentation

For complete API documentation, visit:

```bash
curl http://localhost:8880/docs
```

Or open http://localhost:8880/docs in your browser.

## Next Steps

- [Learn about projects](../concepts/projects/)
- [Understand embeddings](../concepts/embeddings/)
- [Explore sharing projects](../guides/project-sharing/)
- [Set up metadata validation](../guides/metadata-validation/)
