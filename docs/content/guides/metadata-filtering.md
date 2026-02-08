---
title: "Metadata Filtering Guide"
weight: 6
---

# Metadata Filtering Guide

This guide explains how to use metadata filtering to exclude specific documents from similarity search results.

## Overview

When searching for similar documents, you may want to exclude results that share certain metadata values with your query. For example:
- Exclude documents from the same author when finding similar writing styles
- Filter out documents from the same source when finding related content
- Exclude documents with the same category when exploring diversity

embapi provides metadata filtering using query parameters that perform **negative matching** - they exclude documents where the metadata field matches the specified value.

## Query Parameters

Both similarity search endpoints support metadata filtering:

- `metadata_path`: The JSON path to the metadata field (e.g., `author`, `source.id`, `tags[0]`)
- `metadata_value`: The value to exclude from results

Both parameters must be used together. If you specify one without the other, the API returns an error.

## Basic Filtering Examples

### Exclude Documents by Author

Find similar documents but exclude those from the same author:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/literary-corpus/hamlet-soliloquy?count=10&metadata_path=author&metadata_value=William%20Shakespeare" \
  -H "Authorization: Bearer alice_api_key"
```

This returns similar documents, excluding any with `metadata.author == "William Shakespeare"`.

### Exclude Documents from Same Source

Find similar content from different sources:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/news-articles/article123?count=10&metadata_path=source&metadata_value=NYTimes" \
  -H "Authorization: Bearer alice_api_key"
```

This excludes any documents with `metadata.source == "NYTimes"`.

### Exclude by Category

Find documents in different categories:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/products/product456?count=10&metadata_path=category&metadata_value=electronics" \
  -H "Authorization: Bearer alice_api_key"
```

This excludes any documents with `metadata.category == "electronics"`.

## Filtering with Raw Embeddings

Metadata filtering also works when searching with raw embedding vectors:

```bash
curl -X POST "https://api.example.com/v1/similars/alice/literary-corpus?count=10&metadata_path=author&metadata_value=William%20Shakespeare" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.032, -0.018, 0.056, ...]
  }'
```

This searches using the provided vector but excludes documents where `metadata.author == "William Shakespeare"`.

## Nested Metadata Paths

For nested metadata objects, use dot notation:

### Example Metadata Structure

```json
{
  "author": {
    "name": "Jane Doe",
    "id": "author123",
    "affiliation": "University"
  },
  "publication": {
    "journal": "Science",
    "year": 2023
  }
}
```

### Filter by Nested Field

```bash
# Exclude documents from same author ID
curl -X GET "https://api.example.com/v1/similars/alice/papers/paper001?count=10&metadata_path=author.id&metadata_value=author123" \
  -H "Authorization: Bearer alice_api_key"

# Exclude documents from same journal
curl -X GET "https://api.example.com/v1/similars/alice/papers/paper001?count=10&metadata_path=publication.journal&metadata_value=Science" \
  -H "Authorization: Bearer alice_api_key"
```

## Combining with Other Parameters

Metadata filtering works seamlessly with other search parameters:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/documents/doc123?count=20&threshold=0.8&limit=10&offset=0&metadata_path=source_id&metadata_value=src_456" \
  -H "Authorization: Bearer alice_api_key"
```

Parameters:
- `count=20`: Consider top 20 similar documents
- `threshold=0.8`: Only include documents with similarity ≥ 0.8
- `limit=10`: Return at most 10 results
- `offset=0`: Start from first result (for pagination)
- `metadata_path=source_id`: Filter on this metadata field
- `metadata_value=src_456`: Exclude documents with this value

## Use Cases

### 1. Finding Similar Writing Styles Across Authors

When analyzing writing styles, you want similar texts from different authors:

```bash
# Upload documents with author metadata
curl -X POST "https://api.example.com/v1/embeddings/alice/writing-styles" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "tolstoy-passage-1",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, ...],
      "vector_dim": 3072,
      "text": "Happy families are all alike...",
      "metadata": {
        "author": "Leo Tolstoy",
        "work": "Anna Karenina",
        "language": "Russian"
      }
    }]
  }'

# Find similar writing styles from other authors
curl -X GET "https://api.example.com/v1/similars/alice/writing-styles/tolstoy-passage-1?count=10&metadata_path=author&metadata_value=Leo%20Tolstoy" \
  -H "Authorization: Bearer alice_api_key"
```

### 2. Cross-Source Content Discovery

Find related news articles from different sources:

```bash
# Search for similar content, excluding same source
curl -X GET "https://api.example.com/v1/similars/alice/news-corpus/nyt-article-456?count=15&metadata_path=source&metadata_value=New%20York%20Times" \
  -H "Authorization: Bearer alice_api_key"
```

This helps discover how different outlets cover similar topics.

### 3. Product Recommendations Across Categories

Find similar products in different categories:

```bash
# User is viewing a laptop
curl -X GET "https://api.example.com/v1/similars/alice/product-catalog/laptop-001?count=10&threshold=0.7&metadata_path=category&metadata_value=electronics" \
  -H "Authorization: Bearer alice_api_key"
```

This could recommend accessories, furniture (for home office), or other complementary items.

### 4. Research Paper Discovery

Find related papers from different research groups:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/research-papers/paper123?count=20&metadata_path=lab_id&metadata_value=lab_abc_001" \
  -H "Authorization: Bearer alice_api_key"
```

Helps researchers discover related work from other institutions.

### 5. Avoiding Duplicate Content

When building a diverse content feed, exclude items from the same collection:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/blog-posts/post789?count=5&metadata_path=collection_id&metadata_value=series_xyz" \
  -H "Authorization: Bearer alice_api_key"
```

### 6. Cross-Language Document Discovery

Find similar documents in other languages:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/multilingual-docs/doc_en_123?count=10&metadata_path=language&metadata_value=en" \
  -H "Authorization: Bearer alice_api_key"
```

This finds semantically similar documents in languages other than English.

## Working with Multiple Values

Currently, you can only filter by one metadata field at a time. To exclude multiple values, you need to:

1. **Make multiple requests** and merge results in your application
2. **Use more specific metadata fields** that combine multiple attributes
3. **Post-process results** on the client side

### Example: Excluding Multiple Authors

```python
import requests

def find_similar_excluding_authors(doc_id, exclude_authors):
    """Find similar docs excluding multiple authors"""
    all_results = []
    
    for author in exclude_authors:
        response = requests.get(
            f"https://api.example.com/v1/similars/alice/corpus/{doc_id}",
            headers={"Authorization": "Bearer alice_api_key"},
            params={
                "count": 20,
                "metadata_path": "author",
                "metadata_value": author
            }
        )
        results = response.json()['results']
        all_results.extend(results)
    
    # Deduplicate and sort by similarity
    seen = set()
    unique_results = []
    for r in sorted(all_results, key=lambda x: x['similarity'], reverse=True):
        if r['id'] not in seen:
            seen.add(r['id'])
            unique_results.append(r)
    
    return unique_results[:10]

# Usage
similar = find_similar_excluding_authors(
    "doc123",
    ["Author A", "Author B", "Author C"]
)
```

## Combining with Metadata Validation

For reliable filtering, combine with metadata schema validation:

```bash
# Step 1: Create project with metadata schema
curl -X POST "https://api.example.com/v1/projects/alice/validated-corpus" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "validated-corpus",
    "description": "Corpus with validated metadata",
    "instance_id": 123,
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"source_id\":{\"type\":\"string\"}},\"required\":[\"author\",\"source_id\"]}"
  }'

# Step 2: Upload embeddings with metadata
curl -X POST "https://api.example.com/v1/embeddings/alice/validated-corpus" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc001",
      "instance_handle": "openai-large",
      "vector": [0.1, 0.2, ...],
      "vector_dim": 3072,
      "metadata": {
        "author": "John Doe",
        "source_id": "source_123"
      }
    }]
  }'

# Step 3: Search with guaranteed metadata field existence
curl -X GET "https://api.example.com/v1/similars/alice/validated-corpus/doc001?count=10&metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer alice_api_key"
```

See the [Metadata Validation Guide](./metadata-validation.md) for more details.

## Understanding the Filter Logic

The metadata filter uses **negative matching**:

```
INCLUDE document IF:
  - document.similarity >= threshold
  AND
  - document.metadata[metadata_path] != metadata_value
```

**Important:** Documents without the specified metadata field are included (not filtered out).

### Example

Given this query:
```bash
?metadata_path=author&metadata_value=Alice
```

**Included:**
- Documents where `metadata.author == "Bob"`
- Documents where `metadata.author == "Charlie"`
- Documents without an `author` field in metadata

**Excluded:**
- Documents where `metadata.author == "Alice"`

## Performance Considerations

Metadata filtering is performed at the database level using efficient indexing:

1. **Vector similarity** is computed first
2. **Metadata filter** is applied to the similarity results
3. Results are sorted and limited

For best performance:
- Use indexed metadata fields when possible
- Keep metadata values relatively small (under 1KB per document)
- Consider using IDs instead of full names for filtering

## Error Handling

### Missing One Parameter

```bash
# Missing metadata_value
curl -X GET "https://api.example.com/v1/similars/alice/corpus/doc123?metadata_path=author" \
  -H "Authorization: Bearer alice_api_key"
```

**Error:**
```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata_path and metadata_value must be used together"
}
```

### Non-Existent Metadata Field

```bash
# Filtering on field that doesn't exist in documents
curl -X GET "https://api.example.com/v1/similars/alice/corpus/doc123?count=10&metadata_path=nonexistent_field&metadata_value=some_value" \
  -H "Authorization: Bearer alice_api_key"
```

**Result:** Returns all matching documents (since none have the field, none are excluded).

### URL Encoding

Remember to URL-encode metadata values with special characters:

```bash
# Correct: URL-encoded value
curl -X GET "https://api.example.com/v1/similars/alice/corpus/doc123?metadata_path=author&metadata_value=John%20Doe%20%26%20Jane%20Smith" \
  -H "Authorization: Bearer alice_api_key"

# Incorrect: Unencoded special characters
curl -X GET "https://api.example.com/v1/similars/alice/corpus/doc123?metadata_path=author&metadata_value=John Doe & Jane Smith" \
  -H "Authorization: Bearer alice_api_key"
```

## Complete Example

Here's a complete workflow demonstrating metadata filtering:

```bash
# 1. Create project
curl -X POST "https://api.example.com/v1/projects/alice/literature" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "literature",
    "instance_id": 123
  }'

# 2. Upload documents with metadata
curl -X POST "https://api.example.com/v1/embeddings/alice/literature" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [
      {
        "text_id": "tolstoy_1",
        "instance_handle": "openai-large",
        "vector": [0.1, 0.2, ...],
        "vector_dim": 3072,
        "text": "All happy families...",
        "metadata": {"author": "Tolstoy", "work": "Anna Karenina"}
      },
      {
        "text_id": "tolstoy_2",
        "instance_handle": "openai-large",
        "vector": [0.11, 0.21, ...],
        "vector_dim": 3072,
        "text": "It was the best of times...",
        "metadata": {"author": "Tolstoy", "work": "War and Peace"}
      },
      {
        "text_id": "dickens_1",
        "instance_handle": "openai-large",
        "vector": [0.12, 0.19, ...],
        "vector_dim": 3072,
        "text": "It was the age of wisdom...",
        "metadata": {"author": "Dickens", "work": "Tale of Two Cities"}
      }
    ]
  }'

# 3. Find similar to tolstoy_1, excluding Tolstoy's works
curl -X GET "https://api.example.com/v1/similars/alice/literature/tolstoy_1?count=10&metadata_path=author&metadata_value=Tolstoy" \
  -H "Authorization: Bearer alice_api_key"

# Result: Returns dickens_1, excludes tolstoy_2
```

## Related Documentation

- [RAG Workflow Guide](./rag-workflow.md) - Complete RAG implementation
- [Metadata Validation Guide](./metadata-validation.md) - Schema validation
- [Batch Operations Guide](./batch-operations.md) - Upload large datasets

## Troubleshooting

### No Results Returned

**Problem:** Filter excludes all results

**Solution:** 
- Verify the metadata field exists in your documents
- Check that the metadata value matches exactly (case-sensitive)
- Try without the filter to ensure there are similar documents

### Filter Not Working

**Problem:** Still seeing documents you want to exclude

**Solution:**
- Check URL encoding of the metadata value
- Verify the metadata path is correct (use dot notation for nested fields)
- Ensure both `metadata_path` and `metadata_value` are specified

### Want Positive Matching

**Problem:** Want to include only specific values, not exclude them

**Solution:** Currently, only negative matching (exclusion) is supported. For positive matching, retrieve all results and filter on the client side, or use multiple negative filters to exclude everything except your target values.
