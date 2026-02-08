---
title: "Endpoints"
weight: 1
---

# API Endpoints

Complete reference for all embapi API endpoints.

## Endpoint Categories

- [Users](users/) - User management
- [Projects](projects/) - Project operations
- [LLM Services](llm-services/) - LLM service instances
- [API Standards](api-standards/) - API standard definitions
- [Embeddings](embeddings/) - Embedding storage and retrieval
- [Similars](similars/) - Similarity search

## Endpoint Format

All endpoints follow the pattern:

```
{METHOD} /v1/{resource}/{user}/{identifier}
```

Where:
- `METHOD`: HTTP method (GET, POST, PUT, DELETE, PATCH)
- `resource`: Resource type (users, projects, embeddings, etc.)
- `user`: User handle (owner of the resource)
- `identifier`: Specific resource identifier

## Authentication

Most endpoints require authentication via the `Authorization` header:

```
Authorization: Bearer your_api_key_here
```

Public projects allow unauthenticated access to read operations.
