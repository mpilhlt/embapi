---
title: embapi Documentation
type: docs
---

# embapi Documentation

Welcome to the documentation for **embapi**, a vector database designed for Digital Humanities applications at the Max Planck Society initiative.

## What is embapi?

embapi is a PostgreSQL-backed vector database with pgvector support, providing a RESTful API for managing embeddings in Retrieval Augmented Generation (RAG) workflows. It offers multi-user support, project management, and flexible embedding configurations.

## Key Features

- **Multi-user Support**: Role-based access control (admin, owner, reader, editor)
- **Project Management**: Organize embeddings into projects with sharing capabilities
- **LLM Service Management**: Flexible service definitions and instances with encrypted API keys
- **Metadata Support**: JSON Schema validation and filtering in similarity search
- **PostgreSQL Backend**: Reliable storage with pgvector extension
- **RESTful API**: OpenAPI-documented endpoints
- **Docker Ready**: Easy deployment with Docker Compose

## Quick Links

- [Getting Started](/getting-started/) - Installation and first steps
- [Concepts](/concepts/) - Understand how embapi works
- [API Reference](/api/) - Complete API documentation
- [Guides](/guides/) - How-to guides for common tasks

## Getting Help

- 📖 Browse this documentation
- 🐛 [Report issues](https://github.com/mpilhlt/embapi/issues)
- 💬 [GitHub Discussions](https://github.com/mpilhlt/embapi/discussions)

## Quick Example

```bash
# Start the service with Docker
./docker-setup.sh
docker-compose up -d

# Create a user
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_handle": "alice", "name": "Alice Smith"}'

# Create a project and start working with embeddings
# See the Getting Started guide for a complete walkthrough
```

Ready to get started? Head over to the [Installation Guide](/getting-started/installation/).
