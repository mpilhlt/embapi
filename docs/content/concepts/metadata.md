---
title: "Metadata"
weight: 7
---

# Metadata

Structured JSON data attached to embeddings for organization, validation, and filtering.

## Overview

Metadata provides context and structure for your embeddings:

- **Organization**: Categorize and group documents
- **Filtering**: Exclude documents in similarity searches
- **Validation**: Ensure consistent structure (optional)
- **Context**: Store additional document information

## Metadata Structure

### Format

Metadata is JSON stored as JSONB in PostgreSQL:

```json
{
  "author": "William Shakespeare",
  "title": "Hamlet",
  "year": 1603,
  "genre": "drama"
}
```

### Types

Supported JSON types:
- **String**: `"author": "Shakespeare"`
- **Number**: `"year": 1603`
- **Boolean**: `"published": true`
- **Array**: `"tags": ["tragedy", "revenge"]`
- **Object**: `"author": {"name": "...", "id": "..."}`
- **Null**: `"notes": null`

### Nested Structure

Complex hierarchies are supported:

```json
{
  "document": {
    "id": "W0017",
    "type": "manuscript"
  },
  "author": {
    "name": "John Milton",
    "birth_year": 1608,
    "nationality": "English"
  },
  "publication": {
    "year": 1667,
    "publisher": "First Edition",
    "location": "London"
  },
  "tags": ["poetry", "epic", "religious"]
}
```

## Metadata Schemas

### Purpose

JSON Schema validation ensures consistent metadata across all project embeddings.

### Defining a Schema

Include `metadataScheme` when creating/updating project:

```bash
POST /v1/projects/alice

{
  "project_handle": "research",
  "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"author\"]}"
}
```

### Schema Format

Use [JSON Schema](https://json-schema.org/) (draft-07+):

```json
{
  "type": "object",
  "properties": {
    "author": {
      "type": "string",
      "minLength": 1
    },
    "year": {
      "type": "integer",
      "minimum": 1000,
      "maximum": 2100
    },
    "genre": {
      "type": "string",
      "enum": ["poetry", "prose", "drama"]
    }
  },
  "required": ["author", "year"]
}
```

### Validation Behavior

**With schema defined:**
- All embeddings validated on upload
- Invalid metadata rejected with detailed error
- Schema enforced consistently

**Without schema:**
- Any JSON metadata accepted
- No validation performed
- Maximum flexibility

### Common Patterns

See [Metadata Validation Guide](../guides/metadata-validation/) for examples.

## Using Metadata

### On Upload

Include metadata with each embedding:

```bash
POST /v1/embeddings/alice/research

{
  "embeddings": [
    {
      "text_id": "doc1",
      "instance_handle": "my-embeddings",
      "vector": [...],
      "metadata": {
        "author": "Shakespeare",
        "title": "Hamlet",
        "year": 1603
      }
    }
  ]
}
```

### In Responses

Metadata returned when retrieving embeddings:

```bash
GET /v1/embeddings/alice/research/doc1
```

```json
{
  "text_id": "doc1",
  "metadata": {
    "author": "Shakespeare",
    "title": "Hamlet",
    "year": 1603
  },
  "vector": [...],
  ...
}
```

## Metadata Filtering

### Exclusion Filter

Exclude documents where metadata matches value:

```bash
GET /v1/similars/alice/research/doc1?metadata_path=author&metadata_value=Shakespeare
```

**Result**: Returns similar documents **excluding** those with `metadata.author == "Shakespeare"`.

### Path Syntax

Use JSON path notation:

**Simple field:**
```
metadata_path=author
```

**Nested field:**
```
metadata_path=author.name
```

**Array element** (not currently supported):
```
metadata_path=tags[0]
```

### URL Encoding

Encode special characters:

```bash
# Space
metadata_value=John%20Doe

# Quotes (if needed)
metadata_value=%22quoted%20value%22
```

### Use Cases

**Exclude same work:**
```bash
?metadata_path=title&metadata_value=Hamlet
```

**Exclude same author:**
```bash
?metadata_path=author&metadata_value=Shakespeare
```

**Exclude same source:**
```bash
?metadata_path=source_id&metadata_value=corpus-a
```

**Exclude same category:**
```bash
?metadata_path=category&metadata_value=draft
```

See [Metadata Filtering Guide](../guides/metadata-filtering/) for detailed examples.

## Validation Examples

### Simple Schema

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

**Valid metadata:**
```json
{"author": "Shakespeare", "year": 1603}
{"author": "Milton"}
```

**Invalid metadata:**
```json
{"year": 1603}  // Missing required 'author'
{"author": 123}  // Wrong type (should be string)
```

### Schema with Constraints

```json
{
  "type": "object",
  "properties": {
    "title": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200
    },
    "rating": {
      "type": "number",
      "minimum": 0,
      "maximum": 5
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"},
      "minItems": 1,
      "maxItems": 10
    }
  }
}
```

### Schema with Enums

```json
{
  "type": "object",
  "properties": {
    "language": {
      "type": "string",
      "enum": ["en", "de", "fr", "es", "la"]
    },
    "status": {
      "type": "string",
      "enum": ["draft", "review", "published"]
    }
  }
}
```

## Storage and Performance

### Storage

Metadata stored as JSONB in PostgreSQL:

- **Efficient**: Binary storage format
- **Indexable**: Can create indexes on fields
- **Queryable**: Use PostgreSQL JSON operators

### Size Considerations

Typical metadata sizes:

- **Simple**: 50-200 bytes
- **Moderate**: 200-1000 bytes
- **Complex**: 1-5KB
- **Very large**: >5KB (consider storing elsewhere)

### Performance

**Metadata filtering:**
- JSONB queries are efficient
- Add indexes for frequently filtered fields
- Keep metadata reasonably sized

**Example index (if needed):**
```sql
CREATE INDEX idx_embeddings_author 
ON embeddings ((metadata->>'author'));
```

## Common Patterns

### Document Provenance

Track document source and history:

```json
{
  "source": {
    "corpus": "Shakespeare Works",
    "collection": "Tragedies",
    "document_id": "hamlet",
    "version": 2
  },
  "imported_at": "2024-01-15T10:30:00Z",
  "imported_by": "researcher1"
}
```

### Hierarchical Documents

Structure for nested documents:

```json
{
  "work": "Paradise Lost",
  "book": 1,
  "line": 1,
  "chapter": null,
  "section": "Invocation"
}
```

### Multi-Language Content

Track language and translation info:

```json
{
  "language": "en",
  "original_language": "la",
  "translated_by": "John Smith",
  "translation_year": 1850
}
```

### Research Metadata

Academic paper metadata:

```json
{
  "doi": "10.1234/example.2024.001",
  "authors": ["Alice Smith", "Bob Jones"],
  "journal": "Digital Humanities Review",
  "year": 2024,
  "keywords": ["NLP", "embeddings", "RAG"]
}
```

## Updating Metadata

### Current Limitation

Metadata cannot be updated directly. To change:

1. Delete embedding
2. Re-upload with updated metadata

```bash
# Delete
DELETE /v1/embeddings/alice/project/doc1

# Re-upload with new metadata
POST /v1/embeddings/alice/project
{
  "embeddings": [{
    "text_id": "doc1",
    "metadata": {...updated...},
    ...
  }]
}
```

## Schema Updates

### Updating Project Schema

Use PATCH to update schema:

```bash
PATCH /v1/projects/alice/research

{
  "metadataScheme": "{...new schema...}"
}
```

### Effect on Existing Embeddings

- **Existing embeddings**: Not revalidated
- **New embeddings**: Validated against new schema
- **Updates**: Validated against current schema

### Migration Strategy

When updating schema:

1. Update project schema
2. Verify new embeddings work
3. Optionally re-upload existing embeddings

## Validation Errors

### Common Errors

**Missing required field:**
```json
{
  "status": 400,
  "detail": "metadata validation failed: author is required"
}
```

**Wrong type:**
```json
{
  "status": 400,
  "detail": "metadata validation failed: year must be integer"
}
```

**Enum violation:**
```json
{
  "status": 400,
  "detail": "metadata validation failed: genre must be one of [poetry, prose, drama]"
}
```

### Debugging

To debug validation errors:

1. Check project schema: `GET /v1/projects/owner/project`
2. Validate metadata with online tool: [jsonschemavalidator.net](https://www.jsonschemavalidator.net/)
3. Review error message for specific field
4. Update metadata or schema as needed

## Best Practices

### Schema Design

- Start simple, add complexity as needed
- Use required fields for critical data
- Use enums for controlled vocabularies
- Document your schema

### Metadata Content

- Keep metadata focused and relevant
- Avoid redundant data
- Use consistent field names
- Consider future queries and filters

### Performance

- Keep metadata reasonably sized (<5KB)
- Index frequently queried fields
- Avoid deeply nested structures when possible

## Troubleshooting

### Validation Fails

**Problem**: Metadata doesn't validate

**Solutions:**
- Check project schema
- Verify metadata structure
- Test with JSON Schema validator
- Review error message details

### Filtering Not Working

**Problem**: Metadata filter doesn't exclude documents

**Solutions:**
- Verify field path is correct
- Check value matches exactly (case-sensitive)
- URL-encode special characters
- Confirm metadata field exists

### Schema Too Restrictive

**Problem**: Cannot upload valid documents

**Solutions:**
- Make fields optional (remove from `required`)
- Broaden type constraints
- Use `oneOf` for multiple valid formats
- Remove unnecessary validations

## Next Steps

- [Learn about metadata validation](../guides/metadata-validation/)
- [Explore metadata filtering](../guides/metadata-filtering/)
- [Understand similarity search](similarity-search/)
