---
title: "Batch Operations Guide"
weight: 7
---

# Batch Operations Guide

This guide explains how to efficiently upload multiple embeddings, manage large datasets, and implement best practices for batch operations in dhamps-vdb.

## Overview

For production workloads and large datasets, efficient batch operations are essential. This guide covers:
- Uploading multiple embeddings in a single request
- Pagination strategies for large result sets
- Best practices for performance and reliability
- Error handling in batch operations

## Batch Upload Basics

### Single Request with Multiple Embeddings

Upload multiple embeddings in one API call using the `embeddings` array:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [
      {
        "text_id": "doc001",
        "instance_handle": "openai-large",
        "vector": [0.1, 0.2, 0.3, ...],
        "vector_dim": 3072,
        "text": "First document content",
        "metadata": {"category": "science"}
      },
      {
        "text_id": "doc002",
        "instance_handle": "openai-large",
        "vector": [0.11, 0.21, 0.31, ...],
        "vector_dim": 3072,
        "text": "Second document content",
        "metadata": {"category": "history"}
      },
      {
        "text_id": "doc003",
        "instance_handle": "openai-large",
        "vector": [0.12, 0.22, 0.32, ...],
        "vector_dim": 3072,
        "text": "Third document content",
        "metadata": {"category": "literature"}
      }
    ]
  }'
```

**Response:**
```json
{
  "message": "Embeddings uploaded successfully",
  "count": 3
}
```

## Optimal Batch Sizes

### Recommended Batch Sizes

Based on typical embedding dimensions and network constraints:

| Embedding Dimensions | Recommended Batch Size | Maximum Batch Size |
|---------------------|------------------------|-------------------|
| 384 (small models) | 500-1000 | 2000 |
| 768 (BERT-base) | 300-500 | 1000 |
| 1536 (OpenAI small) | 100-300 | 500 |
| 3072 (OpenAI large) | 50-100 | 200 |

**Factors to consider:**
- Network bandwidth and latency
- API gateway timeout limits
- Database transaction size
- Memory constraints
- Client-side serialization limits

### Finding Your Optimal Batch Size

Test different batch sizes to find the sweet spot:

```python
import time
import requests

def test_batch_size(embeddings, batch_size):
    """Test upload performance with given batch size"""
    start_time = time.time()
    
    for i in range(0, len(embeddings), batch_size):
        batch = embeddings[i:i+batch_size]
        response = requests.post(
            "https://api.example.com/v1/embeddings/alice/my-project",
            headers={
                "Authorization": "Bearer alice_api_key",
                "Content-Type": "application/json"
            },
            json={"embeddings": batch}
        )
        response.raise_for_status()
    
    elapsed = time.time() - start_time
    throughput = len(embeddings) / elapsed
    print(f"Batch size {batch_size}: {throughput:.1f} embeddings/sec")

# Test different batch sizes
for size in [50, 100, 200, 500]:
    test_batch_size(my_embeddings, size)
```

## Pagination for Large Datasets

### Retrieving All Embeddings with Pagination

Use `limit` and `offset` parameters to paginate through large result sets:

```bash
# Get first page (embeddings 0-99)
curl -X GET "https://api.example.com/v1/embeddings/alice/my-project?limit=100&offset=0" \
  -H "Authorization: Bearer alice_api_key"

# Get second page (embeddings 100-199)
curl -X GET "https://api.example.com/v1/embeddings/alice/my-project?limit=100&offset=100" \
  -H "Authorization: Bearer alice_api_key"

# Get third page (embeddings 200-299)
curl -X GET "https://api.example.com/v1/embeddings/alice/my-project?limit=100&offset=200" \
  -H "Authorization: Bearer alice_api_key"
```

### Pagination Best Practices

**Default Values:**
- `limit`: 10 (if not specified)
- `offset`: 0 (if not specified)
- Maximum `limit`: 200

**Example: Download Entire Project**

```python
import requests

def download_all_embeddings(owner, project):
    """Download all embeddings from a project"""
    all_embeddings = []
    offset = 0
    limit = 100
    
    while True:
        response = requests.get(
            f"https://api.example.com/v1/embeddings/{owner}/{project}",
            headers={"Authorization": "Bearer api_key"},
            params={"limit": limit, "offset": offset}
        )
        response.raise_for_status()
        
        batch = response.json()['embeddings']
        if not batch:
            break  # No more results
        
        all_embeddings.extend(batch)
        offset += len(batch)
        
        print(f"Downloaded {len(all_embeddings)} embeddings...")
    
    return all_embeddings

# Usage
embeddings = download_all_embeddings("alice", "my-project")
print(f"Total: {len(embeddings)} embeddings")
```

## Efficient Batch Upload Strategies

### Strategy 1: Simple Sequential Upload

Good for small to medium datasets (< 10,000 embeddings):

```python
def upload_sequential(embeddings, batch_size=100):
    """Upload embeddings sequentially in batches"""
    for i in range(0, len(embeddings), batch_size):
        batch = embeddings[i:i+batch_size]
        response = requests.post(
            "https://api.example.com/v1/embeddings/alice/my-project",
            headers={
                "Authorization": "Bearer alice_api_key",
                "Content-Type": "application/json"
            },
            json={"embeddings": batch}
        )
        response.raise_for_status()
        print(f"Uploaded batch {i//batch_size + 1}, total: {i+len(batch)}")
```

### Strategy 2: Parallel Upload with Threading

Good for larger datasets with stable network:

```python
import concurrent.futures
import requests

def upload_batch(batch, batch_num):
    """Upload a single batch"""
    try:
        response = requests.post(
            "https://api.example.com/v1/embeddings/alice/my-project",
            headers={
                "Authorization": "Bearer alice_api_key",
                "Content-Type": "application/json"
            },
            json={"embeddings": batch},
            timeout=60
        )
        response.raise_for_status()
        return batch_num, True, None
    except Exception as e:
        return batch_num, False, str(e)

def upload_parallel(embeddings, batch_size=100, max_workers=4):
    """Upload embeddings in parallel"""
    batches = [
        embeddings[i:i+batch_size]
        for i in range(0, len(embeddings), batch_size)
    ]
    
    failed = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {
            executor.submit(upload_batch, batch, i): i
            for i, batch in enumerate(batches)
        }
        
        for future in concurrent.futures.as_completed(futures):
            batch_num, success, error = future.result()
            if success:
                print(f"✓ Batch {batch_num+1}/{len(batches)} uploaded")
            else:
                print(f"✗ Batch {batch_num+1} failed: {error}")
                failed.append((batch_num, batches[batch_num]))
    
    return failed

# Usage
failed_batches = upload_parallel(my_embeddings, batch_size=100, max_workers=4)
if failed_batches:
    print(f"Failed batches: {len(failed_batches)}")
```

### Strategy 3: Retry with Exponential Backoff

Robust strategy for unreliable networks:

```python
import time
import random

def upload_with_retry(batch, max_retries=3):
    """Upload batch with exponential backoff retry"""
    for attempt in range(max_retries):
        try:
            response = requests.post(
                "https://api.example.com/v1/embeddings/alice/my-project",
                headers={
                    "Authorization": "Bearer alice_api_key",
                    "Content-Type": "application/json"
                },
                json={"embeddings": batch},
                timeout=60
            )
            response.raise_for_status()
            return True
        except requests.exceptions.RequestException as e:
            if attempt < max_retries - 1:
                wait = (2 ** attempt) + random.uniform(0, 1)
                print(f"Retry attempt {attempt+1} after {wait:.1f}s: {e}")
                time.sleep(wait)
            else:
                print(f"Failed after {max_retries} attempts: {e}")
                return False

def upload_robust(embeddings, batch_size=100):
    """Upload with robust error handling"""
    failed = []
    for i in range(0, len(embeddings), batch_size):
        batch = embeddings[i:i+batch_size]
        if not upload_with_retry(batch):
            failed.append((i, batch))
        else:
            print(f"✓ Uploaded {i+batch_size}/{len(embeddings)}")
    
    return failed
```

## Progress Tracking and Resumability

### Checkpoint-Based Upload

For very large datasets, implement checkpointing to resume after failures:

```python
import json
import os

class CheckpointUploader:
    def __init__(self, checkpoint_file="upload_progress.json"):
        self.checkpoint_file = checkpoint_file
        self.progress = self.load_progress()
    
    def load_progress(self):
        """Load upload progress from checkpoint file"""
        if os.path.exists(self.checkpoint_file):
            with open(self.checkpoint_file, 'r') as f:
                return json.load(f)
        return {"uploaded_count": 0, "failed_batches": []}
    
    def save_progress(self):
        """Save current progress"""
        with open(self.checkpoint_file, 'w') as f:
            json.dump(self.progress, f)
    
    def upload(self, embeddings, batch_size=100):
        """Upload with checkpointing"""
        start_idx = self.progress["uploaded_count"]
        
        for i in range(start_idx, len(embeddings), batch_size):
            batch = embeddings[i:i+batch_size]
            
            try:
                response = requests.post(
                    "https://api.example.com/v1/embeddings/alice/my-project",
                    headers={
                        "Authorization": "Bearer alice_api_key",
                        "Content-Type": "application/json"
                    },
                    json={"embeddings": batch},
                    timeout=60
                )
                response.raise_for_status()
                
                self.progress["uploaded_count"] = i + len(batch)
                self.save_progress()
                print(f"✓ Progress: {self.progress['uploaded_count']}/{len(embeddings)}")
                
            except Exception as e:
                print(f"✗ Failed at index {i}: {e}")
                self.progress["failed_batches"].append(i)
                self.save_progress()
                raise

# Usage
uploader = CheckpointUploader()
try:
    uploader.upload(my_embeddings, batch_size=100)
    print("Upload complete!")
except:
    print("Upload interrupted. Run again to resume.")
```

## Error Handling

### Validation Errors

If any embedding in a batch fails validation, the entire batch is rejected:

```bash
curl -X POST "https://api.example.com/v1/embeddings/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [
      {
        "text_id": "doc001",
        "instance_handle": "openai-large",
        "vector": [0.1, 0.2, 0.3],
        "vector_dim": 3072,
        "metadata": {"author": "Alice"}
      },
      {
        "text_id": "doc002",
        "instance_handle": "openai-large",
        "vector": [0.1, 0.2],
        "vector_dim": 3072,
        "metadata": {"author": "Bob"}
      }
    ]
  }'
```

**Error Response:**
```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector length mismatch for text_id 'doc002': actual vector has 2 elements but vector_dim declares 3072"
}
```

**Solution:** Validate all embeddings before batching, or handle errors and retry failed items.

### Pre-Upload Validation

```python
def validate_embeddings(embeddings, expected_dim):
    """Validate embeddings before upload"""
    errors = []
    
    for i, emb in enumerate(embeddings):
        # Check vector length
        if len(emb['vector']) != expected_dim:
            errors.append(f"Index {i} (text_id: {emb['text_id']}): "
                        f"vector length {len(emb['vector'])} != {expected_dim}")
        
        # Check declared dimension
        if emb.get('vector_dim') != expected_dim:
            errors.append(f"Index {i} (text_id: {emb['text_id']}): "
                        f"vector_dim {emb.get('vector_dim')} != {expected_dim}")
        
        # Check required fields
        if not emb.get('text_id'):
            errors.append(f"Index {i}: missing text_id")
        
        if not emb.get('instance_handle'):
            errors.append(f"Index {i}: missing instance_handle")
    
    return errors

# Usage
errors = validate_embeddings(my_embeddings, 3072)
if errors:
    print("Validation errors:")
    for error in errors:
        print(f"  - {error}")
else:
    print("All embeddings valid, proceeding with upload...")
```

## Performance Optimization Tips

### 1. Minimize Payload Size

Exclude unnecessary fields:

```python
# Include text only if needed for retrieval
embeddings_with_text = [
    {
        "text_id": doc_id,
        "instance_handle": "openai-large",
        "vector": vector,
        "vector_dim": 3072,
        "text": text,  # Include if needed
        "metadata": metadata
    }
    for doc_id, vector, text, metadata in documents
]

# Exclude text if not needed (smaller payload)
embeddings_without_text = [
    {
        "text_id": doc_id,
        "instance_handle": "openai-large",
        "vector": vector,
        "vector_dim": 3072,
        "metadata": metadata
    }
    for doc_id, vector, metadata in documents
]
```

### 2. Compress Requests

Use gzip compression for large payloads:

```python
import gzip
import json

def upload_compressed(embeddings):
    """Upload with gzip compression"""
    payload = json.dumps({"embeddings": embeddings})
    compressed = gzip.compress(payload.encode('utf-8'))
    
    response = requests.post(
        "https://api.example.com/v1/embeddings/alice/my-project",
        headers={
            "Authorization": "Bearer alice_api_key",
            "Content-Type": "application/json",
            "Content-Encoding": "gzip"
        },
        data=compressed
    )
    return response
```

### 3. Use Connection Pooling

Reuse HTTP connections for multiple requests:

```python
session = requests.Session()
session.headers.update({
    "Authorization": "Bearer alice_api_key",
    "Content-Type": "application/json"
})

for batch in batches:
    response = session.post(
        "https://api.example.com/v1/embeddings/alice/my-project",
        json={"embeddings": batch}
    )
    response.raise_for_status()
```

### 4. Monitor Upload Rate

Track and display upload progress:

```python
import time

class ProgressTracker:
    def __init__(self, total):
        self.total = total
        self.uploaded = 0
        self.start_time = time.time()
    
    def update(self, count):
        self.uploaded += count
        elapsed = time.time() - self.start_time
        rate = self.uploaded / elapsed if elapsed > 0 else 0
        percent = (self.uploaded / self.total) * 100
        eta = (self.total - self.uploaded) / rate if rate > 0 else 0
        
        print(f"\rProgress: {self.uploaded}/{self.total} ({percent:.1f}%) "
              f"Rate: {rate:.1f} emb/s ETA: {eta:.0f}s", end="")

# Usage
tracker = ProgressTracker(len(all_embeddings))
for batch in batches:
    upload_batch(batch)
    tracker.update(len(batch))
print()  # New line after completion
```

## Complete Example: Production-Grade Uploader

```python
import requests
import time
import json
import logging
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict, Tuple

class ProductionUploader:
    def __init__(self, api_base: str, api_key: str, 
                 owner: str, project: str):
        self.api_base = api_base
        self.api_key = api_key
        self.owner = owner
        self.project = project
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json"
        })
        
        logging.basicConfig(level=logging.INFO)
        self.logger = logging.getLogger(__name__)
    
    def upload_batch(self, batch: List[Dict], batch_num: int, 
                    max_retries: int = 3) -> Tuple[int, bool, str]:
        """Upload a single batch with retry logic"""
        url = f"{self.api_base}/v1/embeddings/{self.owner}/{self.project}"
        
        for attempt in range(max_retries):
            try:
                response = self.session.post(
                    url,
                    json={"embeddings": batch},
                    timeout=60
                )
                response.raise_for_status()
                return batch_num, True, ""
            except Exception as e:
                if attempt < max_retries - 1:
                    wait = 2 ** attempt
                    self.logger.warning(
                        f"Batch {batch_num} attempt {attempt+1} failed: {e}. "
                        f"Retrying in {wait}s..."
                    )
                    time.sleep(wait)
                else:
                    return batch_num, False, str(e)
    
    def upload(self, embeddings: List[Dict], batch_size: int = 100,
               max_workers: int = 4) -> Dict:
        """Upload embeddings with parallel processing and progress tracking"""
        batches = [
            embeddings[i:i+batch_size]
            for i in range(0, len(embeddings), batch_size)
        ]
        
        results = {
            "total": len(embeddings),
            "uploaded": 0,
            "failed": [],
            "start_time": time.time()
        }
        
        self.logger.info(f"Uploading {len(embeddings)} embeddings in "
                        f"{len(batches)} batches...")
        
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = {
                executor.submit(self.upload_batch, batch, i): i
                for i, batch in enumerate(batches)
            }
            
            for future in as_completed(futures):
                batch_num, success, error = future.result()
                
                if success:
                    results["uploaded"] += len(batches[batch_num])
                    percent = (results["uploaded"] / results["total"]) * 100
                    self.logger.info(
                        f"✓ Batch {batch_num+1}/{len(batches)} "
                        f"({percent:.1f}% complete)"
                    )
                else:
                    results["failed"].append({
                        "batch_num": batch_num,
                        "error": error,
                        "embeddings": batches[batch_num]
                    })
                    self.logger.error(f"✗ Batch {batch_num+1} failed: {error}")
        
        results["elapsed"] = time.time() - results["start_time"]
        results["rate"] = results["uploaded"] / results["elapsed"]
        
        self.logger.info(
            f"\nUpload complete: {results['uploaded']}/{results['total']} "
            f"in {results['elapsed']:.1f}s ({results['rate']:.1f} emb/s)"
        )
        
        if results["failed"]:
            self.logger.warning(f"Failed batches: {len(results['failed'])}")
        
        return results

# Usage
uploader = ProductionUploader(
    api_base="https://api.example.com",
    api_key="alice_api_key",
    owner="alice",
    project="my-project"
)

results = uploader.upload(
    embeddings=my_embeddings,
    batch_size=100,
    max_workers=4
)

# Save failed batches for retry
if results["failed"]:
    with open("failed_batches.json", "w") as f:
        json.dump(results["failed"], f)
```

## Best Practices Summary

1. **Batch Size**: Test to find optimal size (typically 50-500 depending on dimensions)
2. **Parallelism**: Use 2-8 parallel workers for large uploads
3. **Retry Logic**: Implement exponential backoff for network errors
4. **Validation**: Pre-validate embeddings before upload
5. **Progress Tracking**: Monitor upload rate and ETA
6. **Checkpointing**: Save progress for resumable uploads
7. **Error Logging**: Log all errors with sufficient context
8. **Connection Reuse**: Use session objects for connection pooling
9. **Compression**: Use gzip for large payloads
10. **Testing**: Test with small batches before full upload

## Related Documentation

- [RAG Workflow Guide](./rag-workflow.md) - Complete RAG implementation
- [Metadata Validation Guide](./metadata-validation.md) - Schema validation
- [Instance Management Guide](./instance-management.md) - Managing LLM instances

## Troubleshooting

### Timeout Errors

**Problem:** Requests timing out with large batches

**Solution:** Reduce batch size or increase timeout value

### Memory Issues

**Problem:** Out of memory when processing large datasets

**Solution:** Process embeddings in streaming fashion, don't load all into memory

### Rate Limiting

**Problem:** Getting rate limited by API

**Solution:** Reduce parallelism (max_workers) or add delays between requests
