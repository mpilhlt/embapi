---
title: "Database Schema Reference"
weight: 2
---

# Database Schema Reference

Complete reference for the dhamps-vdb PostgreSQL database schema. This document describes all tables, columns, types, constraints, relationships, and indexes.

## Overview

The database uses PostgreSQL 12+ with the pgvector extension for vector similarity search. The schema is managed through migrations in `internal/database/migrations/`.

**Key Features:**
- Vector embeddings stored as `halfvec` for efficient storage
- HNSW indexes for fast approximate nearest neighbor search
- Automatic timestamp tracking (`created_at`, `updated_at`)
- Foreign key constraints with CASCADE deletion
- Role-based access control through association tables
- Multi-tenancy support (user-owned resources)

## Schema Migrations

Current schema version is defined by 4 migration files:

1. **001_create_initial_scheme.sql** - Core tables and relationships
2. **002_create_emb_index.sql** - HNSW vector indexes
3. **003_add_public_read_flag.sql** - Public access support
4. **004_refactor_llm_services_architecture.sql** - LLM service architecture refactor

## Core Tables

### users

Stores user accounts with API authentication.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `user_handle` | `VARCHAR(20)` | PRIMARY KEY | Unique user identifier (username) |
| `name` | `TEXT` | | Full name or display name |
| `email` | `TEXT` | UNIQUE, NOT NULL | Email address |
| `vdb_key` | `CHAR(64)` | UNIQUE, NOT NULL | API key (64-character hex) |
| `created_at` | `TIMESTAMP` | NOT NULL | Creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Indexes:**
- Primary key on `user_handle`
- Unique constraint on `email`
- Unique constraint on `vdb_key`

**Special Users:**
- `_system` - Reserved system user for global LLM service definitions

**Relationships:**
- Owns projects (1:N)
- Owns LLM service instances (1:N)
- Owns embeddings (1:N)
- Can share projects (N:M via `users_projects`)
- Can share LLM service instances (N:M via `instances_shared_with`)

**Notes:**
- `vdb_key` is generated during user creation and returned once
- Cannot be recovered if lost
- Transmitted as `Authorization: Bearer` token

### projects

Stores embedding projects owned by users.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `project_id` | `SERIAL` | PRIMARY KEY | Auto-incrementing project ID |
| `project_handle` | `VARCHAR(20)` | NOT NULL | Project identifier (unique per owner) |
| `owner` | `VARCHAR(20)` | NOT NULL, FK→users | Project owner user handle |
| `description` | `TEXT` | | Project description |
| `metadata_scheme` | `TEXT` | | JSON Schema for validating embedding metadata |
| `public_read` | `BOOLEAN` | DEFAULT FALSE | Allow unauthenticated read access |
| `instance_id` | `INTEGER` | FK→instances | LLM service instance used for embeddings |
| `created_at` | `TIMESTAMP` | NOT NULL | Creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `UNIQUE (owner, project_handle)` - Project handles unique per owner
- `ON DELETE CASCADE` - Delete project when owner deleted
- `ON DELETE RESTRICT` - Prevent instance deletion if used by project

**Relationships:**
- Owned by one user (N:1)
- Uses one LLM service instance (N:1)
- Has many embeddings (1:N)
- Can be shared with users (N:M via `users_projects`)

**Metadata Schema:**
- Stored as JSON Schema string
- Used to validate embedding metadata
- Optional (can be NULL)
- See [Data Validation](../../concepts/metadata/) for details

**Public Access:**
- When `public_read=TRUE`, allows unauthenticated embedding queries
- Controlled via API or by setting shared_with to `["*"]`

### embeddings

Stores vector embeddings with text identifiers and metadata.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `embeddings_id` | `SERIAL` | PRIMARY KEY | Auto-incrementing embedding ID |
| `text_id` | `TEXT` | INDEXED | Text identifier (URL, DOI, custom ID) |
| `owner` | `VARCHAR(20)` | NOT NULL, FK→users | Embedding owner user handle |
| `project_id` | `SERIAL` | NOT NULL, FK→projects | Project the embedding belongs to |
| `instance_id` | `SERIAL` | NOT NULL, FK→instances | LLM service instance used |
| `text` | `TEXT` | | Optional full text of the embedded content |
| `vector` | `halfvec` | NOT NULL | Embedding vector (half-precision float) |
| `vector_dim` | `INTEGER` | NOT NULL | Vector dimensionality |
| `metadata` | `jsonb` | | Optional metadata (validated if schema defined) |
| `created_at` | `TIMESTAMP` | NOT NULL | Creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `UNIQUE (text_id, owner, project_id, instance_id)` - Unique text IDs per project/instance
- `ON DELETE CASCADE` - Delete embeddings when owner, project, or instance deleted

**Indexes:**
- Primary key on `embeddings_id`
- B-tree index on `text_id`
- HNSW vector indexes for dimensions: 384, 768, 1024, 1536, 3072 (see [Vector Indexes](#vector-indexes))

**Vector Storage:**
- Uses `halfvec` type (16-bit floating point) for efficient storage
- Dimensions must match LLM service instance configuration
- Validated on upload against instance dimensions

**Relationships:**
- Owned by one user (N:1)
- Belongs to one project (N:1)
- Created with one LLM service instance (N:1)

## LLM Service Architecture

The LLM service architecture separates service definitions (templates) from user-specific instances.

### definitions

Templates for LLM embedding services (shared or user-specific).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `definition_id` | `SERIAL` | PRIMARY KEY | Auto-incrementing definition ID |
| `definition_handle` | `VARCHAR(20)` | NOT NULL | Definition identifier (unique per owner) |
| `owner` | `VARCHAR(20)` | NOT NULL, FK→users | Definition owner (_system for global) |
| `endpoint` | `TEXT` | NOT NULL | API endpoint URL |
| `description` | `TEXT` | | Service description |
| `api_standard` | `VARCHAR(20)` | NOT NULL, FK→api_standards | API standard handle |
| `model` | `TEXT` | NOT NULL | Model name (e.g., text-embedding-3-large) |
| `dimensions` | `INTEGER` | NOT NULL | Vector dimensions produced by model |
| `context_limit` | `INTEGER` | NOT NULL | Maximum context length (tokens/chars) |
| `is_public` | `BOOLEAN` | NOT NULL, DEFAULT FALSE | Share with all users if true |
| `created_at` | `TIMESTAMP` | NOT NULL | Creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `UNIQUE (owner, definition_handle)` - Definition handles unique per owner
- `ON DELETE CASCADE` - Delete definition when owner deleted

**Built-in Definitions:**

System-provided definitions (owned by `_system`, `is_public=TRUE`):

| Handle | Model | Dimensions | Context Limit | API Standard |
|--------|-------|------------|---------------|--------------|
| `openai-large` | text-embedding-3-large | 3072 | 8192 | openai |
| `openai-small` | text-embedding-3-small | 1536 | 8191 | openai |
| `cohere-v4` | embed-v4.0 | 1536 | 128000 | cohere |
| `gemini-embedding-001` | gemini-embedding-001 | 3072 | 2048 | gemini |

**Relationships:**
- Owned by one user (N:1)
- References one API standard (N:1)
- Can be shared with users (N:M via `definitions_shared_with`)
- Used by instances (1:N)

### instances

User-specific configurations of LLM services with API keys.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `instance_id` | `SERIAL` | PRIMARY KEY | Auto-incrementing instance ID |
| `instance_handle` | `VARCHAR(20)` | NOT NULL | Instance identifier (unique per owner) |
| `owner` | `VARCHAR(20)` | NOT NULL, FK→users | Instance owner user handle |
| `endpoint` | `TEXT` | NOT NULL | API endpoint URL |
| `description` | `TEXT` | | Instance description |
| `api_standard` | `VARCHAR(20)` | NOT NULL, FK→api_standards | API standard handle |
| `model` | `TEXT` | NOT NULL | Model name |
| `dimensions` | `INTEGER` | NOT NULL | Vector dimensions |
| `context_limit` | `INTEGER` | NOT NULL | Maximum context length |
| `definition_id` | `INTEGER` | FK→definitions | Reference to definition template |
| `api_key_encrypted` | `BYTEA` | | Encrypted API key for service authentication |
| `created_at` | `TIMESTAMP` | NOT NULL | Creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `UNIQUE (owner, instance_handle)` - Instance handles unique per owner
- `ON DELETE CASCADE` - Delete instance when owner deleted
- `ON DELETE SET NULL` - Keep instance if definition deleted

**Indexes:**
- Composite index on `(owner, instance_handle)`

**API Key Encryption:**
- Encrypted using AES-256-GCM
- Encryption key from `ENCRYPTION_KEY` environment variable
- Cannot be recovered if encryption key is lost
- Stored as base64-encoded BYTEA

**Relationships:**
- Owned by one user (N:1)
- References one API standard (N:1)
- Based on one definition template (N:1, optional)
- Used by projects (1:N)
- Used by embeddings (1:N)
- Can be shared with users (N:M via `instances_shared_with`)

### api_standards

Defines API specifications for LLM embedding services.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `api_standard_handle` | `VARCHAR(20)` | PRIMARY KEY | Standard identifier (e.g., openai, cohere) |
| `description` | `TEXT` | | API description and version info |
| `key_method` | `VARCHAR(20)` | NOT NULL, FK→key_methods | Authentication method |
| `key_field` | `VARCHAR(20)` | | Header/field name for API key |
| `created_at` | `TIMESTAMP` | NOT NULL | Creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Built-in Standards:**

| Handle | Description | Key Method | Key Field |
|--------|-------------|------------|-----------|
| `openai` | OpenAI Embeddings API v1 | auth_bearer | Authorization |
| `cohere` | Cohere Embed API v2 | auth_bearer | Authorization |
| `gemini` | Gemini Embeddings API | auth_bearer | x-goog-api-key |

**Relationships:**
- Used by definitions (1:N)
- Used by instances (1:N)
- References one key_method (N:1)

## Association Tables

### users_projects

Defines project sharing and access control.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `user_handle` | `VARCHAR(20)` | FK→users, PK | User being granted access |
| `project_id` | `SERIAL` | FK→projects, PK | Project being shared |
| `role` | `VARCHAR(20)` | NOT NULL, FK→vdb_roles | Access level (owner/editor/reader) |
| `created_at` | `TIMESTAMP` | NOT NULL | Share creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `PRIMARY KEY (user_handle, project_id)` - One role per user per project
- `ON DELETE CASCADE` - Remove sharing when user or project deleted

**Roles:**
- `owner` - Full control (only project creator)
- `editor` - Can add/modify/delete embeddings
- `reader` - Read-only access to embeddings

### instances_shared_with

Defines LLM service instance sharing.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `user_handle` | `VARCHAR(20)` | FK→users, PK | User being granted access |
| `instance_id` | `INTEGER` | FK→instances, PK | Instance being shared |
| `role` | `VARCHAR(20)` | NOT NULL, FK→vdb_roles | Access level |
| `created_at` | `TIMESTAMP` | NOT NULL | Share creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `PRIMARY KEY (user_handle, instance_id)` - One role per user per instance
- `ON DELETE CASCADE` - Remove sharing when user or instance deleted

### definitions_shared_with

Defines LLM service definition sharing.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `user_handle` | `VARCHAR(20)` | FK→users, PK | User being granted access |
| `definition_id` | `INTEGER` | FK→definitions, PK | Definition being shared |
| `created_at` | `TIMESTAMP` | NOT NULL | Share creation timestamp |
| `updated_at` | `TIMESTAMP` | NOT NULL | Last update timestamp |

**Constraints:**
- `PRIMARY KEY (user_handle, definition_id)` - One share per user per definition
- `ON DELETE CASCADE` - Remove sharing when user or definition deleted

**Indexes:**
- Index on `user_handle` for efficient user lookups
- Index on `definition_id` for efficient definition lookups

## Reference Tables

### vdb_roles

Enumeration of valid access roles.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `vdb_role` | `VARCHAR(20)` | PRIMARY KEY | Role name |

**Values:**
- `owner` - Full control over resource
- `editor` - Read and write access
- `reader` - Read-only access

### key_methods

Enumeration of API authentication methods.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `key_method` | `VARCHAR(20)` | PRIMARY KEY | Authentication method |

**Values:**
- `auth_bearer` - Bearer token in Authorization header
- `body_form` - API key in request body
- `query_param` - API key in URL query parameter
- `custom_header` - API key in custom header

## Vector Indexes

HNSW (Hierarchical Navigable Small World) indexes for fast approximate nearest neighbor search.

### Index Configuration

All vector indexes use HNSW with these parameters:
- `m = 24` - Number of neighbors per node
- `ef_construction = 200` - Build-time accuracy parameter
- `ef_search = 100` - Query-time accuracy parameter
- Expected recall: ~99.8%

### Dimension-Specific Indexes

| Index Name | Dimensions | Common Models |
|-----------|------------|---------------|
| `embeddings_vector_384` | 384 | Cohere embed-multilingual-light-v3.0, embed-english-light-v3.0 |
| `embeddings_vector_768` | 768 | BERT base, Cohere embed-multilingual-v2.0, Gemini Embeddings |
| `embeddings_vector_1024` | 1024 | BERT large, SBERT, Cohere embed-multilingual-v3.0, embed-english-v3.0 |
| `embeddings_vector_1536` | 1536 | OpenAI text-embedding-ada-002, text-embedding-3-small, Cohere embed-v4.0 |
| `embeddings_vector_3072` | 3072 | OpenAI text-embedding-3-large, Gemini embedding-001 |

**Index Structure:**
```sql
CREATE INDEX embeddings_vector_1536 ON embeddings 
USING hnsw ((vector::halfvec(1536)) halfvec_cosine_ops) 
WITH (m = 24, ef_construction = 200) 
WHERE (vector_dim = 1536);
```

**Usage:**
- Indexes are partial - only include vectors of matching dimension
- Automatically used for similarity queries with matching dimensions
- Use cosine distance for similarity calculation

## Relationships Diagram

```
users (1) ──owns──> (N) projects (1) ──contains──> (N) embeddings
  │                      │                              │
  │                      └─> uses (1) instances (N) <───┘
  │                                    │
  │                                    └─> based on (1) definitions
  │                                              │
  ├──owns──> (N) instances ──uses──> (1) api_standards
  │               │                          │
  │               └─> references (1) definitions
  │
  └──owns──> (N) definitions ──references──> (1) api_standards

Sharing:
  users (N) <──shares─> (M) projects (via users_projects)
  users (N) <──shares─> (M) instances (via instances_shared_with)
  users (N) <──shares─> (M) definitions (via definitions_shared_with)
```

## Data Validation

### Dimension Validation

Embeddings must have dimensions matching their LLM service instance:
- `embeddings.vector_dim` must equal `instances.dimensions`
- Enforced at API level during upload
- Similarity queries automatically filter by matching dimensions

### Metadata Validation

Projects can define JSON Schema in `metadata_scheme`:
- Validates all embedding metadata on upload
- Optional - if NULL, metadata not validated
- Enforced at API level
- Admin sanity check validates all existing metadata

### Sanity Check Queries

The `/v1/admin/sanity-check` endpoint verifies:
1. All embeddings have dimensions matching their instance
2. All metadata conforms to project schemas (if defined)

## Common Queries

### Find User's Projects

```sql
SELECT p.project_handle, p.description, p.created_at
FROM projects p
WHERE p.owner = 'alice'
ORDER BY p.created_at DESC;
```

### Find Shared Projects

```sql
SELECT p.project_handle, p.owner, up.role
FROM users_projects up
JOIN projects p ON up.project_id = p.project_id
WHERE up.user_handle = 'bob'
ORDER BY p.owner, p.project_handle;
```

### Find Similar Embeddings

```sql
SELECT text_id, vector <-> '[0.1, 0.2, ...]'::vector AS distance
FROM embeddings
WHERE project_id = 123 AND vector_dim = 1536
ORDER BY vector <-> '[0.1, 0.2, ...]'::vector
LIMIT 10;
```

### Count Embeddings by Project

```sql
SELECT p.project_handle, COUNT(e.embeddings_id) AS embedding_count
FROM projects p
LEFT JOIN embeddings e ON p.project_id = e.project_id
WHERE p.owner = 'alice'
GROUP BY p.project_id, p.project_handle
ORDER BY embedding_count DESC;
```

## Database Maintenance

### Backup Recommendations

**Critical Data:**
- User accounts (`users`)
- API keys (`vdb_key` in users, `api_key_encrypted` in instances)
- ENCRYPTION_KEY environment variable (backup separately!)
- Projects and embeddings

**Backup Strategy:**
```bash
# Full database backup
pg_dump -U postgres dhamps_vdb > backup.sql

# Backup encryption key separately
echo "$ENCRYPTION_KEY" > encryption_key.backup
chmod 400 encryption_key.backup
```

### Vacuum and Analyze

```sql
-- Regular maintenance
VACUUM ANALYZE embeddings;
VACUUM ANALYZE projects;

-- Rebuild vector indexes if needed
REINDEX INDEX embeddings_vector_1536;
```

### Monitoring Queries

```sql
-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE tablename = 'embeddings'
ORDER BY idx_scan DESC;

-- Check table sizes
SELECT relname, pg_size_pretty(pg_total_relation_size(oid))
FROM pg_class
WHERE relname IN ('embeddings', 'projects', 'users')
ORDER BY pg_total_relation_size(oid) DESC;
```

## Related Documentation

- [Concepts - Architecture](../../concepts/architecture/)
- [Concepts - Projects](../../concepts/projects/)
- [Concepts - Metadata](../../concepts/metadata/)
- [Deployment - Database Setup](../../deployment/database/)
- [Reference - Configuration](configuration/)
- [API - Endpoints](../../api/endpoints/)
