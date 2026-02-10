# EmbAPI ⚽
Vector Database for the DH at Max Planck Society initiative

[![Go Report Card](https://goreportcard.com/badge/github.com/mpilhlt/embapi?style=flat-square)](https://goreportcard.com/report/github.com/mpilhlt/embapi) [![Release](https://img.shields.io/github/v/release/mpilhlt/embapi.svg?style=flat-square&include_prereleases)](https://github.com/mpilhlt/embapi/releases/latest)

## Introduction

EmbAPI (/ɛmˈbɑːpeɪ/) ⚽ is a PostgreSQL-backed vector database with pgvector support, providing a RESTful API for managing embeddings in Retrieval Augmented Generation (RAG) workflows. Store embeddings for text snippets with metadata, then find similar content using cosine similarity search.

The typical use case is as a RAG component: Create embeddings for your text collection, upload them with identifiers and optional metadata, then query for similar texts either by identifier (`GET`) or by posting raw embeddings (`POST`). The service returns text identifiers with similarity scores for use in your application.

## Features

### Core Capabilities
- **PostgreSQL with pgvector backend** - Reliable, scalable vector storage
- **RESTful API** - OpenAPI-documented endpoints
- **Docker deployment ready** - Includes PostgreSQL with pgvector
- **Comprehensive test coverage** - Integration tests with testcontainers

### Multi-User & Access Control
- **Multi-user support** - Role-based access control (admin, owner, reader, editor)
- **Project sharing** - Collaborate with specific users
- **Public access mode** - Enable unauthenticated read access for projects
- **Project ownership transfer** - Transfer projects between users

### LLM & Embedding Management
- **LLM service management** - Service definitions and instances with encrypted API keys
- **Multiple embedding configurations** - Support for different dimensions
- **Automatic dimension validation** - Ensures vector consistency
- **Flexible instance sharing** - Share LLM service instances across users

### Data Validation & Search
- **JSON Schema-based metadata validation** - Enforce metadata structure
- **Metadata filtering in similarity search** - Exclude documents by metadata field values
- **PATCH support** - Partial updates for projects and embeddings
- **Configurable thresholds** - Control similarity search results

## Quick Start

### 1. Start with Docker

```bash
# Automated setup (generates secure keys)
./docker-setup.sh

# Start services (includes PostgreSQL with pgvector)
docker-compose up -d

# Access the API documentation
curl http://localhost:8880/docs
```

### 2. Create a User

```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_handle": "alice", "name": "Alice Smith"}'

# Response includes: {"embapi_key": "alice_abc123..."}
# ⚠️ Save the embapi_key! It cannot be recovered.
```

### 3. Create an LLM Service Instance

```bash
# Use a system-provided definition (openai-large, openai-small, etc.)
curl -X PUT http://localhost:8880/v1/llm-instances/alice/my-openai \
  -H "Authorization: Bearer ALICE_EMBAPI_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "definition_owner": "_system",
    "definition_handle": "openai-large",
    "api_key_encrypted": "YOUR_OPENAI_API_KEY"
  }'
```

### 4. Create a Project

```bash
curl -X POST http://localhost:8880/v1/projects/alice \
  -H "Authorization: Bearer ALICE_EMBAPI_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "my-texts",
    "description": "My text embeddings",
    "instance_owner": "alice",
    "instance_handle": "my-openai"
  }'
```

### 5. Upload Embeddings

```bash
curl -X POST http://localhost:8880/v1/embeddings/alice/my-texts \
  -H "Authorization: Bearer ALICE_EMBAPI_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc1",
      "instance_handle": "my-openai",
      "vector": [0.1, 0.2, 0.3, ...],
      "vector_dim": 3072,
      "metadata": {"author": "John Doe", "year": 2024}
    }]
  }'
```

### 6. Find Similar Documents

```bash
# Get documents similar to doc1
curl "http://localhost:8880/v1/similars/alice/my-texts/doc1?threshold=0.7&limit=5" \
  -H "Authorization: Bearer ALICE_EMBAPI_KEY"
```

### 7. Filter by Metadata

```bash
# Exclude documents from the same author
curl "http://localhost:8880/v1/similars/alice/my-texts/doc1?threshold=0.7&metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer ALICE_EMBAPI_KEY"
```

## Getting Started

📚 **[Read the Full Documentation](https://mpilhlt.github.io/embapi/)**

- **[Installation Guide](https://mpilhlt.github.io/embapi/getting-started/installation/)** - Build from source
- **[Docker Guide](https://mpilhlt.github.io/embapi/getting-started/docker/)** - Detailed Docker deployment (also see [DOCKER.md](./DOCKER.md))
- **[Configuration](https://mpilhlt.github.io/embapi/getting-started/configuration/)** - Environment variables and options
- **[Your First Project](https://mpilhlt.github.io/embapi/getting-started/first-project/)** - Complete walkthrough
- **[API Reference](https://mpilhlt.github.io/embapi/api/)** - Complete API documentation

## Key Concepts

- **[Architecture](https://mpilhlt.github.io/embapi/concepts/architecture/)** - How EmbAPI works
- **[Users & Authentication](https://mpilhlt.github.io/embapi/concepts/users-and-auth/)** - API keys and roles
- **[Projects](https://mpilhlt.github.io/embapi/concepts/projects/)** - Organizing embeddings
- **[LLM Services](https://mpilhlt.github.io/embapi/concepts/llm-services/)** - Definitions vs. instances
- **[Similarity Search](https://mpilhlt.github.io/embapi/concepts/similarity-search/)** - How it works

## Common Tasks

- **[RAG Workflow](https://mpilhlt.github.io/embapi/guides/rag-workflow/)** - End-to-end example
- **[Project Sharing](https://mpilhlt.github.io/embapi/guides/project-sharing/)** - Collaborate with others
- **[Metadata Filtering](https://mpilhlt.github.io/embapi/guides/metadata-filtering/)** - Exclude documents in search
- **[Metadata Validation](https://mpilhlt.github.io/embapi/guides/metadata-validation/)** - Enforce schemas
- **[Public Projects](https://mpilhlt.github.io/embapi/guides/public-projects/)** - Unauthenticated access

## Development

### Building from Source

```bash
# Install dependencies and generate code
go get ./...
sqlc generate --no-remote

# Build
go build -o build/embapi main.go

# Or run directly
go run main.go
```

### Running Tests

Tests use testcontainers for integration testing:

```bash
# Start container runtime (if using podman)
systemctl --user start podman.socket
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock

# Run tests
go test -v ./...
```

For more details, see the **[Testing Guide](https://mpilhlt.github.io/embapi/development/testing/)**.

## Contributing

Contributions are welcome! Please see our **[Contributing Guide](https://mpilhlt.github.io/embapi/development/contributing/)** for details.

## License

This project is licensed under the terms specified in the [LICENSE](./LICENSE) file.

## Support

- 📖 [Documentation](https://mpilhlt.github.io/embapi/)
- 🐛 [Report Issues](https://github.com/mpilhlt/embapi/issues)
- 💬 [GitHub Discussions](https://github.com/mpilhlt/embapi/discussions)
