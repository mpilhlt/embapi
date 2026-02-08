---
title: "Product Roadmap"
weight: 3
---

# Product Roadmap

Development roadmap for dhamps-vdb, tracking completed features, in-progress work, and planned enhancements.

## Overview

This roadmap outlines the development priorities for dhamps-vdb. Items marked with [x] are completed, items in progress are noted, and planned features are listed by priority.

## Completed Features

### Core Functionality

- [x] **User authentication & restrictions on some API calls**
  - Bearer token authentication
  - Role-based access control
  - Admin vs user permissions

- [x] **API versioning**
  - Version 1 API with `/v1/` prefix
  - Backward compatibility support

- [x] **Better options handling**
  - Command-line flags via Huma CLI
  - Environment variable configuration
  - `.env` file support

- [x] **Handle metadata**
  - JSONB storage for flexible metadata
  - Metadata attached to embeddings

- [x] **Validation with metadata schema**
  - JSON Schema validation for embedding metadata
  - Project-level schema definitions
  - Automatic validation on upload

- [x] **Filter similar passages by metadata field**
  - Metadata-based filtering in similarity queries
  - Exclude documents by metadata value
  - Query parameters: `metadata_path` and `metadata_value`

### Data Management

- [x] **Use transactions**
  - Atomic operations for multi-step actions
  - Consistency for project creation with sharing
  - Rollback on errors

- [x] **Catch POST to existing resources**
  - Prevent duplicate creation
  - Return appropriate error codes
  - Suggest using PUT for updates

- [x] **Always use specific error messages**
  - Detailed error descriptions
  - Helpful troubleshooting information
  - Consistent error response format

### Testing & Quality

- [x] **Tests**
  - Integration tests for all major operations
  - Testcontainers for isolated database testing
  - Cleanup verification queries

- [x] **When testing, check cleanup by adding a new query/function to see if all tables are empty**
  - Verify test isolation
  - Ensure no data leakage between tests

- [x] **Make sure input is validated consistently**
  - Dimension validation for embeddings
  - Schema validation for metadata
  - Request validation via Huma

### Collaboration Features

- [x] **Add project sharing/unsharing functions & API paths**
  - Share projects with specific users
  - Define roles: owner, editor, reader
  - API endpoints for managing sharing

- [x] **Add mechanism to allow anonymous/public reading access to embeddings**
  - `public_read` flag on projects
  - Wildcard sharing via `"*"` in `shared_with`
  - Unauthenticated access to public embeddings

- [x] **Transfer of projects from one owner to another as new operation**
  - Owner-initiated project transfers
  - Ownership verification
  - Automatic cleanup of old owner associations

### Service Architecture

- [x] **Add definition creation/listing/deletion functions & paths**
  - LLM service definitions (templates)
  - Instances (user-specific configurations)
  - System-provided global definitions

### Deployment & Operations

- [x] **Dockerization**
  - Multi-stage Dockerfile
  - Docker Compose with PostgreSQL
  - External database support
  - Automated setup script

- [x] **Make sure pagination is supported consistently**
  - Limit and offset parameters
  - Consistent across all list endpoints
  - Documented pagination behavior

### Security

- [x] **Prevent acceptance of requests as user "_system"**
  - Reserved system user for internal use
  - Blocked from external authentication
  - Protected system-owned resources

## In Progress

### Documentation

- [ ] **Revisit all documentation**
  - Comprehensive reference documentation
  - Updated API examples
  - Docker deployment guides

- [ ] **Add documentation for metadata filtering of similars**
  - Document `metadata_path` and `metadata_value` parameters
  - Provide usage examples
  - Explain use cases (exclude same author, etc.)
  - **Note:** Query parameters are: `metadata_path` and `metadata_value` as in: `https://xy.org/vdb-api/v1/similars/sal/sal-openai-large/https%3A%2F%2Fid.myproject.net%2Ftexts%2FW0011%3A1.3.1.3.1?threshold=0.7&limit=5&metadata_path=author_id&metadata_value=A0083`

## Planned Features

### High Priority

#### Network Connectivity

- [ ] **Implement and make consequent use of max_idle (5), max_concurr (5), timeouts, and cancellations**
  - Connection pool management
  - Maximum idle connections: 5
  - Maximum concurrent connections: 5
  - Request timeouts
  - Context cancellation support

- [ ] **Concurrency (leaky bucket approach) and Rate limiting**
  - Leaky bucket algorithm for concurrency control
  - Rate limiting using Redis
  - Sliding window implementation
  - Standard rate limit headers
  - See [Huma request limits](https://huma.rocks/features/request-limits/) for implementation

- [ ] **Caching**
  - Response caching for frequently accessed data
  - Cache invalidation strategies
  - Redis or in-memory caching
  - Configurable TTL

#### API Standards

- [ ] **Add API standards for anthropic, mistral, llama.cpp, ollama, vllm, llmstudio**
  - Anthropic embeddings API
  - Mistral embeddings API
  - llama.cpp server API
  - Ollama embeddings API
  - vLLM embeddings API
  - LM Studio embeddings API
  - Standard authentication methods
  - Example definitions in testdata

### Medium Priority

#### User Experience

- [ ] **HTML UI**
  - Web-based interface for API management
  - User-friendly project creation
  - Visual embedding explorer
  - API key management
  - Alternative to CLI/API usage

- [ ] **Allow to request verbose information even in list outputs**
  - `verbose=yes` query parameter
  - Full object details in list endpoints
  - Optional vs default minimal output
  - Performance considerations

- [ ] **Add possibility to use PATCH method to change existing resources**
  - Partial updates without full replacement
  - PATCH support for users, projects, instances
  - Merge semantics for nested objects
  - Validation of partial updates
  - **Status:** Partially implemented via automatic PATCH handler

#### Logging and Monitoring

- [ ] **Proper logging with `--verbose` and `--quiet` modes**
  - Structured logging (JSON format)
  - Log levels: ERROR, WARN, INFO, DEBUG, TRACE
  - `--verbose` flag for detailed logs
  - `--quiet` flag for minimal logs
  - Request/response logging
  - Performance metrics logging
  - Integration with log aggregation systems

### Future Enhancements

#### Advanced Features

- [ ] **Bulk Operations**
  - Batch embedding upload
  - Bulk deletion
  - Transaction support for large operations

- [ ] **Advanced Search**
  - Combined vector + metadata filtering
  - Hybrid search (vector + keyword)
  - Multi-vector search
  - Weighted search results

- [ ] **Embeddings Management**
  - Update embeddings in place
  - Re-embedding workflows
  - Embedding versioning
  - Dimension conversion utilities

#### Performance

- [ ] **Query Optimization**
  - Query plan analysis
  - Index optimization
  - Materialized views for common queries
  - Database connection pooling improvements

- [ ] **Scaling**
  - Horizontal scaling support
  - Read replicas for query load
  - Partitioning strategies for large datasets
  - Distributed vector search

#### Security & Access Control

- [ ] **Fine-grained Permissions**
  - Custom roles beyond owner/editor/reader
  - Permission inheritance
  - Temporary access grants
  - IP-based access control

- [ ] **Audit Logging**
  - Track all API operations
  - User action history
  - Security event logging
  - Compliance reporting

- [ ] **OAuth/SAML Integration**
  - OAuth 2.0 authentication
  - SAML SSO support
  - Identity provider integration
  - External authentication services

#### Integration

- [ ] **Webhooks**
  - Event notifications for embeddings changes
  - Project updates notifications
  - Configurable webhook endpoints
  - Retry logic and delivery guarantees

- [ ] **Export/Import**
  - Project export to standard formats
  - Bulk embedding export
  - Import from other vector databases
  - Migration utilities

- [ ] **SDK Support**
  - Python SDK
  - JavaScript/TypeScript SDK
  - Go SDK
  - CLI improvements

## Development Process

### Release Cycle

- **Minor versions** (0.x.0): New features, API additions
- **Patch versions** (0.0.x): Bug fixes, documentation updates
- **Major versions** (x.0.0): Breaking API changes (future)

### Feature Requests

To request a feature or suggest improvements:

1. Check existing issues on GitHub
2. Open a new issue with:
   - Clear description of the feature
   - Use cases and motivation
   - Proposed implementation (if any)
3. Engage in discussion with maintainers

### Contribution Guidelines

Contributions are welcome! See the main repository for:
- Development setup instructions
- Code style guidelines
- Testing requirements
- Pull request process

## Version History

### v0.1.0 (2026-02-08)
- Fix many things
- Add many things
- Still API v1 on the way to stable

### v0.0.1 (2024-12-10)
- Initial public release
- API v1 (work in progress)
- Core functionality implemented
- Docker support
- Project sharing
- Metadata validation

## Feedback

We value your feedback! Please share:

- **Feature requests** - What would make dhamps-vdb more useful?
- **Bug reports** - Help us improve quality
- **Use cases** - How are you using dhamps-vdb?
- **Documentation** - What needs clarification?

Open an issue on GitHub or contact the maintainers directly.

## Related Documentation

- [Getting Started](../../getting-started/)
- [API Documentation](../../api/)
- [Deployment Guide](../../deployment/)
- [Reference - Configuration](configuration/)
- [Reference - Database Schema](database-schema/)
- [GitHub Repository](https://github.com/mpilhlt/dhamps-vdb)
