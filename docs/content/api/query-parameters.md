---
title: "Query Parameters"
weight: 7
---

# Query Parameters Reference

Comprehensive reference for query parameters used across API endpoints for pagination, filtering, and search configuration.

## Pagination Parameters

### limit

Maximum number of results to return in a single response.

**Type:** Integer  
**Default:** 10  
**Maximum:** 200  
**Minimum:** 1

**Used by:**
- `GET /v1/embeddings/{user}/{project}` - List embeddings
- `GET /v1/similars/{user}/{project}/{id}` - Similarity search
- `POST /v1/similars/{user}/{project}` - Raw vector search
- `GET /v1/projects/{user}` - List projects

**Example:**

```bash
# Get 50 embeddings
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs?limit=50" \
  -H "Authorization: Bearer alice_api_key"
```

**Aliases:**
- `count` - Used in similarity search endpoints (same behavior as `limit`)

---

### offset

Number of results to skip before returning data. Used for pagination.

**Type:** Integer  
**Default:** 0  
**Minimum:** 0

**Used by:**
- `GET /v1/embeddings/{user}/{project}` - List embeddings
- `GET /v1/similars/{user}/{project}/{id}` - Similarity search
- `POST /v1/similars/{user}/{project}` - Raw vector search
- `GET /v1/projects/{user}` - List projects

**Example:**

```bash
# Get results 21-40 (page 2)
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs?limit=20&offset=20" \
  -H "Authorization: Bearer alice_api_key"

# Get results 41-60 (page 3)
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs?limit=20&offset=40" \
  -H "Authorization: Bearer alice_api_key"
```

**Pagination Formula:**
```
Page N: offset = (N - 1) * limit
```

---

## Similarity Search Parameters

### count

Number of similar documents to return. Alias for `limit` in similarity search endpoints.

**Type:** Integer  
**Default:** 10  
**Maximum:** 200  
**Minimum:** 1

**Used by:**
- `GET /v1/similars/{user}/{project}/{id}` - Similarity search
- `POST /v1/similars/{user}/{project}` - Raw vector search

**Example:**

```bash
# Find 5 most similar documents
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=5" \
  -H "Authorization: Bearer alice_api_key"
```

**Note:** `count` and `limit` can be used interchangeably in similarity endpoints.

---

### threshold

Minimum similarity score threshold. Only results with similarity >= threshold are returned.

**Type:** Float  
**Default:** 0.5  
**Range:** 0.0 to 1.0  
**Note:** 1.0 = most similar, 0.0 = least similar

**Used by:**
- `GET /v1/similars/{user}/{project}/{id}` - Similarity search
- `POST /v1/similars/{user}/{project}` - Raw vector search

**Example:**

```bash
# Only return highly similar documents (>= 0.8)
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?threshold=0.8" \
  -H "Authorization: Bearer alice_api_key"

# Return moderately similar documents (>= 0.6)
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?threshold=0.6" \
  -H "Authorization: Bearer alice_api_key"
```

**Threshold Guidelines:**

| Threshold | Interpretation | Use Case |
|-----------|----------------|----------|
| 0.95-1.0 | Near-identical | Duplicate detection |
| 0.85-0.95 | Very similar | Finding closely related documents |
| 0.7-0.85 | Moderately similar | Broad similarity search |
| 0.5-0.7 | Loosely similar | Exploratory search |
| 0.0-0.5 | Weakly similar | Generally not useful |

---

## Metadata Filtering Parameters

### metadata_path

JSON path to a metadata field for filtering results. Must be used together with `metadata_value`.

**Type:** String  
**Format:** JSON path notation (e.g., `"author"`, `"author.name"`, `"publication.year"`)

**Used by:**
- `GET /v1/similars/{user}/{project}/{id}` - Similarity search
- `POST /v1/similars/{user}/{project}` - Raw vector search

**Path Notation:**
- **Simple field:** `author`
- **Nested field:** `author.name`
- **Deeply nested:** `publication.journal.name`

**Example:**

```bash
# Simple field path
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=author&metadata_value=John%20Doe"

# Nested field path
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=author.id&metadata_value=A0083"
```

---

### metadata_value

Metadata value to exclude from similarity search results. Must be used together with `metadata_path`.

**Type:** String  
**Behavior:** Negative matching (excludes documents where field equals this value)

**Used by:**
- `GET /v1/similars/{user}/{project}/{id}` - Similarity search
- `POST /v1/similars/{user}/{project}` - Raw vector search

**Example:**

```bash
# Exclude documents from the same author
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=author&metadata_value=Alice%20Doe" \
  -H "Authorization: Bearer alice_api_key"

# Exclude documents from the same category
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?metadata_path=category&metadata_value=biology" \
  -H "Authorization: Bearer alice_api_key"
```

**URL Encoding:**

Always URL-encode metadata values that contain special characters:

```bash
# Value: "John Doe" → "John%20Doe"
# Value: "2024-01-15" → "2024-01-15" (no special chars)
# Value: "author@example.com" → "author%40example.com"
```

---

## Parameter Combinations

### Pagination with Filtering

Combine limit, offset, and threshold:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20&offset=40&threshold=0.7" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Similarity Search with Metadata Filtering

Combine threshold and metadata filtering:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=10&threshold=0.8&metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Complete Query with All Parameters

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=50&offset=0&threshold=0.75&metadata_path=category&metadata_value=biology" \
  -H "Authorization: Bearer alice_api_key"
```

---

## Endpoint-Specific Parameters

### List Embeddings

**Endpoint:** `GET /v1/embeddings/{user}/{project}`

**Supported Parameters:**
- `limit` (default: 10, max: 200)
- `offset` (default: 0)

**Example:**

```bash
curl -X GET "https://api.example.com/v1/embeddings/alice/research-docs?limit=100&offset=200" \
  -H "Authorization: Bearer alice_api_key"
```

---

### List Projects

**Endpoint:** `GET /v1/projects/{user}`

**Supported Parameters:**
- `limit` (default: 10, max: 200)
- `offset` (default: 0)

**Example:**

```bash
curl -X GET "https://api.example.com/v1/projects/alice?limit=50&offset=0" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Similarity Search (GET)

**Endpoint:** `GET /v1/similars/{user}/{project}/{id}`

**Supported Parameters:**
- `count` / `limit` (default: 10, max: 200)
- `offset` (default: 0)
- `threshold` (default: 0.5, range: 0.0-1.0)
- `metadata_path` (optional, requires `metadata_value`)
- `metadata_value` (optional, requires `metadata_path`)

**Example:**

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-docs/doc123?count=20&threshold=0.7&metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer alice_api_key"
```

---

### Similarity Search (POST)

**Endpoint:** `POST /v1/similars/{user}/{project}`

**Supported Parameters:** Same as GET similarity search

**Example:**

```bash
curl -X POST "https://api.example.com/v1/similars/alice/research-docs?count=10&threshold=0.8" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [-0.020850, 0.018522, 0.053270, ...]
  }'
```

---

## Parameter Validation

### Invalid Parameter Values

**Error Response:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "limit must be between 1 and 200"
}
```

**Common Validation Errors:**

- `limit` exceeds maximum (200)
- `limit` less than minimum (1)
- `offset` is negative
- `threshold` outside range 0.0-1.0
- `metadata_path` without `metadata_value`
- `metadata_value` without `metadata_path`

---

## Best Practices

### Pagination

**Do:**
- Use consistent `limit` values across pages
- Start with `offset=0` for the first page
- Increment `offset` by `limit` for each subsequent page

**Example pagination logic:**

```python
limit = 20
page = 1

# Page 1
offset = (page - 1) * limit  # 0
url = f"/v1/embeddings/alice/research?limit={limit}&offset={offset}"

# Page 2
page = 2
offset = (page - 1) * limit  # 20
url = f"/v1/embeddings/alice/research?limit={limit}&offset={offset}"

# Page 3
page = 3
offset = (page - 1) * limit  # 40
url = f"/v1/embeddings/alice/research?limit={limit}&offset={offset}"
```

---

### Similarity Search

**Do:**
- Use higher thresholds (0.7-0.9) for focused searches
- Use lower thresholds (0.5-0.7) for exploratory searches
- Combine with `count` to limit result size
- Use metadata filtering to exclude unwanted results

**Don't:**
- Request more results than needed (affects performance)
- Use very low thresholds (<0.5) unless necessary

---

### Metadata Filtering

**Do:**
- URL-encode metadata values with special characters
- Use specific field paths for nested metadata
- Test metadata paths with sample queries first

**Example:**

```bash
# Good: URL-encoded, specific path
metadata_path=author.id&metadata_value=A0083

# Good: Simple field
metadata_path=category&metadata_value=biology

# Bad: Not URL-encoded
metadata_path=author name&metadata_value=John Doe

# Good: URL-encoded
metadata_path=author%20name&metadata_value=John%20Doe
```

---

## Response Formats

### Paginated Response

Endpoints that support pagination typically return:

```json
{
  "items": [...],
  "total_count": 500,
  "limit": 20,
  "offset": 40,
  "has_more": true
}
```

**Fields:**
- `items`: Array of results
- `total_count`: Total number of items available
- `limit`: Number of items requested per page
- `offset`: Current offset
- `has_more`: Boolean indicating if more results exist

---

### Similarity Response

Similarity endpoints return:

```json
{
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
    }
  ]
}
```

**Notes:**
- Results are ordered by similarity (highest first)
- Only results >= `threshold` are included
- Maximum of `count`/`limit` results returned
- Filtered by `metadata_path`/`metadata_value` if specified

---

## Related Documentation

- [Embeddings](endpoints/embeddings/) - Embedding storage and retrieval
- [Similars](endpoints/similars/) - Similarity search details
- [Projects](endpoints/projects/) - Project management
- [Error Handling](error-handling/) - Error responses
