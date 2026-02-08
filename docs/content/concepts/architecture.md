---
title: "Architecture"
weight: 1
---

# Architecture

embapi is a vector database API designed for RAG (Retrieval Augmented Generation) workflows in Digital Humanities research.

## System Overview

```
┌─────────────┐
│   Client    │
│ Application │
└──────┬──────┘
       │ HTTP/REST
       │
┌──────▼──────────────────────────┐
│      embapi API Server      │
│  ┌──────────────────────────┐   │
│  │   Authentication Layer   │   │
│  └────────┬─────────────────┘   │
│  ┌────────▼─────────────────┐   │
│  │   Request Handlers       │   │
│  │  (Users, Projects, etc)  │   │
│  └────────┬─────────────────┘   │
│  ┌────────▼─────────────────┐   │
│  │   Validation Layer       │   │
│  │  (Dimensions, Metadata)  │   │
│  └────────┬─────────────────┘   │
│  ┌────────▼─────────────────┐   │
│  │     SQLC Queries         │   │
│  │  (Type-safe SQL)         │   │
│  └────────┬─────────────────┘   │
└───────────┼──────────────────────┘
            │
    ┌───────▼──────────────┐
    │   PostgreSQL + 16    │
    │  with pgvector 0.7   │
    │                      │
    │  ┌────────────────┐  │
    │  │ Vector Index   │  │
    │  │ (HNSW/IVFFlat) │  │
    │  └────────────────┘  │
    └──────────────────────┘
```

## Core Components

### API Layer

Built with [Huma](https://huma.rocks/) framework on top of Go's `http.ServeMux`:

- OpenAPI documentation generation
- Automatic request/response validation
- JSON schema support
- REST endpoint routing

### Authentication

Token-based authentication using API keys:

- **Admin key**: For administrative operations (user creation, system management)
- **User keys**: SHA-256 hashed, unique per user
- **Bearer token**: Transmitted in `Authorization` header

### Data Storage

PostgreSQL with pgvector extension:

- **Vector storage**: Native pgvector support for embeddings
- **Vector search**: Cosine similarity using `<=>` operator
- **ACID compliance**: Transactional consistency
- **Relational integrity**: Foreign keys and constraints

### Code Generation

Uses [sqlc](https://sqlc.dev/) for type-safe database queries:

- SQL queries → Go functions
- Compile-time type checking
- No ORM overhead
- Direct PostgreSQL integration

## Data Model

### Core Entities

```
users
  ├── projects (1:many)
  │   ├── embeddings (1:many)
  │   └── instance (1:1)
  │
  └── instances (1:many)
      └── definition (many:1, optional)

_system (special user)
  └── definitions (1:many)
```

### Key Relationships

**Users → Projects**
- One user owns many projects
- Projects can be shared with other users (reader/editor roles)
- Projects can be public (unauthenticated read access)

**Projects → Instances**
- Each project references exactly one LLM service instance
- Instance defines embedding dimensions and configuration

**Projects → Embeddings**
- One project contains many embeddings
- Each embedding has a unique text_id within the project
- Embeddings store vector, metadata, and optional text

**Users → Instances**
- Users own their instances
- Instances can be shared with other users
- Instances store encrypted API keys

**Instances → Definitions**
- Instances can optionally reference a definition (template)
- System definitions (`_system` owner) provide defaults
- User definitions allow custom templates

## Request Flow

### 1. Create Embedding

```
Client Request
     ↓
Authentication Middleware
     ↓
Authorization Check (owner/editor?)
     ↓
Dimension Validation (vector_dim matches instance?)
     ↓
Metadata Validation (matches project schema?)
     ↓
Database Insert (with transaction)
     ↓
Response
```

### 2. Similarity Search

```
Client Request (text_id or vector)
     ↓
Authentication Middleware (or public check)
     ↓
Authorization Check (owner/reader/public?)
     ↓
Dimension Validation (if raw vector)
     ↓
Vector Similarity Query
     ├── Cosine distance calculation
     ├── Threshold filtering
     ├── Metadata filtering (exclude matches)
     └── Limit/offset pagination
     ↓
Results (sorted by similarity)
     ↓
Response
```

## Storage Architecture

### Vector Index

pgvector supports multiple index types:

- **IVFFlat**: Faster build, approximate search
- **HNSW**: Slower build, better recall

Current implementation uses HNSW for better accuracy.

### Vector Storage Format

```sql
CREATE TABLE embeddings (
  embedding_id SERIAL PRIMARY KEY,
  text_id TEXT NOT NULL,
  project_id INT REFERENCES projects,
  vector vector(3072),  -- Dimension varies
  vector_dim INT NOT NULL,
  metadata JSONB,
  text TEXT,
  ...
)
```

### Index Strategy

```sql
CREATE INDEX embedding_vector_idx 
ON embeddings 
USING hnsw (vector vector_cosine_ops);
```

Optimized for cosine similarity searches.

## Security Architecture

### API Key Encryption

- **Algorithm**: AES-256-GCM
- **Key Source**: `ENCRYPTION_KEY` environment variable
- **Key Derivation**: SHA-256 hash to ensure 32-byte key
- **Storage**: Binary (BYTEA) in database

### Access Control

**Three-tier access model:**

1. **Owner**: Full control (read, write, delete, share, transfer)
2. **Editor**: Read and write embeddings
3. **Reader**: Read-only access to embeddings and search

**Special access:**
- **Admin**: System-wide operations (user management, sanity checks)
- **Public**: Unauthenticated read access (if `public_read=true`)

### Data Isolation

- Users can only access their own resources or shared resources
- Cross-user queries are prevented at the database level
- Project ownership enforced via foreign keys

## Migration System

Uses [tern](https://github.com/jackc/tern) for database migrations:

```
migrations/
  ├── 001_create_initial_scheme.sql
  ├── 002_create_emb_index.sql
  ├── 003_add_public_read_flag.sql
  └── 004_refactor_llm_services_architecture.sql
```

Migrations run automatically on startup with rollback support.

## Performance Characteristics

### Vector Search Performance

- **Small datasets** (<10K embeddings): <10ms per query
- **Medium datasets** (10K-100K): 10-50ms per query
- **Large datasets** (>100K): 50-200ms per query

Performance depends on:
- Vector dimensions
- Index type and parameters
- Hardware (CPU, RAM, disk)
- Number of results requested

### Scaling Considerations

**Vertical Scaling:**
- More RAM = faster searches (more vectors in memory)
- Faster CPUs = faster vector comparisons
- SSD storage = faster index scans

**Horizontal Scaling:**
- Read replicas for search queries
- Separate write/read workloads
- Connection pooling for concurrent requests

## Technology Stack

### Core Technologies

- **Language**: Go 1.21+
- **Web Framework**: Huma 2.x
- **Database**: PostgreSQL 16+
- **Vector Extension**: pgvector 0.7.4
- **Query Generator**: sqlc 1.x
- **Migration Tool**: tern 2.x

### Development Tools

- **Testing**: Go standard library + testcontainers
- **Documentation**: OpenAPI 3.0 (auto-generated)
- **Building**: Docker multi-stage builds
- **Deployment**: Docker Compose

## Design Principles

### 1. Type Safety

- sqlc generates type-safe Go code from SQL
- Strong typing prevents SQL injection
- Compile-time validation of queries

### 2. Simplicity

- REST API (not GraphQL)
- Straightforward URL patterns
- Standard HTTP methods

### 3. Security

- API key encryption at rest
- No API keys in responses
- Role-based access control

### 4. Validation

- Automatic dimension validation
- Optional metadata schema validation
- Request/response validation via OpenAPI

### 5. Extensibility

- User-defined metadata schemas
- Custom LLM service configurations
- Flexible sharing model

## Limitations

### Current Constraints

- **No multi-tenancy**: Each installation is single-tenant
- **No replication**: Manual setup required for HA
- **No caching**: All queries hit database
- **Synchronous API**: No async/batch upload endpoints

### Future Enhancements

See [Roadmap](../../reference/roadmap/) for planned improvements.

## Next Steps

- [Learn about users and authentication](users-and-auth/)
- [Understand projects](projects/)
- [Explore LLM services](llm-services/)
