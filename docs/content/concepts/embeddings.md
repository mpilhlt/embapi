---
title: "Embeddings"
weight: 4
---

# Embeddings

Embeddings are vector representations of text stored in embapi for similarity search and retrieval.

## What are Embeddings?

Embeddings are numerical representations (vectors) of text that capture semantic meaning:

- **Vector**: Array of floating-point numbers (e.g., 1536 or 3072 dimensions)
- **Dimensions**: Fixed length determined by LLM model
- **Similarity**: Vectors of similar text are close in vector space
- **Purpose**: Enable semantic search and retrieval

## Embedding Structure

### Required Fields

- **text_id**: Unique identifier for the document (max 300 characters)
- **instance_handle**: LLM service instance that generated the embedding
- **vector**: Array of float32 values (embedding vector)
- **vector_dim**: Declared dimension count (must match vector length)

### Optional Fields

- **text**: Original text content (for reference)
- **metadata**: Structured JSON data about the document

### Example

```json
{
  "text_id": "doc-123",
  "instance_handle": "my-openai",
  "text": "Introduction to machine learning concepts",
  "vector": [0.023, -0.015, 0.087, ..., 0.042],
  "vector_dim": 3072,
  "metadata": {
    "title": "ML Introduction",
    "author": "Alice",
    "year": 2024,
    "category": "tutorial"
  }
}
```

## Creating Embeddings

### Single Embedding

```bash
POST /v1/embeddings/alice/research-docs

{
  "embeddings": [
    {
      "text_id": "doc1",
      "instance_handle": "my-openai",
      "vector": [0.1, 0.2, ..., 0.3],
      "vector_dim": 3072,
      "metadata": {"author": "Alice"}
    }
  ]
}
```

### Batch Upload

```bash
POST /v1/embeddings/alice/research-docs

{
  "embeddings": [
    {
      "text_id": "doc1",
      "instance_handle": "my-openai",
      "vector": [...],
      "vector_dim": 3072
    },
    {
      "text_id": "doc2",
      "instance_handle": "my-openai",
      "vector": [...],
      "vector_dim": 3072
    },
    ...
  ]
}
```

**Batch upload tips:**
- Upload 100-1000 embeddings per request
- Use consistent instance_handle
- Ensure all vectors have same dimensions
- Include metadata for searchability

## Text Identifiers

### Format

Text IDs can be any string up to 300 characters:

**Common patterns:**
- **URLs**: `https://id.example.com/doc/123`
- **URNs**: `urn:example:doc:123`
- **Paths**: `/corpus/section1/doc123`
- **IDs**: `doc-abc-123-xyz`

### URL Encoding

URL-encode text IDs when using them in API paths:

```bash
# Original ID
text_id="https://id.example.com/texts/W0017:1.3.1"

# URL-encoded for API
encoded="https%3A%2F%2Fid.example.com%2Ftexts%2FW0017%3A1.3.1"

# Use in API call
GET /v1/embeddings/alice/project/$encoded
```

### Uniqueness

Text IDs must be unique within a project:
- Same ID in different projects: ✅ Allowed
- Same ID twice in one project: ❌ Conflict error

## Validation

### Dimension Validation

The system automatically validates vector dimensions:

**Checks performed:**
1. `vector_dim` matches declared instance dimensions
2. Actual `vector` array length matches `vector_dim`
3. All embeddings in project have consistent dimensions

**Example error:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch: embedding declares 3072 dimensions but LLM service 'my-openai' expects 1536 dimensions"
}
```

### Metadata Validation

If project has a metadata schema, all embeddings are validated:

**Example error:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc1': metadata validation failed:\n  - author is required\n  - year must be integer"
}
```

See [Metadata Validation Guide](../guides/metadata-validation/) for details.

## Retrieving Embeddings

### List All Embeddings

```bash
GET /v1/embeddings/alice/research-docs?limit=100&offset=0
```

Returns paginated list of embeddings with:
- text_id
- metadata
- vector_dim
- created_at

Vectors are included by default (can be large).

### Get Single Embedding

```bash
GET /v1/embeddings/alice/research-docs/doc1
```

Returns complete embedding including vector.

### Pagination

Use `limit` and `offset` for large projects:

```bash
# First page (0-99)
GET /v1/embeddings/alice/research-docs?limit=100&offset=0

# Second page (100-199)
GET /v1/embeddings/alice/research-docs?limit=100&offset=100

# Third page (200-299)
GET /v1/embeddings/alice/research-docs?limit=100&offset=200
```

## Updating Embeddings

Currently, embeddings cannot be updated directly. To modify:

1. **Delete** existing embedding
2. **Upload** new version with same text_id

```bash
# Delete old version
DELETE /v1/embeddings/alice/research-docs/doc1

# Upload new version
POST /v1/embeddings/alice/research-docs
{
  "embeddings": [{
    "text_id": "doc1",
    "instance_handle": "my-openai",
    "vector": [...new vector...],
    "vector_dim": 3072,
    "metadata": {...updated metadata...}
  }]
}
```

## Deleting Embeddings

### Delete Single Embedding

```bash
DELETE /v1/embeddings/alice/research-docs/doc1
```

### Delete All Embeddings

```bash
DELETE /v1/embeddings/alice/research-docs
```

**Warning:** This deletes all embeddings in the project permanently.

## Metadata

### Purpose

Metadata provides structured information about documents:

- **Filtering**: Exclude documents in similarity searches
- **Organization**: Categorize and group documents
- **Context**: Store additional document information
- **Validation**: Ensure consistent structure (with schema)

### Structure

Metadata is stored as JSONB in PostgreSQL:

```json
{
  "author": "William Shakespeare",
  "title": "Hamlet",
  "year": 1603,
  "act": 1,
  "scene": 1,
  "genre": "drama",
  "language": "English"
}
```

### Nested Metadata

Complex structures are supported:

```json
{
  "author": {
    "name": "William Shakespeare",
    "birth_year": 1564,
    "nationality": "English"
  },
  "publication": {
    "year": 1603,
    "publisher": "First Folio",
    "edition": 1
  },
  "tags": ["tragedy", "revenge", "madness"]
}
```

### Filtering by Metadata

Use metadata to exclude documents from similarity searches:

```bash
# Exclude documents from same author
GET /v1/similars/alice/project/doc1?metadata_path=author&metadata_value=Shakespeare
```

See [Metadata Filtering Guide](../guides/metadata-filtering/) for details.

## Storage Considerations

### Vector Storage

Vectors are stored using pgvector extension:

- **Type**: `vector(N)` where N is dimension count
- **Size**: 4 bytes per dimension + overhead
- **Example**: 3072-dimension vector ≈ 12KB

### Storage Calculation

Estimate storage per embedding:

```
Vector:   4 bytes × dimensions
Text ID:  length in bytes (avg ~50 bytes)
Text:     length in bytes (optional)
Metadata: JSON size (varies, avg ~500 bytes)
Overhead: ~100 bytes (indexes, etc.)

Example (3072-dim with metadata):
4 × 3072 + 50 + 500 + 100 ≈ 13KB per embedding
```

### Large Projects

For projects with millions of embeddings:

- Use pagination when listing
- Consider partial indexes for metadata
- Monitor database size
- Plan backup strategy

## Performance

### Upload Performance

- **Small batches** (1-10): ~100ms per request
- **Medium batches** (100-500): ~500ms-2s per request
- **Large batches** (1000+): ~2-10s per request

### Retrieval Performance

- **Single embedding**: <10ms
- **Paginated list** (100 items): ~50ms
- **Large project scan**: Use pagination

### Optimization Tips

- Batch uploads when possible
- Use appropriate page sizes
- Include only needed fields
- Monitor query performance

## Common Patterns

### Document Chunking

Split long documents into chunks:

```json
{
  "embeddings": [
    {
      "text_id": "doc1:chunk1",
      "text": "First part of document...",
      "vector": [...],
      "metadata": {"doc_id": "doc1", "chunk": 1}
    },
    {
      "text_id": "doc1:chunk2",
      "text": "Second part of document...",
      "vector": [...],
      "metadata": {"doc_id": "doc1", "chunk": 2}
    }
  ]
}
```

### Versioned Documents

Track document versions:

```json
{
  "text_id": "doc1:v2",
  "vector": [...],
  "metadata": {
    "doc_id": "doc1",
    "version": 2,
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### Multi-Language Documents

Store embeddings for different languages:

```json
{
  "embeddings": [
    {
      "text_id": "doc1:en",
      "text": "English version...",
      "vector": [...],
      "metadata": {"doc_id": "doc1", "language": "en"}
    },
    {
      "text_id": "doc1:de",
      "text": "Deutsche Version...",
      "vector": [...],
      "metadata": {"doc_id": "doc1", "language": "de"}
    }
  ]
}
```

## Troubleshooting

### Dimension Mismatch

**Error**: "vector dimension mismatch"

**Cause**: Vector dimensions don't match instance configuration

**Solution**:
- Check instance dimensions: `GET /v1/llm-services/owner/instance`
- Regenerate embeddings with correct model
- Ensure `vector_dim` matches actual vector length

### Metadata Validation Failed

**Error**: "metadata validation failed"

**Cause**: Metadata doesn't match project schema

**Solution**:
- Check project schema: `GET /v1/projects/owner/project`
- Update metadata to match schema
- Or update schema to accept metadata

### Text ID Conflict

**Error**: "embedding with text_id already exists"

**Cause**: Attempting to upload duplicate text_id

**Solution**:
- Use different text_id
- Delete existing embedding first
- Check for unintended duplicates

## Next Steps

- [Learn about similarity search](similarity-search/)
- [Explore metadata filtering](../guides/metadata-filtering/)
- [Understand LLM services](llm-services/)
