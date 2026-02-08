---
title: "LLM Services"
weight: 5
---

# LLM Services

LLM Services configure embedding generation, defining models, dimensions, and API access.

## Architecture

embapi separates LLM services into two concepts:

### LLM Service Definitions

Reusable configuration templates owned by `_system` or users:

- **Purpose**: Provide standard configurations
- **Ownership**: `_system` (global) or individual users
- **Contents**: Endpoint, model, dimensions, API standard
- **API Keys**: Not stored (templates only)
- **Usage**: Templates for creating instances

### LLM Service Instances

User-specific configurations with encrypted API keys:

- **Purpose**: Actual service configurations users employ
- **Ownership**: Individual users
- **Contents**: Endpoint, model, dimensions, API key (encrypted)
- **Sharing**: Can be shared with other users
- **Projects**: Each project references exactly one instance

## System Definitions

### Available Definitions

The `_system` user provides default definitions:

| Handle | Model | Dimensions | API Standard |
|--------|-------|------------|--------------|
| openai-large | text-embedding-3-large | 3072 | openai |
| openai-small | text-embedding-3-small | 1536 | openai |
| cohere-v4 | embed-multilingual-v4.0 | 1536 | cohere |
| gemini-embedding-001 | text-embedding-004 | 3072 | gemini |

### Viewing System Definitions

```bash
GET /v1/llm-services/_system
Authorization: Bearer user_embapi_key
```

Returns list of available system definitions.

## Creating Instances

### From System Definition

Use a predefined system configuration:

```bash
PUT /v1/llm-services/alice/my-openai

{
  "definition_owner": "_system",
  "definition_handle": "openai-large",
  "description": "My OpenAI embeddings",
  "api_key_encrypted": "sk-proj-your-openai-key"
}
```

Inherits endpoint, model, dimensions from system definition.

### From User Definition

Reference a user-created definition:

```bash
PUT /v1/llm-services/alice/custom-instance

{
  "definition_owner": "alice",
  "definition_handle": "my-custom-config",
  "api_key_encrypted": "your-api-key"
}
```

### Standalone Instance

Create without a definition:

```bash
PUT /v1/llm-services/alice/standalone

{
  "endpoint": "https://api.openai.com/v1/embeddings",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072,
  "description": "Standalone OpenAI instance",
  "api_key_encrypted": "sk-proj-your-key"
}
```

All fields must be specified.

## Instance Properties

### Core Fields

- **instance_handle**: Unique identifier (3-20 characters)
- **owner**: User who owns the instance
- **endpoint**: API endpoint URL
- **api_standard**: Authentication mechanism (openai, cohere, gemini)
- **model**: Model identifier
- **dimensions**: Vector dimensionality
- **description**: Human-readable description (optional)
- **definition_id**: Reference to definition (optional)

### API Key Storage

- **Write-only**: Provided on create/update
- **Encrypted**: AES-256-GCM encryption
- **Never returned**: Not included in GET responses
- **Secure**: Cannot be retrieved after creation

## API Standards

### Supported Standards

| Standard | Key Method | Documentation |
|----------|------------|---------------|
| openai | Authorization: Bearer | [OpenAI Docs](https://platform.openai.com/docs/api-reference/embeddings) |
| cohere | Authorization: Bearer | [Cohere Docs](https://docs.cohere.com/reference/embed) |
| gemini | x-goog-api-key header | [Gemini Docs](https://ai.google.dev/gemini-api/docs/embeddings) |

### Creating API Standards

Admins can add new standards:

```bash
POST /v1/api-standards
Authorization: Bearer ADMIN_KEY

{
  "api_standard_handle": "custom",
  "description": "Custom LLM API",
  "key_method": "auth_bearer",
  "key_field": null
}
```

## Instance Management

### List Instances

List all accessible instances (owned + shared):

```bash
GET /v1/llm-services/alice
Authorization: Bearer alice_embapi_key
```

Returns instances where alice is owner or has been granted access.

### Get Instance Details

```bash
GET /v1/llm-services/alice/my-openai
Authorization: Bearer alice_embapi_key
```

Returns instance configuration (API key not included).

### Update Instance

```bash
PATCH /v1/llm-services/alice/my-openai

{
  "description": "Updated description",
  "api_key_encrypted": "new-api-key"
}
```

Only owner can update instances.

### Delete Instance

```bash
DELETE /v1/llm-services/alice/my-openai
Authorization: Bearer alice_embapi_key
```

**Constraints:**
- Cannot delete instance used by projects
- Delete projects first, then instance

## Instance Sharing

### Share with User

```bash
POST /v1/llm-services/alice/my-openai/share

{
  "share_with_handle": "bob",
  "role": "reader"
}
```

**Shared users can:**
- Use instance in their projects
- View instance configuration
- **Cannot:**
  - See API key
  - Modify instance
  - Delete instance

### Unshare from User

```bash
DELETE /v1/llm-services/alice/my-openai/share/bob
```

### List Shared Users

```bash
GET /v1/llm-services/alice/my-openai/shared-with
```

Only owner can view shared users.

## Instance References

### In Projects

Projects reference instances by owner and handle:

```json
{
  "project_handle": "my-project",
  "instance_owner": "alice",
  "instance_handle": "my-openai"
}
```

### In Embeddings

Embeddings reference instances by handle:

```json
{
  "text_id": "doc1",
  "instance_handle": "my-openai",
  "vector": [...]
}
```

The instance owner is inferred from the project.

### Shared Instance Format

When using shared instances:

**Own instance**: `"instance_handle": "my-openai"`
**Shared instance**: Reference via project configuration

## Encryption

### API Key Encryption

API keys are encrypted using AES-256-GCM:

**Encryption process:**
1. User provides plaintext API key
2. Server encrypts using `ENCRYPTION_KEY` from environment
3. Encrypted bytes stored in database
4. Key derivation: SHA-256 hash ensures 32-byte key

**Decryption:**
- Only occurs internally for LLM API calls
- Never exposed via API responses
- Requires same `ENCRYPTION_KEY`

### Security Notes

- **Encryption key**: Set via `ENCRYPTION_KEY` environment variable
- **Key loss**: Losing encryption key means losing access to API keys
- **Key rotation**: Not currently supported
- **Backup**: Back up encryption key securely

## LLM Processing

### Current Status

LLM processing (generating embeddings) is **not yet implemented**.

### Future Implementation

Planned features:
- Process text to generate embeddings
- Call external LLM APIs
- Store generated embeddings
- Batch processing support

### Current Workflow

Users must generate embeddings externally:

1. Generate embeddings using LLM API (OpenAI, Cohere, etc.)
2. Upload pre-generated embeddings to embapi
3. Use embapi for storage and similarity search

## Common Patterns

### Per-Environment Instances

```bash
# Development instance
PUT /v1/llm-services/alice/dev-embeddings
{
  "definition_owner": "_system",
  "definition_handle": "openai-small",
  "api_key_encrypted": "dev-api-key"
}

# Production instance
PUT /v1/llm-services/alice/prod-embeddings
{
  "definition_owner": "_system",
  "definition_handle": "openai-large",
  "api_key_encrypted": "prod-api-key"
}
```

### Team Shared Instance

```bash
# Owner creates instance
PUT /v1/llm-services/team-lead/team-embeddings
{
  "definition_owner": "_system",
  "definition_handle": "openai-large",
  "api_key_encrypted": "team-api-key"
}

# Share with team members
POST /v1/llm-services/team-lead/team-embeddings/share
{"share_with_handle": "member1", "role": "reader"}

POST /v1/llm-services/team-lead/team-embeddings/share
{"share_with_handle": "member2", "role": "reader"}

# Members use in their projects
POST /v1/projects/member1
{
  "project_handle": "my-project",
  "instance_owner": "team-lead",
  "instance_handle": "team-embeddings"
}
```

### Multi-Model Setup

```bash
# Large model for important documents
PUT /v1/llm-services/alice/high-quality
{
  "definition_owner": "_system",
  "definition_handle": "openai-large",
  "api_key_encrypted": "api-key"
}

# Small model for drafts
PUT /v1/llm-services/alice/fast-processing
{
  "definition_owner": "_system",
  "definition_handle": "openai-small",
  "api_key_encrypted": "api-key"
}
```

## Troubleshooting

### Cannot Create Instance

**Possible causes:**
- Instance handle already exists
- Referenced definition doesn't exist
- Missing required fields
- Invalid API standard

**Solutions:**
- Choose different handle
- Verify definition: `GET /v1/llm-services/_system`
- Include all required fields
- Use valid API standard: `GET /v1/api-standards`

### Cannot Use Instance in Project

**Possible causes:**
- Instance doesn't exist
- Instance not owned or shared with user
- Incorrect owner/handle reference

**Solutions:**
- Verify instance exists
- Check instance is accessible
- Confirm spelling of owner and handle

### Dimension Mismatch

**Error**: "dimension validation failed"

**Cause**: Embedding dimensions don't match instance

**Solutions:**
- Check instance dimensions
- Regenerate embeddings with correct model
- Create instance with correct dimensions

## Next Steps

- [Understand projects](projects/)
- [Learn about embeddings](embeddings/)
- [Explore instance management](../guides/instance-management/)
