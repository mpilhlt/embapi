---
title: "Metadata Validation Guide"
weight: 5
---

# Metadata Validation Guide

This guide explains how to use JSON Schema validation to ensure consistent metadata structure across your embeddings.

## Overview

dhamps-vdb supports optional metadata validation using JSON Schema. When you define a metadata schema for a project, the API automatically validates all embedding metadata against that schema, ensuring data quality and consistency.

Benefits:
- Enforce consistent metadata structure across all embeddings
- Catch data entry errors early
- Document expected metadata fields
- Enable reliable metadata-based filtering

## Defining a Metadata Schema

### When Creating a Project

Include a `metadataScheme` field with a valid JSON Schema when creating a project:

```bash
curl -X POST "https://api.example.com/v1/projects/alice/validated-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "validated-project",
    "description": "Project with metadata validation",
    "instance_id": 123,
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"author\"]}"
  }'
```

### Updating an Existing Project

Add or update a metadata schema using PATCH:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"title\":{\"type\":\"string\"},\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"title\",\"author\"]}"
  }'
```

**Note:** Schema updates only affect new or updated embeddings. Existing embeddings are not retroactively validated.

## Common Schema Patterns

### Simple Required Fields

Require specific fields with basic types:

```json
{
  "type": "object",
  "properties": {
    "author": {"type": "string"},
    "year": {"type": "integer"}
  },
  "required": ["author"]
}
```

Example usage:

```bash
curl -X POST "https://api.example.com/v1/projects/alice/literary-texts" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "literary-texts",
    "description": "Literary texts with structured metadata",
    "instance_id": 123,
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"author\"]}"
  }'
```

### Using Enums for Controlled Values

Restrict fields to specific allowed values:

```json
{
  "type": "object",
  "properties": {
    "genre": {
      "type": "string",
      "enum": ["poetry", "prose", "drama", "essay"]
    },
    "language": {
      "type": "string",
      "enum": ["en", "de", "fr", "es", "la"]
    }
  },
  "required": ["genre"]
}
```

Example:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/literary-texts" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "hamlet-soliloquy",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "William Shakespeare",
        "year": 1603,
        "genre": "drama"
      }
    }]
  }'
```

### Nested Objects

Define structured metadata with nested objects:

```json
{
  "type": "object",
  "properties": {
    "author": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "birth_year": {"type": "integer"},
        "nationality": {"type": "string"}
      },
      "required": ["name"]
    },
    "publication": {
      "type": "object",
      "properties": {
        "year": {"type": "integer"},
        "publisher": {"type": "string"},
        "city": {"type": "string"}
      }
    }
  },
  "required": ["author"]
}
```

Example:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/academic-papers" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "paper001",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": {
          "name": "Jane Smith",
          "birth_year": 1975,
          "nationality": "USA"
        },
        "publication": {
          "year": 2023,
          "publisher": "Academic Press",
          "city": "Boston"
        }
      }
    }]
  }'
```

### Arrays and Lists

Define arrays of values:

```json
{
  "type": "object",
  "properties": {
    "keywords": {
      "type": "array",
      "items": {"type": "string"},
      "minItems": 1,
      "maxItems": 10
    },
    "categories": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["philosophy", "literature", "science", "history"]
      }
    }
  }
}
```

Example:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/research-docs" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc001",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "keywords": ["machine learning", "embeddings", "NLP"],
        "categories": ["science", "literature"]
      }
    }]
  }'
```

### Numeric Constraints

Apply minimum, maximum, and range constraints:

```json
{
  "type": "object",
  "properties": {
    "rating": {
      "type": "number",
      "minimum": 0,
      "maximum": 5
    },
    "page_count": {
      "type": "integer",
      "minimum": 1
    },
    "confidence": {
      "type": "number",
      "minimum": 0.0,
      "maximum": 1.0
    }
  }
}
```

### String Constraints

Apply length and pattern constraints:

```json
{
  "type": "object",
  "properties": {
    "title": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "isbn": {
      "type": "string",
      "pattern": "^[0-9]{13}$"
    },
    "doi": {
      "type": "string",
      "pattern": "^10\\.\\d{4,}/[\\w\\-\\.]+$"
    }
  }
}
```

## Validation Examples

### Valid Upload

When metadata conforms to the schema:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/literary-texts" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "kant-critique",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "Immanuel Kant",
        "year": 1781,
        "genre": "prose"
      }
    }]
  }'
```

**Response:**
```json
{
  "message": "Embeddings uploaded successfully",
  "count": 1
}
```

### Validation Error: Missing Required Field

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/literary-texts" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "some-text",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "year": 1781
      }
    }]
  }'
```

**Error Response:**
```json
{
  "$schema": "http://localhost:8080/schemas/ErrorModel.json",
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'some-text': metadata validation failed:\n  - author is required"
}
```

### Validation Error: Wrong Type

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/literary-texts" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "some-text",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "John Doe",
        "year": "1781"
      }
    }]
  }'
```

**Error Response:**
```json
{
  "$schema": "http://localhost:8080/schemas/ErrorModel.json",
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'some-text': metadata validation failed:\n  - year: expected integer, got string"
}
```

### Validation Error: Invalid Enum Value

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/literary-texts" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "some-text",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "John Doe",
        "year": 1781,
        "genre": "novel"
      }
    }]
  }'
```

**Error Response:**
```json
{
  "$schema": "http://localhost:8080/schemas/ErrorModel.json",
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'some-text': metadata validation failed:\n  - genre: value must be one of: poetry, prose, drama, essay"
}
```

### Validation Error: Value Out of Range

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/rated-content" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "review001",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {
        "rating": 7.5
      }
    }]
  }'
```

**Error Response:**
```json
{
  "$schema": "http://localhost:8080/schemas/ErrorModel.json",
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'review001': metadata validation failed:\n  - rating: must be <= 5"
}
```

## Real-World Schema Examples

### Academic Publications

```json
{
  "type": "object",
  "properties": {
    "doi": {
      "type": "string",
      "pattern": "^10\\.\\d{4,}/[\\w\\-\\.]+$"
    },
    "title": {
      "type": "string",
      "minLength": 1,
      "maxLength": 500
    },
    "authors": {
      "type": "array",
      "items": {"type": "string"},
      "minItems": 1
    },
    "year": {
      "type": "integer",
      "minimum": 1900,
      "maximum": 2100
    },
    "journal": {"type": "string"},
    "volume": {"type": "integer"},
    "pages": {"type": "string"},
    "keywords": {
      "type": "array",
      "items": {"type": "string"}
    }
  },
  "required": ["doi", "title", "authors", "year"]
}
```

### Legal Documents

```json
{
  "type": "object",
  "properties": {
    "case_number": {"type": "string"},
    "court": {"type": "string"},
    "date": {
      "type": "string",
      "pattern": "^\\d{4}-\\d{2}-\\d{2}$"
    },
    "jurisdiction": {
      "type": "string",
      "enum": ["federal", "state", "local"]
    },
    "category": {
      "type": "string",
      "enum": ["civil", "criminal", "administrative"]
    },
    "parties": {
      "type": "array",
      "items": {"type": "string"}
    }
  },
  "required": ["case_number", "court", "date"]
}
```

### Product Catalog

```json
{
  "type": "object",
  "properties": {
    "sku": {
      "type": "string",
      "pattern": "^[A-Z]{3}-\\d{6}$"
    },
    "name": {"type": "string"},
    "category": {
      "type": "string",
      "enum": ["electronics", "clothing", "books", "home", "toys"]
    },
    "price": {
      "type": "number",
      "minimum": 0
    },
    "in_stock": {"type": "boolean"},
    "tags": {
      "type": "array",
      "items": {"type": "string"}
    }
  },
  "required": ["sku", "name", "category", "price"]
}
```

## Admin Sanity Check

Administrators can verify database integrity using the `/v1/admin/sanity-check` endpoint:

```bash
curl -X GET "https://api.example.com/v1/admin/sanity-check" \
  -H "Authorization: Bearer admin_api_key"
```

**Response:**
```json
{
  "status": "PASSED",
  "total_projects": 5,
  "issues_count": 0,
  "warnings_count": 1,
  "warnings": [
    "Project alice/project1 has 100 embeddings but no metadata schema defined"
  ]
}
```

**Status Values:**
- `PASSED`: No issues or warnings found
- `WARNING`: No critical issues, but warnings exist
- `FAILED`: Validation issues found that need attention

The sanity check:
- Validates all embeddings have dimensions matching their LLM service
- Validates all metadata against project schemas (if defined)
- Reports projects without schemas as warnings

## Best Practices

### 1. Start Simple, Add Complexity Later

Begin with basic required fields:

```json
{
  "type": "object",
  "properties": {
    "source": {"type": "string"}
  },
  "required": ["source"]
}
```

Add more constraints as your needs evolve.

### 2. Test Schemas Before Deployment

Use online JSON Schema validators like [jsonschemavalidator.net](https://www.jsonschemavalidator.net/) to test your schemas before deploying them.

### 3. Document Your Schema

Include a description in your project:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Project with metadata schema: author (required string), year (integer), genre (enum)"
  }'
```

### 4. Version Your Schemas

If you need to change a schema significantly, consider creating a new project rather than updating the existing one.

### 5. Optional vs Required

Be judicious with `required` fields. Too many required fields can make uploads cumbersome.

### 6. Escape JSON Properly

When passing JSON schemas in curl commands, escape quotes properly or use single quotes for the outer JSON.

## Projects Without Schemas

If you don't provide a `metadataScheme` when creating a project:
- Metadata validation is skipped
- You can upload any valid JSON metadata
- This is useful for exploratory work or heterogeneous data

## Schema Updates and Existing Data

When you update a project's metadata schema:
- Existing embeddings are **not** revalidated
- The new schema only applies to new or updated embeddings
- Use the admin sanity check to find existing embeddings that don't conform

## Removing a Schema

To remove metadata validation from a project:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"metadataScheme": null}'
```

After this, new embeddings can have any metadata structure.

## Related Documentation

- [RAG Workflow Guide](./rag-workflow.md) - Complete RAG implementation
- [Metadata Filtering Guide](./metadata-filtering.md) - Filter search results by metadata
- [Batch Operations Guide](./batch-operations.md) - Upload large datasets efficiently

## Troubleshooting

### Schema Syntax Errors

**Error:** "Invalid JSON Schema"

**Solution:** Validate your schema syntax using an online validator. Common issues:
- Missing commas between properties
- Unescaped quotes
- Invalid JSON structure

### Uploads Failing After Schema Change

**Problem:** Uploads worked before, now failing with validation errors

**Solution:** Check that your metadata matches the new schema requirements. Review the error message for specific validation failures.

### Want to Fix Non-Conforming Data

**Problem:** Sanity check shows validation errors in existing data

**Solution:** Either:
1. Update the schema to accept existing data
2. Re-upload conforming data to replace non-conforming embeddings
3. Delete and re-upload the project with correct metadata
