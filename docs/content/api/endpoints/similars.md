---
title: "Similars"
weight: 6
---

# Similarity Search Endpoints

Find similar documents using vector similarity search. The API provides two methods: searching from stored embeddings or searching with raw vectors without storing them.

## Endpoints

### GET Similar Documents (from stored embeddings)

Find documents similar to an already-stored document by its text identifier.

**Endpoint:** `GET /v1/similars/{username}/{projectname}/{identifier}`

**Authentication:** Admin, owner, authorized readers, or public if `public_read` is enabled

**Query Parameters:**

- `count` (integer, optional, default: 10, max: 200): Number of similar documents to return
- `threshold` (float, optional, default: 0.5, range: 0-1): Minimum similarity score threshold
- `limit` (integer, optional, default: 10, max: 200): Maximum number of results to return (alias for `count`)
- `offset` (integer, optional, default: 0): Pagination offset
- `metadata_path` (string, optional): Metadata field path for filtering (must be used with `metadata_value`)
- `metadata_value` (string, optional): Metadata value to exclude from results (must be used with `metadata_path`)

**Example - Basic search:**

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=5&threshold=0.7" \
  -H "Authorization: Bearer alice_api_key"
```

**Example - With metadata filtering:**

```bash
# Exclude documents with author="John Doe"
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=10&metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**

```json
{
  "$schema": "http://localhost:8080/schemas/SimilarResponseBody.json",
  "user_handle": "alice",
  "project_handle": "research-docs",
  "results": [
    {
      "id": "doc456",
      "similarity": 0.95
    },
    {
      "id": "doc789",
      "similarity": 0.87
    },
    {
      "id": "doc321",
      "similarity": 0.82
    }
  ]
}
```

---

### POST Similar Documents (from raw embeddings)

Find similar documents by submitting a raw embedding vector without storing it in the database. Useful for one-time queries or testing.

**Endpoint:** `POST /v1/similars/{username}/{projectname}`

**Authentication:** Admin, owner, authorized readers, or public if `public_read` is enabled

**Query Parameters:** Same as GET endpoint above

**Request Body:**

```json
{
  "vector": [-0.020850, 0.018522, 0.053270, 0.071384, 0.020003, ...]
}
```

The vector must be an array of float values with dimensions matching the project's LLM service instance configuration.

**Example - Basic search:**

```bash
curl -X POST "https://api.example.com/v1/similars/alice/research-docs?count=10&threshold=0.8" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [-0.020850, 0.018522, 0.053270, 0.071384, 0.020003]
  }'
```

**Example - With metadata filtering:**

```bash
# Exclude documents from the same category
curl -X POST "https://api.example.com/v1/similars/alice/research-docs?count=5&metadata_path=category&metadata_value=biology" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [-0.020850, 0.018522, 0.053270, ...]
  }'
```

**Response:** Same format as GET endpoint

---

## Query Parameters Reference

### count / limit

Maximum number of similar documents to return.

- **Type:** Integer
- **Default:** 10
- **Max:** 200
- **Note:** `count` and `limit` are aliases; use either one

**Example:**

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20"
```

---

### threshold

Minimum similarity score threshold. Only documents with similarity scores >= threshold are returned.

- **Type:** Float
- **Default:** 0.5
- **Range:** 0.0 to 1.0 (where 1.0 is most similar)

**Example:**

```bash
# Only return very similar documents (>= 0.8)
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?threshold=0.8"
```

---

### offset

Pagination offset for large result sets.

- **Type:** Integer
- **Default:** 0
- **Use:** Skip the first N results

**Example:**

```bash
# Get results 21-40
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20&offset=20"
```

---

### metadata_path

Metadata field path for filtering results. Must be used together with `metadata_value`.

- **Type:** String
- **Format:** JSON path notation (e.g., `"author"`, `"author.name"`, `"publication.year"`)
- **Use:** Exclude documents where metadata field matches a specific value

**Examples:**

```bash
# Simple field
metadata_path=author

# Nested field
metadata_path=author.name

# Deeply nested field
metadata_path=publication.journal.name
```

---

### metadata_value

Metadata value to exclude from results. Must be used together with `metadata_path`.

- **Type:** String
- **Use:** Excludes documents where the metadata field at `metadata_path` equals this value

**Example - Exclude same author:**

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=author&metadata_value=Alice%20Doe"
```

**Example - Exclude same category:**

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=category&metadata_value=research"
```

**Example - Nested field:**

```bash
# Exclude documents from same author ID
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=author.id&metadata_value=A0083"
```

---

## Response Format

Both GET and POST endpoints return the same response format:

```json
{
  "$schema": "http://localhost:8080/schemas/SimilarResponseBody.json",
  "user_handle": "alice",
  "project_handle": "research-docs",
  "results": [
    {
      "id": "doc456",
      "similarity": 0.95
    },
    {
      "id": "doc789",
      "similarity": 0.87
    },
    {
      "id": "doc321",
      "similarity": 0.82
    }
  ]
}
```

**Response Fields:**

- `$schema` (string): JSON schema reference
- `user_handle` (string): Project owner's username
- `project_handle` (string): Project identifier
- `results` (array): Array of similar documents, ordered by similarity (highest first)
  - `id` (string): Document text identifier
  - `similarity` (float): Cosine similarity score (0-1, where 1 is most similar)

---

## Similarity Calculation

### Cosine Distance

The API uses **cosine distance** (or equivalently, cosine similarity) to calculate vector similarity:

- **Range:** 0 to 1
- **1.0:** Identical vectors
- **0.0:** Orthogonal vectors (completely dissimilar)
- **Higher values:** More similar documents

### Dimension Filtering

The system automatically filters results to only include embeddings with matching dimensions. This ensures:
- Only embeddings with matching `vector_dim` are compared
- Only embeddings from the same project are considered
- Invalid comparisons are prevented

---

## Dimension Validation (POST only)

When using the POST endpoint with raw embeddings, the API validates:

1. The project has an associated LLM service instance
2. The submitted vector dimensions match the instance's configured dimensions
3. If dimensions don't match, a `400 Bad Request` error is returned

**Error example:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "vector dimension mismatch: expected 1536 dimensions, got 768"
}
```

---

## Metadata Filtering

Both endpoints support metadata filtering to exclude documents based on metadata field values. This uses **negative matching** (excludes documents where the field matches the value).

### Use Cases

**Exclude documents from the same source:**

```bash
# When finding similar documents to doc123, exclude others from the same author
curl -X GET ".../similars/alice/research-docs/doc123?metadata_path=author_id&metadata_value=A0083"
```

**Exclude documents from the same category:**

```bash
# Find similar documents in other categories
curl -X GET ".../similars/alice/research-docs/doc123?metadata_path=category&metadata_value=biology"
```

**Exclude documents with the same tag:**

```bash
# Find documents with similar content but different tags
curl -X POST ".../similars/alice/research-docs?metadata_path=primary_tag&metadata_value=machine-learning" \
  -d '{"vector": [...]}'
```

### Nested Field Access

Use dot notation for nested metadata fields:

```bash
# Exclude documents from the same author (nested field)
metadata_path=author.id&metadata_value=author-123

# Exclude documents from the same publication year
metadata_path=publication.year&metadata_value=2024

# Deeply nested field
metadata_path=source.journal.publisher&metadata_value=Springer
```

---

## Examples

### Basic Similarity Search

Find 5 most similar documents with at least 0.7 similarity:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=5&threshold=0.7" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Search with Raw Vector

Submit a vector without storing it:

```bash
curl -X POST "https://api.example.com/v1/similars/alice/research-docs?count=10" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [-0.020850, 0.018522, 0.053270, 0.071384, 0.020003]
  }'
```

---

### Search with Metadata Filtering

Find similar documents but exclude those from the same author:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=10&metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Paginated Results

Get the next page of results:

```bash
# Page 1
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20&offset=0"

# Page 2
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20&offset=20"

# Page 3
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20&offset=40"
```

---

### Complex Query

High threshold, metadata filtering, and pagination:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=50&threshold=0.9&offset=0&metadata_path=category&metadata_value=biology" \
  -H "Authorization: Bearer alice_api_key"
```

---

## Common Errors

### 400 Bad Request - Dimension Mismatch (POST only)

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "vector dimension mismatch: expected 1536 dimensions, got 768"
}
```

### 400 Bad Request - Missing metadata_value

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata_path requires metadata_value to be specified"
}
```

### 400 Bad Request - Invalid threshold

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "threshold must be between 0.0 and 1.0"
}
```

### 403 Forbidden

```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "You don't have permission to search this project"
}
```

### 404 Not Found - Project

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Project 'alice/research-docs' not found"
}
```

### 404 Not Found - Embedding (GET only)

```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Embedding 'doc123' not found in project 'alice/research-docs'"
}
```

---

## Performance Considerations

### Indexing

The database uses vector indexes for efficient similarity search. See the database migrations for index configuration.

### Result Limits

- Default limit: 10 results
- Maximum limit: 200 results
- Use pagination for large result sets

### Threshold Optimization

Higher thresholds reduce result set size and improve performance:

- **0.5-0.7:** Broad similarity (default)
- **0.7-0.85:** Moderate similarity
- **0.85-0.95:** High similarity
- **0.95-1.0:** Near-identical documents

---

## Related Documentation

- [Embeddings](embeddings/) - Storing embeddings for similarity search
- [Projects](projects/) - Project configuration
- [Query Parameters](../query-parameters/) - Complete parameter reference
- [Error Handling](../error-handling/) - Error response format
