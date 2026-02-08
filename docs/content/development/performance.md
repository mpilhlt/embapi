---
title: "Performance"
weight: 4
---

# Performance Optimization Guide

This guide covers performance optimization strategies for dhamps-vdb, including query optimization, indexing, caching, and performance testing.

## Query Optimization

### GetAllAccessibleInstances Query

**Problem:** The original implementation uses a LEFT JOIN with OR conditions, which can result in inefficient query execution.

**Current Implementation:**

```sql
SELECT instances.*, 
       COALESCE(instances_shared_with.role, 'owner') as role,
       (instances.owner = $1) as is_owner
FROM instances
LEFT JOIN instances_shared_with
  ON instances.instance_id = instances_shared_with.instance_id
WHERE instances.owner = $1
   OR instances_shared_with.user_handle = $1
ORDER BY instances.owner ASC, instances.instance_handle ASC 
LIMIT $2 OFFSET $3;
```

**Issue:** The query planner may struggle to use indexes effectively with LEFT JOIN combined with OR conditions in the WHERE clause.

### Recommended: UNION ALL Pattern

Use UNION ALL to separate owned instances from shared instances:

```sql
-- Get owned instances
SELECT instances.*, 
       'owner' as role, 
       true as is_owner
FROM instances
WHERE instances.owner = $1

UNION ALL

-- Get shared instances
SELECT instances.*, 
       instances_shared_with.role,
       false as is_owner
FROM instances
INNER JOIN instances_shared_with
  ON instances.instance_id = instances_shared_with.instance_id
WHERE instances_shared_with.user_handle = $1
  AND instances.owner != $1  -- Avoid duplicates

ORDER BY owner ASC, instance_handle ASC
LIMIT $2 OFFSET $3;
```

**Benefits:**

1. **Separate index scans**: Query planner can use different indexes for each UNION branch
2. **Owned instances**: Can use index on `(owner)`
3. **Shared instances**: Can use index on `(user_handle)` 
4. **Clearer execution plan**: Easier to understand and optimize
5. **Better performance**: Especially noticeable with large datasets

**Trade-offs:**

- Slightly more complex SQL
- Need to deduplicate if user somehow has instance both owned and shared (unlikely scenario)
- Both queries must have same column structure

### Implementation Example

**Before (queries.sql):**

```sql
-- name: GetAllAccessibleInstances :many
SELECT instances.*, ...
FROM instances
LEFT JOIN instances_shared_with ON ...
WHERE instances.owner = $1 OR instances_shared_with.user_handle = $1
```

**After (queries.sql):**

```sql
-- name: GetAllAccessibleInstances :many
SELECT instances.*, 'owner' as role, true as is_owner
FROM instances
WHERE instances.owner = $1
UNION ALL
SELECT instances.*, isw.role, false as is_owner
FROM instances
INNER JOIN instances_shared_with isw ON instances.instance_id = isw.instance_id
WHERE isw.user_handle = $1 AND instances.owner != $1
ORDER BY owner, instance_handle
LIMIT $2 OFFSET $3;
```

**When to optimize:**

- Current implementation is **correct** and works well for small-medium datasets
- Consider optimization if performance becomes an issue with:
  - Large numbers of instances (>1000)
  - Many shared relationships (>100 shares per user)
  - Query time consistently >100ms

**Always profile first:**

```sql
EXPLAIN ANALYZE 
SELECT instances.*, ...
FROM instances
LEFT JOIN instances_shared_with ON ...
WHERE instances.owner = 'alice' OR instances_shared_with.user_handle = 'alice';
```

## Index Optimization

### Current Indexes

From migration `004_refactor_llm_services_architecture.sql`:

```sql
-- API Standards / Definitions
CREATE INDEX idx_definitions_handle 
  ON definitions(definition_handle);

CREATE INDEX idx_definitions_owner_handle 
  ON definitions(owner, definition_handle);

-- Instances
CREATE INDEX idx_instances_handle 
  ON instances(instance_handle);

-- Sharing (implicit from PRIMARY KEY)
-- instances_shared_with(instance_id, user_handle)
```

### Recommended Additional Indexes

#### 1. Owner Lookups

```sql
-- For queries: WHERE instances.owner = ?
CREATE INDEX idx_instances_owner 
  ON instances(owner);
```

**Benefit:** Fast retrieval of user's owned instances

**Use case:**
```sql
SELECT * FROM instances WHERE owner = 'alice';
```

#### 2. Shared Instance Lookups

```sql
-- For queries: WHERE user_handle = ?
CREATE INDEX idx_instances_shared_user 
  ON instances_shared_with(user_handle);
```

**Benefit:** Fast retrieval of instances shared with user

**Use case:**
```sql
SELECT i.* FROM instances i
INNER JOIN instances_shared_with isw ON i.instance_id = isw.instance_id
WHERE isw.user_handle = 'bob';
```

#### 3. Composite Owner+Handle Index

```sql
-- For unique constraint and lookups
CREATE UNIQUE INDEX idx_instances_owner_handle 
  ON instances(owner, instance_handle);
```

**Benefit:** Enforces uniqueness and enables index-only scans

**Use case:**
```sql
SELECT * FROM instances 
WHERE owner = 'alice' AND instance_handle = 'my-service';
```

#### 4. Embedding Dimension Filtering

```sql
-- Critical for similarity search performance
CREATE INDEX idx_embeddings_project_dim 
  ON embeddings(project_id, vector_dim);
```

**Benefit:** Filters embeddings by dimension before vector comparison

**Use case:**
```sql
SELECT * FROM embeddings 
WHERE project_id = 123 
  AND vector_dim = 1536
ORDER BY vector <=> $1::vector
LIMIT 10;
```

#### 5. Project Ownership

```sql
CREATE INDEX idx_projects_owner 
  ON projects(owner);

CREATE UNIQUE INDEX idx_projects_owner_handle 
  ON projects(owner, project_handle);
```

### Index Analysis

Check if indexes are being used:

```sql
-- Analyze query plan
EXPLAIN ANALYZE 
SELECT * FROM instances WHERE owner = 'alice';

-- Check index usage statistics
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan,
  idx_tup_read,
  idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Find unused indexes
SELECT 
  schemaname,
  tablename,
  indexname
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND schemaname = 'public';
```

### Index Maintenance

```sql
-- Update statistics for query planner
ANALYZE instances;
ANALYZE instances_shared_with;
ANALYZE embeddings;

-- Rebuild index if fragmented
REINDEX INDEX idx_instances_owner;

-- Check index size
SELECT 
  indexname,
  pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```

## Vector Index Optimization

### HNSW vs IVFFlat

**Current implementation uses HNSW:**

```sql
CREATE INDEX embedding_vector_idx 
ON embeddings 
USING hnsw (vector vector_cosine_ops);
```

#### HNSW (Hierarchical Navigable Small World)

**Pros:**
- Better recall (finds more similar results)
- Better query performance
- No training required

**Cons:**
- Slower index build time
- Higher memory usage
- Larger index size

**Configuration options:**

```sql
-- Default: m=16, ef_construction=64
CREATE INDEX embedding_vector_idx 
ON embeddings 
USING hnsw (vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Higher quality (slower build): m=32, ef_construction=128
CREATE INDEX embedding_vector_idx_hq
ON embeddings 
USING hnsw (vector vector_cosine_ops)
WITH (m = 32, ef_construction = 128);
```

**Parameters:**
- `m`: Number of connections per layer (default 16, range 2-100)
- `ef_construction`: Size of candidate list during build (default 64, range 4-1000)
- Higher values = better recall but slower build and more memory

#### IVFFlat (Inverted File with Flat compression)

**Alternative for very large datasets:**

```sql
CREATE INDEX embedding_vector_idx_ivf
ON embeddings 
USING ivfflat (vector vector_cosine_ops)
WITH (lists = 100);
```

**Pros:**
- Faster index build
- Lower memory usage
- Smaller index size

**Cons:**
- Requires training (ANALYZE before creating index)
- Lower recall than HNSW
- Need to tune `lists` parameter

**When to use:**
- Dataset >1M embeddings
- Build time is critical
- Memory constrained environment

**Configuration:**

```sql
-- Rule of thumb: lists = sqrt(total_rows)
-- For 100K embeddings: lists = 316
-- For 1M embeddings: lists = 1000

-- Train the index
ANALYZE embeddings;

-- Create with appropriate lists
CREATE INDEX embedding_vector_idx_ivf
ON embeddings 
USING ivfflat (vector vector_cosine_ops)
WITH (lists = 1000);

-- Set probes at query time
SET ivfflat.probes = 10;  -- Default is 1, higher = better recall
```

### Query-Time Optimization

For HNSW, set `hnsw.ef_search`:

```sql
-- Default: ef_search = 40
-- Higher = better recall but slower queries
SET hnsw.ef_search = 100;

SELECT vector <=> $1::vector AS distance
FROM embeddings
WHERE project_id = 123
ORDER BY distance
LIMIT 10;
```

### Dimension-Specific Indexes

For multi-dimensional projects:

```sql
-- Separate indexes per common dimension
CREATE INDEX idx_embeddings_768_vector 
ON embeddings USING hnsw (vector vector_cosine_ops)
WHERE vector_dim = 768;

CREATE INDEX idx_embeddings_1536_vector 
ON embeddings USING hnsw (vector vector_cosine_ops)
WHERE vector_dim = 1536;

CREATE INDEX idx_embeddings_3072_vector 
ON embeddings USING hnsw (vector vector_cosine_ops)
WHERE vector_dim = 3072;
```

**Benefit:** Smaller indexes = faster queries for specific dimensions

## Caching Strategies

### 1. System Definitions Cache

System definitions rarely change:

```go
var (
	systemDefsCache   []models.Definition
	systemDefsCacheMu sync.RWMutex
	systemDefsCacheTTL = 5 * time.Minute
	systemDefsCacheExp time.Time
)

func GetSystemDefinitions(ctx context.Context, pool *pgxpool.Pool) ([]models.Definition, error) {
	systemDefsCacheMu.RLock()
	if time.Now().Before(systemDefsCacheExp) && systemDefsCache != nil {
		defer systemDefsCacheMu.RUnlock()
		return systemDefsCache, nil
	}
	systemDefsCacheMu.RUnlock()
	
	// Fetch from database
	defs, err := db.GetDefinitionsByOwner(ctx, "_system")
	if err != nil {
		return nil, err
	}
	
	// Update cache
	systemDefsCacheMu.Lock()
	systemDefsCache = defs
	systemDefsCacheExp = time.Now().Add(systemDefsCacheTTL)
	systemDefsCacheMu.Unlock()
	
	return defs, nil
}
```

### 2. User Instances Cache

Cache user's instance list with short TTL:

```go
type InstanceCache struct {
	data map[string][]models.Instance
	mu   sync.RWMutex
	ttl  time.Duration
	exp  map[string]time.Time
}

func (c *InstanceCache) Get(userHandle string) ([]models.Instance, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if exp, ok := c.exp[userHandle]; ok && time.Now().Before(exp) {
		return c.data[userHandle], true
	}
	return nil, false
}

func (c *InstanceCache) Set(userHandle string, instances []models.Instance) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data[userHandle] = instances
	c.exp[userHandle] = time.Now().Add(c.ttl)
}

func (c *InstanceCache) Invalidate(userHandle string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.data, userHandle)
	delete(c.exp, userHandle)
}
```

**Usage:**

```go
var instanceCache = &InstanceCache{
	data: make(map[string][]models.Instance),
	exp:  make(map[string]time.Time),
	ttl:  30 * time.Second,
}

func GetUserInstances(ctx context.Context, pool *pgxpool.Pool, userHandle string) ([]models.Instance, error) {
	// Check cache
	if instances, ok := instanceCache.Get(userHandle); ok {
		return instances, nil
	}
	
	// Query database
	instances, err := db.GetAccessibleInstances(ctx, userHandle)
	if err != nil {
		return nil, err
	}
	
	// Cache results
	instanceCache.Set(userHandle, instances)
	return instances, nil
}
```

### 3. Project Metadata Cache

Cache project metadata including schema:

```go
type ProjectCache struct {
	projects map[string]*models.Project  // key: "owner/handle"
	mu       sync.RWMutex
	ttl      time.Duration
}

func (c *ProjectCache) Get(owner, handle string) (*models.Project, bool) {
	key := fmt.Sprintf("%s/%s", owner, handle)
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	project, ok := c.projects[key]
	return project, ok
}
```

### 4. Redis-Based Caching

For distributed deployments:

```go
import "github.com/go-redis/redis/v8"

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func (c *RedisCache) GetProject(ctx context.Context, owner, handle string) (*models.Project, error) {
	key := fmt.Sprintf("project:%s:%s", owner, handle)
	
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil  // Not in cache
	} else if err != nil {
		return nil, err
	}
	
	var project models.Project
	err = json.Unmarshal(data, &project)
	return &project, err
}

func (c *RedisCache) SetProject(ctx context.Context, project *models.Project) error {
	key := fmt.Sprintf("project:%s:%s", project.Owner, project.Handle)
	data, err := json.Marshal(project)
	if err != nil {
		return err
	}
	
	return c.client.Set(ctx, key, data, c.ttl).Err()
}
```

### Cache Invalidation

Always invalidate cache on updates:

```go
func UpdateProject(ctx context.Context, pool *pgxpool.Pool, project *models.Project) error {
	// Update database
	err := db.UpdateProject(ctx, project)
	if err != nil {
		return err
	}
	
	// Invalidate cache
	projectCache.Invalidate(project.Owner, project.Handle)
	
	return nil
}
```

## Connection Pool Optimization

### Pool Configuration

```go
func InitDB(opts *models.Options) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatal(err)
	}
	
	// Connection pool settings
	config.MaxConns = 20                      // Max concurrent connections
	config.MinConns = 5                       // Keep-alive connections
	config.MaxConnLifetime = time.Hour        // Recycle connections
	config.MaxConnIdleTime = 5 * time.Minute  // Close idle connections
	config.HealthCheckPeriod = time.Minute    // Health check frequency
	
	// Statement cache
	config.ConnConfig.StatementCacheCapacity = 100
	
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	return pool
}
```

### Pool Sizing

**General rule:**

```
MaxConns = (available_cores * 2) + effective_spindle_count
```

**Example scenarios:**

- **4-core CPU, SSD**: MaxConns = 10-20
- **8-core CPU, SSD**: MaxConns = 20-40
- **Under heavy load**: Start conservative, increase based on monitoring

### Monitor Pool Usage

```go
func monitorPool(pool *pgxpool.Pool) {
	ticker := time.NewTicker(30 * time.Second)
	
	for range ticker.C {
		stat := pool.Stat()
		log.Printf("Pool stats: total=%d, idle=%d, acquired=%d, waiting=%d",
			stat.TotalConns(),
			stat.IdleConns(),
			stat.AcquiredConns(),
			stat.MaxConns()-stat.TotalConns(),
		)
	}
}
```

## Performance Testing

### Load Testing Setup

Use [vegeta](https://github.com/tsenart/vegeta) for HTTP load testing:

```bash
# Install vegeta
go install github.com/tsenart/vegeta@latest

# Create targets file
cat > targets.txt <<EOF
GET http://localhost:8880/v1/projects/alice
Authorization: Bearer alice_api_key

POST http://localhost:8880/v1/similars/alice/project1/doc1
Authorization: Bearer alice_api_key
Content-Type: application/json

GET http://localhost:8880/v1/embeddings/alice/project1?limit=100
Authorization: Bearer alice_api_key
EOF

# Run load test
echo "GET http://localhost:8880/v1/projects/alice" | \
  vegeta attack -duration=60s -rate=100 -header="Authorization: Bearer key" | \
  vegeta report
```

### Benchmark Tests

Create Go benchmarks:

```go
func BenchmarkGetAccessibleInstances(b *testing.B) {
	pool := setupBenchmarkDB(b)
	defer pool.Close()
	
	// Create test data
	createTestInstances(b, pool, 1000)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetAllAccessibleInstances(ctx, "testuser", 10, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimilaritySearch(b *testing.B) {
	pool := setupBenchmarkDB(b)
	defer pool.Close()
	
	// Create embeddings
	createTestEmbeddings(b, pool, 10000)
	
	ctx := context.Background()
	queryVector := generateRandomVector(1536)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SearchSimilar(ctx, pool, "alice", "project1", queryVector, 10, 0.5)
		if err != nil {
			b.Fatal(err)
		}
	}
}
```

**Run benchmarks:**

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkSimilaritySearch -benchmem ./internal/handlers

# Compare before/after
go test -bench=. -benchmem ./... > old.txt
# Make changes
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt
```

### Database Performance Testing

Test with realistic data:

```sql
-- Generate test data
INSERT INTO embeddings (project_id, text_id, vector, vector_dim, metadata)
SELECT 
  1,
  'doc_' || generate_series,
  array_fill(random()::real, ARRAY[1536])::vector,
  1536,
  '{"author": "test"}'::jsonb
FROM generate_series(1, 100000);

-- Test query performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT text_id, vector <=> $1::vector AS distance
FROM embeddings
WHERE project_id = 1 AND vector_dim = 1536
ORDER BY distance
LIMIT 10;
```

## Metrics and Monitoring

### Application Metrics

Track key metrics:

```go
type Metrics struct {
	QueryDuration    prometheus.Histogram
	QueryCount       prometheus.Counter
	CacheHits        prometheus.Counter
	CacheMisses      prometheus.Counter
	PoolWaitDuration prometheus.Histogram
}

func recordQueryMetrics(start time.Time, query string) {
	duration := time.Since(start).Seconds()
	metrics.QueryDuration.Observe(duration)
	metrics.QueryCount.Inc()
}
```

### PostgreSQL Metrics

Monitor database performance:

```sql
-- Slow queries
SELECT 
  query,
  calls,
  total_time,
  mean_time,
  max_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- Table statistics
SELECT 
  schemaname,
  tablename,
  n_tup_ins,
  n_tup_upd,
  n_tup_del,
  n_live_tup,
  n_dead_tup
FROM pg_stat_user_tables;

-- Index usage
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan,
  idx_tup_read,
  idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

### Performance Targets

Based on typical usage:

| Operation | Target | Acceptable | Action Required |
|-----------|--------|------------|-----------------|
| Single instance lookup | < 10ms | < 50ms | > 50ms |
| List accessible instances (<100) | < 50ms | < 100ms | > 100ms |
| Create/update instance | < 100ms | < 200ms | > 200ms |
| Similarity search (10 results) | < 50ms | < 100ms | > 100ms |
| Similarity search (100 results) | < 100ms | < 200ms | > 200ms |
| Embedding insert (single) | < 50ms | < 100ms | > 100ms |
| Embedding batch (100) | < 500ms | < 1000ms | > 1000ms |

## Implementation Priority

### High Priority

1. **Profile current performance** with realistic data
2. **Add dimension filtering index**: `idx_embeddings_project_dim`
3. **Monitor slow queries** with pg_stat_statements

### Medium Priority

1. **Implement UNION ALL optimization** if GetAllAccessibleInstances > 100ms
2. **Add caching for system definitions**
3. **Optimize connection pool settings** based on load

### Low Priority

1. **Add Redis caching layer** for high-traffic deployments
2. **Implement application metrics** with Prometheus
3. **Add additional indexes** based on actual query patterns
4. **Tune HNSW parameters** for specific use cases

## General Best Practices

### 1. Measure Before Optimizing

```bash
# Profile in production
EXPLAIN ANALYZE your_query;

# Load test
vegeta attack -duration=60s -rate=100

# Benchmark
go test -bench=. -benchmem
```

### 2. Start Conservative

- Don't optimize prematurely
- Use simple queries first
- Add complexity only when needed

### 3. Monitor Continuously

- Track query performance
- Monitor connection pool
- Watch index usage
- Alert on slow queries

### 4. Document Optimizations

- Note why optimization was needed
- Include before/after metrics
- Document trade-offs made

## Further Reading

- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [pgvector Performance Guide](https://github.com/pgvector/pgvector#performance)
- [Go Database/SQL Tutorial](https://go.dev/doc/database/querying)
- [Connection Pool Best Practices](https://github.com/jackc/pgx/wiki/Connection-Pool-Best-Practices)
