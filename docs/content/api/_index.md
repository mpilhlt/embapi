---
title: "API Reference"
weight: 4
---

# API Reference

Complete reference for the dhamps-vdb REST API.

## API Version

Current version: **v1**

All endpoints are prefixed with `/v1/` (e.g., `POST /v1/embeddings/{user}/{project}`).

## API Documentation

The complete, always up-to-date API specification is available at:

- **OpenAPI YAML**: `/openapi.yaml`
- **Interactive Documentation**: `/docs`

## Reference Sections

- [Authentication](authentication/) - API key authentication
- [Endpoints](endpoints/) - All available API endpoints
- [Query Parameters](query-parameters/) - Filtering and pagination
- [PATCH Updates](patch-updates/) - Partial resource updates
- [Error Handling](error-handling/) - Error responses and codes

## Quick Example

```bash
# Authenticate with API key
curl -X GET "https://api.example.com/v1/projects/alice" \
  -H "Authorization: Bearer your_api_key_here"
```

All API requests require authentication except for public project read operations.
