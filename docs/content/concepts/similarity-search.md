---
title: "Similarity Search"
weight: 6
---

# Similarity Search

Find documents with similar semantic meaning using vector similarity.

## How it Works

Similarity search compares embedding vectors using cosine distance:

1. **Query vector**: Either from stored embedding or raw vector
2. **Comparison**: Calculate cosine similarity with all project embeddings
3. **Filtering**: Apply threshold and metadata filters
4. **Ranking**: Sort by similarity score (highest first)
5. **Return**: Top N most similar documents

## Search Methods

### Stored Document Search (GET)

Find documents similar to an already-stored embedding:

```bash
GET /v1/similars/alice/research/doc1?count=10&threshold=0.7
```

**Use cases:**
- Find related documents
- Discover similar passages
- Identify duplicates

### Raw Vector Search (POST)

Search using a new embedding without storing it:

```bash
POST /v1/similars/alice/research?count=10&threshold=0.7

{
  "vector": [0.023, -0.015, ..., 0.042]
}
```

**Use cases:**
- Query without saving
- Test embeddings
- Real-time search

## Query Parameters

### count

Number of similar documents to return.

- **Type**: Integer
- **Range**: 1-200
- **Default**: 10

```bash
GET /v1/similars/alice/project/doc1?count=5
```

### threshold

Minimum similarity score (0-1).

- **Type**: Float
- **Range**: 0.0-1.0
- **Default**: 0.5
- **Meaning**: 1.0 = identical, 0.0 = unrelated

```bash
GET /v1/similars/alice/project/doc1?threshold=0.8
```

### limit

Maximum number of results (same as count).

- **Type**: Integer
- **Range**: 1-200
- **Default**: 10

### offset

Skip first N results (pagination).

- **Type**: Integer
- **Minimum**: 0
- **Default**: 0

```bash
# First page
GET /v1/similars/alice/project/doc1?limit=10&offset=0

# Second page
GET /v1/similars/alice/project/doc1?limit=10&offset=10
```

### metadata_path

JSON path to metadata field for filtering.

- **Type**: String
- **Purpose**: Specify metadata field to filter
- **Must be used with**: metadata_value

```bash
?metadata_path=author
```

### metadata_value

Value to **exclude** from results.

- **Type**: String
- **Purpose**: Exclude documents matching this value
- **Must be used with**: metadata_path

```bash
?metadata_path=author&metadata_value=Shakespeare
```

**Important**: Excludes matches, doesn't include them.

## Similarity Scores

### Cosine Similarity

dhamps-vdb uses cosine similarity:

```
similarity = 1 - cosine_distance
```

**Score ranges:**
- **1.0**: Identical vectors
- **0.9-1.0**: Very similar
- **0.7-0.9**: Similar
- **0.5-0.7**: Somewhat similar
- **<0.5**: Not similar

### Interpreting Scores

Typical thresholds:

- **0.9+**: Duplicates or near-duplicates
- **0.8+**: Strong semantic similarity
- **0.7+**: Related topics
- **0.5-0.7**: Weak relation
- **<0.5**: Unrelated

Optimal threshold depends on your use case and model.

## Metadata Filtering

### Exclude by Field

Exclude documents where metadata field matches value:

```bash
# Exclude documents from same author
GET /v1/similars/alice/lit-study/hamlet-act1?metadata_path=author&metadata_value=Shakespeare
```

**Result**: Returns similar documents, excluding those with `metadata.author == "Shakespeare"`.

### Nested Fields

Use dot notation for nested metadata:

```bash
# Exclude documents from same author.name
GET /v1/similars/alice/project/doc1?metadata_path=author.name&metadata_value=John%20Doe
```

### Common Patterns

**Exclude same work:**
```bash
?metadata_path=title&metadata_value=Hamlet
```

**Exclude same source:**
```bash
?metadata_path=source_id&metadata_value=corpus-A
```

**Exclude same category:**
```bash
?metadata_path=category&metadata_value=tutorial
```

See [Metadata Filtering Guide](../guides/metadata-filtering/) for details.

## Response Format

```json
{
  "user_handle": "alice",
  "project_handle": "research",
  "results": [
    {
      "id": "doc2",
      "similarity": 0.95
    },
    {
      "id": "doc5",
      "similarity": 0.87
    },
    {
      "id": "doc8",
      "similarity": 0.82
    }
  ]
}
```

**Fields:**
- **user_handle**: Project owner
- **project_handle**: Project identifier
- **results**: Array of similar documents
  - **id**: Document text_id
  - **similarity**: Similarity score (0-1)

Results are sorted by similarity (highest first).

## Performance

### Query Speed

Typical performance:
- **<10K embeddings**: <10ms
- **10K-100K embeddings**: 10-50ms
- **100K-1M embeddings**: 50-200ms
- **>1M embeddings**: 200-1000ms

### Optimization

**HNSW Index:**
- Faster queries than IVFFlat
- Better recall
- Larger index size

**Query optimization:**
- Use appropriate threshold (higher = fewer results)
- Limit result count (lower = faster)
- Consider dimension reduction for large projects

### Scaling

For large datasets:
- Monitor query performance
- Consider read replicas
- Use connection pooling
- Cache frequent queries (application level)

## Common Use Cases

### RAG Workflow

Retrieval Augmented Generation:

```bash
# 1. User query
query="What is machine learning?"

# 2. Generate query embedding (external)
query_vector=[...]

# 3. Find similar documents
POST /v1/similars/alice/knowledge-base?count=5&threshold=0.7
{"vector": $query_vector}

# 4. Retrieve full text for top results
for each result:
  GET /v1/embeddings/alice/knowledge-base/$result_id

# 5. Send context to LLM for generation
```

### Duplicate Detection

Find near-duplicate documents:

```bash
# High threshold for duplicates
GET /v1/similars/alice/corpus/doc1?count=10&threshold=0.95
```

Documents with similarity > 0.95 are likely duplicates.

### Content Discovery

Find related content:

```bash
# Moderate threshold for recommendations
GET /v1/similars/alice/articles/article1?count=10&threshold=0.7&metadata_path=article_id&metadata_value=article1
```

Excludes the source article itself.

### Topic Clustering

Find documents on similar topics:

```bash
# For each document, find similar ones
for doc in documents:
  GET /v1/similars/alice/corpus/$doc?count=20&threshold=0.8
```

Group documents by similarity for clustering.

## Dimension Consistency

### Automatic Filtering

Similarity queries only compare embeddings with matching dimensions:

```
Project embeddings:
  - doc1: 3072 dimensions
  - doc2: 3072 dimensions
  - doc3: 1536 dimensions (different model)

Query for doc1 similars:
  → Only compares with doc2
  → Ignores doc3 (dimension mismatch)
```

### Multiple Instances

Projects can have embeddings from multiple instances (if dimensions match):

```json
{
  "text_id": "doc1",
  "instance_handle": "openai-large",
  "vector_dim": 3072
}

{
  "text_id": "doc2",
  "instance_handle": "custom-model",
  "vector_dim": 3072
}
```

Both searchable together (same dimensions).

## Access Control

### Authentication

Similarity search respects project access control:

**Owner**: Full access
**Editor**: Can search (read permission)
**Reader**: Can search (read permission)
**Public** (if public_read=true): Can search (no auth required)

### Public Projects

Public projects allow unauthenticated similarity search:

```bash
# No Authorization header needed
GET /v1/similars/alice/public-project/doc1?count=10
```

See [Public Projects Guide](../guides/public-projects/).

## Limitations

### Current Constraints

- **No cross-project search**: Similarity search is per-project only
- **No filtering by multiple metadata fields**: One field at a time
- **No custom distance metrics**: Cosine similarity only
- **No approximate search tuning**: Uses default HNSW parameters

### Workarounds

**Cross-project search:**
- Query each project separately
- Merge results in application

**Multiple metadata filters:**
- Filter by one field in query
- Apply additional filters in application

## Troubleshooting

### No Results Returned

**Possible causes:**
- Threshold too high
- No embeddings in project
- Dimension mismatch
- All results filtered by metadata

**Solutions:**
- Lower threshold (try 0.5)
- Verify embeddings exist
- Check dimensions match
- Remove metadata filter

### Unexpected Results

**Possible causes:**
- Threshold too low
- Poor quality embeddings
- Incorrect model used
- Metadata filter excluding desired results

**Solutions:**
- Increase threshold
- Regenerate embeddings
- Verify correct model/dimensions
- Adjust metadata filter

### Slow Queries

**Possible causes:**
- Large dataset (>100K embeddings)
- No vector index
- High result count
- Complex metadata filtering

**Solutions:**
- Reduce result count
- Check index exists
- Optimize database
- Use read replicas

## Next Steps

- [Learn about metadata filtering](../guides/metadata-filtering/)
- [Understand RAG workflows](../guides/rag-workflow/)
- [Explore embeddings](embeddings/)
