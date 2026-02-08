---
title: "RAG Workflow Guide"
weight: 1
---

# Complete RAG Workflow Guide

This guide demonstrates a complete Retrieval Augmented Generation (RAG) workflow using dhamps-vdb as your vector database.

## Overview

A typical RAG workflow involves:
1. Generate embeddings from your text content (using an external LLM service)
2. Upload embeddings to dhamps-vdb
3. Search for similar documents based on a query
4. Retrieve the relevant context
5. Use the context with an LLM to generate responses

## Prerequisites

- Access to dhamps-vdb API with a valid API key
- An external LLM service for generating embeddings (e.g., OpenAI, Cohere)
- Text content you want to process

## Step 1: Generate Embeddings Externally

First, use your chosen LLM service to generate embeddings for your text content. Here's an example using OpenAI's API:

```python
import openai

# Initialize OpenAI client
client = openai.OpenAI(api_key="your-openai-key")

# Generate embeddings for your text
text = "The quick brown fox jumps over the lazy dog"
response = client.embeddings.create(
    model="text-embedding-3-large",
    input=text,
    dimensions=3072
)

embedding_vector = response.data[0].embedding
```

## Step 2: Create LLM Service Instance

Before uploading embeddings, create an LLM service instance in dhamps-vdb that matches your embedding configuration:

```bash
curl -X PUT "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "description": "OpenAI large embedding model",
    "api_key_encrypted": "sk-proj-your-openai-key"
  }'
```

**Response:**
```json
{
  "instance_id": 123,
  "instance_handle": "my-openai",
  "owner": "alice",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "model": "text-embedding-3-large",
  "dimensions": 3072
}
```

## Step 3: Create a Project

Create a project to organize your embeddings:

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/my-documents" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "my-documents",
    "description": "Document embeddings for RAG",
    "instance_id": 123
  }'
```

## Step 4: Upload Embeddings to dhamps-vdb

Upload your pre-generated embeddings along with metadata and optional text content:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/my-documents" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc001",
      "instance_handle": "my-openai",
      "vector": [0.021, -0.015, 0.043, ...],
      "vector_dim": 3072,
      "text": "The quick brown fox jumps over the lazy dog",
      "metadata": {
        "source": "example.txt",
        "author": "Alice",
        "category": "animals"
      }
    }]
  }'
```

**Tip:** Upload multiple embeddings in batches for efficiency (see [Batch Operations Guide](./batch-operations.md)).

## Step 5: Search for Similar Documents

When you need to retrieve relevant context for a query:

### Option A: Search by Stored Document ID

If you already have a document in your database that represents your query:

```bash
curl -X GET "https://api.example.com/v1/similars/alice/my-documents/doc001?count=5&threshold=0.7" \
  -H "Authorization: Bearer alice_api_key"
```

### Option B: Search with Raw Query Embedding

Generate an embedding for your query and search without storing it:

```python
# Generate query embedding
query = "What animals are mentioned?"
query_response = client.embeddings.create(
    model="text-embedding-3-large",
    input=query,
    dimensions=3072
)
query_vector = query_response.data[0].embedding
```

```bash
curl -X POST "https://api.example.com/v1/similars/alice/my-documents?count=5&threshold=0.7" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.032, -0.018, 0.056, ...]
  }'
```

**Response:**
```json
{
  "user_handle": "alice",
  "project_handle": "my-documents",
  "results": [
    {
      "id": "doc001",
      "similarity": 0.95
    },
    {
      "id": "doc042",
      "similarity": 0.87
    },
    {
      "id": "doc103",
      "similarity": 0.82
    }
  ]
}
```

## Step 6: Retrieve Context Documents

Retrieve the full content and metadata for the most similar documents:

```bash
curl -X GET "https://api.example.com/v1/embeddings/alice/my-documents/doc001" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**
```json
{
  "text_id": "doc001",
  "text": "The quick brown fox jumps over the lazy dog",
  "metadata": {
    "source": "example.txt",
    "author": "Alice",
    "category": "animals"
  },
  "vector_dim": 3072
}
```

## Step 7: Use Context with LLM

Combine the retrieved context with your original query to generate an informed response:

```python
# Collect context from similar documents
context_docs = []
for result in similarity_results['results'][:3]:
    doc = get_document(result['id'])  # Your function to fetch document
    context_docs.append(doc['text'])

# Build context string
context = "\n\n".join(context_docs)

# Generate response with context
response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "system", "content": "Answer based on the provided context."},
        {"role": "user", "content": f"Context:\n{context}\n\nQuestion: {query}"}
    ]
)

answer = response.choices[0].message.content
```

## Complete Python Example

Here's a complete example combining all steps:

```python
import openai
import requests

# Configuration
DHAMPS_API = "https://api.example.com"
DHAMPS_KEY = "your-dhamps-api-key"
OPENAI_KEY = "your-openai-key"

# Initialize OpenAI
client = openai.OpenAI(api_key=OPENAI_KEY)

def embed_and_store(text_id, text, metadata=None):
    """Generate embedding and store in dhamps-vdb"""
    # Generate embedding
    response = client.embeddings.create(
        model="text-embedding-3-large",
        input=text,
        dimensions=3072
    )
    vector = response.data[0].embedding
    
    # Upload to dhamps-vdb
    requests.post(
        f"{DHAMPS_API}/v1/embeddings/alice/my-documents",
        headers={
            "Authorization": f"Bearer {DHAMPS_KEY}",
            "Content-Type": "application/json"
        },
        json={
            "embeddings": [{
                "text_id": text_id,
                "instance_handle": "my-openai",
                "vector": vector,
                "vector_dim": 3072,
                "text": text,
                "metadata": metadata or {}
            }]
        }
    )

def search_similar(query, count=5):
    """Search for similar documents using query text"""
    # Generate query embedding
    response = client.embeddings.create(
        model="text-embedding-3-large",
        input=query,
        dimensions=3072
    )
    query_vector = response.data[0].embedding
    
    # Search in dhamps-vdb
    result = requests.post(
        f"{DHAMPS_API}/v1/similars/alice/my-documents?count={count}",
        headers={
            "Authorization": f"Bearer {DHAMPS_KEY}",
            "Content-Type": "application/json"
        },
        json={"vector": query_vector}
    )
    return result.json()['results']

def retrieve_context(doc_ids):
    """Retrieve full document content"""
    docs = []
    for doc_id in doc_ids:
        response = requests.get(
            f"{DHAMPS_API}/v1/embeddings/alice/my-documents/{doc_id}",
            headers={"Authorization": f"Bearer {DHAMPS_KEY}"}
        )
        docs.append(response.json())
    return docs

def rag_query(query):
    """Complete RAG workflow"""
    # Search for similar documents
    similar = search_similar(query, count=3)
    
    # Retrieve context
    context_docs = retrieve_context([r['id'] for r in similar])
    context = "\n\n".join([doc['text'] for doc in context_docs])
    
    # Generate answer with LLM
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": "Answer based on the provided context."},
            {"role": "user", "content": f"Context:\n{context}\n\nQuestion: {query}"}
        ]
    )
    
    return response.choices[0].message.content

# Usage
embed_and_store("doc001", "The quick brown fox jumps over the lazy dog", 
                {"category": "animals"})
answer = rag_query("What animals are mentioned?")
print(answer)
```

## Best Practices

1. **Batch Upload**: Upload embeddings in batches of 100-1000 for better performance
2. **Use Metadata**: Include rich metadata for better filtering and organization
3. **Set Thresholds**: Use similarity thresholds (e.g., 0.7) to filter low-quality matches
4. **Cache Embeddings**: Cache generated embeddings to avoid redundant API calls
5. **Monitor Dimensions**: Ensure all embeddings use consistent dimensions (3072 for text-embedding-3-large)

## Advanced Features

### Metadata Filtering

Exclude certain documents from search results using metadata filters:

```bash
# Exclude documents from the same author as the query
curl -X GET "https://api.example.com/v1/similars/alice/my-documents/doc001?metadata_path=author&metadata_value=Alice" \
  -H "Authorization: Bearer alice_api_key"
```

See the [Metadata Filtering Guide](./metadata-filtering.md) for more details.

### Metadata Validation

Enforce consistent metadata structure using JSON Schema validation:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-documents" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"category\":{\"type\":\"string\"}},\"required\":[\"author\"]}"
  }'
```

See the [Metadata Validation Guide](./metadata-validation.md) for more details.

## Related Documentation

- [Batch Operations Guide](./batch-operations.md) - Efficiently upload large datasets
- [Metadata Filtering Guide](./metadata-filtering.md) - Advanced search filtering
- [Metadata Validation Guide](./metadata-validation.md) - Schema validation
- [Instance Management Guide](./instance-management.md) - Managing LLM service instances

## Troubleshooting

### Dimension Mismatch Error

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch"
}
```

**Solution**: Ensure the `vector_dim` field matches the dimensions configured in your LLM service instance.

### No Similar Results

If searches return no results, try:
- Lowering the similarity threshold (e.g., from 0.8 to 0.5)
- Increasing the count parameter
- Verifying embeddings are uploaded correctly
- Checking that query embeddings use the same model and dimensions
