---
title: "Instance Management Guide"
weight: 8
---

# Instance Management Guide

This guide explains how to create, configure, and share LLM service instances for generating and managing embeddings.

## Overview

LLM service instances define the configuration for connecting to embedding services (like OpenAI, Cohere, or Gemini). Each instance includes:
- API endpoint and credentials
- Model name and version
- Vector dimensions
- API standard (protocol)

Instances can be created from system templates, user-defined templates, or as standalone configurations. They can also be shared with other users for collaborative work.

## Instance Architecture

### System Definitions

The system provides pre-configured templates for common LLM services:

```bash
# List available system definitions (no auth required)
curl -X GET "https://api.example.com/v1/llm-service-definitions/_system"
```

**Default System Definitions:**
- `openai-large`: OpenAI text-embedding-3-large (3072 dimensions)
- `openai-small`: OpenAI text-embedding-3-small (1536 dimensions)
- `cohere-v4`: Cohere Embed v4 (1536 dimensions)
- `gemini-embedding-001`: Google Gemini embedding-001 (3072 dimensions, default size)

### User Instances

Users create instances for their own use. Instances contain:
- Configuration (endpoint, model, dimensions)
- Encrypted API keys (write-only, never returned)
- Optional reference to a definition template

## Creating LLM Service Instances

### Option 1: Standalone Instance

Create an instance by specifying all configuration fields:

```bash
curl -X PUT "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "my-openai",
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "description": "OpenAI large embedding model for research",
    "api_key_encrypted": "sk-proj-your-openai-api-key-here"
  }'
```

**Response:**
```json
{
  "instance_id": 123,
  "instance_handle": "my-openai",
  "owner": "alice",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072,
  "description": "OpenAI large embedding model for research"
}
```

**Note:** The `api_key_encrypted` field is not returned in the response for security reasons.

### Option 2: From System Definition

Create an instance based on a system template (only requires API key):

```bash
curl -X POST "https://api.example.com/v1/llm-services/alice/my-openai-instance" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "my-openai-instance",
    "definition_owner": "_system",
    "definition_handle": "openai-large",
    "api_key_encrypted": "sk-proj-your-openai-api-key-here"
  }'
```

This inherits configuration from the `_system/openai-large` definition and only requires you to provide your API key.

### Option 3: From User Definition

Users can create their own definitions as templates:

```bash
# Step 1: Create a custom definition
curl -X PUT "https://api.example.com/v1/llm-service-definitions/alice/my-custom-config" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "definition_handle": "my-custom-config",
    "endpoint": "https://custom-api.example.com/embeddings",
    "api_standard": "openai",
    "model": "custom-model-v2",
    "dimensions": 2048,
    "description": "Custom embedding service"
  }'

# Step 2: Create instance from that definition
curl -X POST "https://api.example.com/v1/llm-services/alice/custom-instance" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "custom-instance",
    "definition_owner": "alice",
    "definition_handle": "my-custom-config",
    "api_key_encrypted": "your-api-key-here"
  }'
```

## Managing Instances

### List Your Instances

Get all instances you own or have access to:

```bash
curl -X GET "https://api.example.com/v1/llm-services/alice" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**
```json
{
  "owned_instances": [
    {
      "instance_id": 123,
      "instance_handle": "my-openai",
      "endpoint": "https://api.openai.com/v1/embeddings",
      "model": "text-embedding-3-large",
      "dimensions": 3072
    },
    {
      "instance_id": 124,
      "instance_handle": "my-cohere",
      "endpoint": "https://api.cohere.ai/v1/embed",
      "model": "embed-english-v4.0",
      "dimensions": 1536
    }
  ],
  "shared_instances": [
    {
      "instance_id": 456,
      "instance_handle": "team-openai",
      "owner": "bob",
      "endpoint": "https://api.openai.com/v1/embeddings",
      "model": "text-embedding-3-large",
      "dimensions": 3072,
      "role": "reader"
    }
  ]
}
```

### Get Instance Details

Retrieve details for a specific instance:

```bash
curl -X GET "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key"
```

### Update Instance

Update instance configuration (owner only):

```bash
curl -X PATCH "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "api_key_encrypted": "sk-proj-new-api-key-here"
  }'
```

### Delete Instance

Delete an instance (owner only):

```bash
curl -X DELETE "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key"
```

**Note:** Cannot delete instances that are used by existing projects.

## API Key Encryption

### How Encryption Works

API keys are encrypted using AES-256-GCM encryption:

1. **Encryption Key Source**: `ENCRYPTION_KEY` environment variable
2. **Key Derivation**: SHA256 hash of the environment variable (ensures 32-byte key)
3. **Algorithm**: AES-256-GCM (authenticated encryption)
4. **Storage**: Encrypted bytes stored in `api_key_encrypted` column

### Setting Up Encryption

Add to your environment configuration:

```bash
# .env file
ENCRYPTION_KEY="your-secure-random-32-character-key-or-longer"
```

**Important Security Notes:**
- Keep this key secure and backed up
- Losing the key means losing access to encrypted API keys
- Use a strong, random string (32+ characters)
- Never commit the key to version control
- Rotate the key periodically (requires re-encrypting all API keys)

### API Key Security

API keys are **write-only** in the API:

```bash
# Upload API key (works)
curl -X PUT "https://api.example.com/v1/llm-services/alice/my-instance" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"api_key_encrypted": "sk-..."}'

# Retrieve instance (API key NOT returned)
curl -X GET "https://api.example.com/v1/llm-services/alice/my-instance" \
  -H "Authorization: Bearer alice_api_key"

# Response does NOT include api_key_encrypted field
{
  "instance_id": 123,
  "instance_handle": "my-instance",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "model": "text-embedding-3-large",
  "dimensions": 3072
  // No api_key_encrypted field!
}
```

This protects API keys from:
- Accidental exposure in logs
- Unauthorized access via shared instances
- Client-side data breaches

## Instance Sharing

### Share an Instance

Grant another user access to your instance:

```bash
curl -X POST "https://api.example.com/v1/llm-services/alice/my-openai/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "share_with_handle": "bob",
    "role": "reader"
  }'
```

**Roles:**
- `reader`: Can use the instance in projects (read-only)
- `editor`: Can use the instance (currently same as reader)
- `owner`: Full control (only one owner, the creator)

### List Shared Users

See who has access to your instance:

```bash
curl -X GET "https://api.example.com/v1/llm-services/alice/my-openai/shared-with" \
  -H "Authorization: Bearer alice_api_key"
```

**Response:**
```json
{
  "instance_handle": "my-openai",
  "owner": "alice",
  "shared_with": [
    {"user_handle": "bob", "role": "reader"},
    {"user_handle": "charlie", "role": "reader"}
  ]
}
```

### Unshare an Instance

Revoke access:

```bash
curl -X DELETE "https://api.example.com/v1/llm-services/alice/my-openai/share/bob" \
  -H "Authorization: Bearer alice_api_key"
```

### Using Shared Instances

Bob can reference Alice's shared instance in his projects:

```bash
# Bob creates a project using Alice's instance
curl -X PUT "https://api.example.com/v1/projects/bob/my-project" \
  -H "Authorization: Bearer bob_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "my-project",
    "description": "Using shared instance",
    "instance_owner": "alice",
    "instance_handle": "my-openai"
  }'

# Bob uploads embeddings using the shared instance
curl -X POST "https://api.example.com/v1/embeddings/bob/my-project" \
  -H "Authorization: Bearer bob_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "embeddings": [{
      "text_id": "doc001",
      "instance_handle": "alice/my-openai",
      "vector": [0.1, 0.2, ...],
      "vector_dim": 3072
    }]
  }'
```

**Important:** Bob can use the instance but cannot see Alice's API key.

## Instance Sharing Patterns

### Team Shared Instance

A team lead creates and shares an instance for the team:

```bash
# Team lead creates instance
curl -X PUT "https://api.example.com/v1/llm-services/team_lead/team-openai" \
  -H "Authorization: Bearer team_lead_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "team-openai",
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "api_key_encrypted": "sk-proj-team-api-key"
  }'

# Share with team members
for member in alice bob charlie; do
  curl -X POST "https://api.example.com/v1/llm-services/team_lead/team-openai/share" \
    -H "Authorization: Bearer team_lead_api_key" \
    -H "Content-Type: application/json" \
    -d "{\"share_with_handle\": \"$member\", \"role\": \"reader\"}"
done
```

### Organization-Wide Instance

Create a shared instance for organization-wide use:

```bash
# Organization admin creates instance
curl -X PUT "https://api.example.com/v1/llm-services/org_admin/org-embeddings" \
  -H "Authorization: Bearer org_admin_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "org-embeddings",
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "description": "Organization-wide embedding service",
    "api_key_encrypted": "sk-proj-org-api-key"
  }'
```

### Per-Project Instance

Each project maintainer creates their own instance:

```bash
# Alice creates her own instance for her project
curl -X PUT "https://api.example.com/v1/llm-services/alice/research-embeddings" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "research-embeddings",
    "endpoint": "https://api.openai.com/v1/embeddings",
    "api_standard": "openai",
    "model": "text-embedding-3-large",
    "dimensions": 3072,
    "api_key_encrypted": "sk-proj-alice-research-key"
  }'
```

## Common Configurations

### OpenAI Configuration

```json
{
  "instance_handle": "openai-large",
  "endpoint": "https://api.openai.com/v1/embeddings",
  "api_standard": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072,
  "api_key_encrypted": "sk-proj-..."
}
```

### Cohere Configuration

```json
{
  "instance_handle": "cohere-english",
  "endpoint": "https://api.cohere.ai/v1/embed",
  "api_standard": "cohere",
  "model": "embed-english-v4.0",
  "dimensions": 1536,
  "api_key_encrypted": "your-cohere-api-key"
}
```

### Google Gemini Configuration

```json
{
  "instance_handle": "gemini-embedding",
  "endpoint": "https://generativelanguage.googleapis.com/v1beta/models/embedding-001:embedContent",
  "api_standard": "gemini",
  "model": "embedding-001",
  "dimensions": 3072,
  "api_key_encrypted": "your-gemini-api-key"
}
```

### Custom/Self-Hosted Service

```json
{
  "instance_handle": "custom-service",
  "endpoint": "https://custom-api.example.com/v1/embeddings",
  "api_standard": "openai",
  "model": "custom-model-v2",
  "dimensions": 2048,
  "description": "Self-hosted embedding service",
  "api_key_encrypted": "custom-auth-token"
}
```

## Projects and Instances

### 1:1 Relationship

Each project must reference exactly one instance:

```bash
curl -X PUT "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "project_handle": "my-project",
    "description": "Project with instance",
    "instance_id": 123
  }'
```

### Changing Instance

Update a project to use a different instance:

```bash
curl -X PATCH "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"instance_id": 456}'
```

**Note:** Only switch to instances with matching dimensions, or you'll get validation errors on future uploads.

### Finding Instance for Project

```bash
curl -X GET "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key"
```

The response includes the `instance_id` field.

## Best Practices

### 1. Use System Definitions

Start with system definitions for common services:

```bash
# Easiest approach
curl -X POST "https://api.example.com/v1/llm-services/alice/my-instance" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "my-instance",
    "definition_owner": "_system",
    "definition_handle": "openai-large",
    "api_key_encrypted": "sk-..."
  }'
```

### 2. Descriptive Instance Names

Use clear, descriptive names:

```bash
# Good names
"research-openai-large"
"prod-cohere-english"
"test-gemini-embedding"

# Avoid generic names
"instance1"
"my-instance"
"test"
```

### 3. Separate Production and Development

Create separate instances for different environments:

```bash
# Development instance
curl -X PUT "https://api.example.com/v1/llm-services/alice/dev-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"instance_handle": "dev-openai", ...}'

# Production instance
curl -X PUT "https://api.example.com/v1/llm-services/alice/prod-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"instance_handle": "prod-openai", ...}'
```

### 4. Document Instance Purpose

Use the description field:

```bash
curl -X PUT "https://api.example.com/v1/llm-services/alice/team-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "instance_handle": "team-openai",
    "description": "Shared OpenAI instance for research team. Contact alice@example.com for access.",
    ...
  }'
```

### 5. Regular Key Rotation

Periodically update API keys:

```bash
curl -X PATCH "https://api.example.com/v1/llm-services/alice/my-openai" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"api_key_encrypted": "sk-proj-new-key-here"}'
```

### 6. Monitor Instance Usage

Track which projects use each instance to avoid deleting in-use instances.

## Troubleshooting

### Cannot Delete Instance

**Error:** "Instance is in use by existing projects"

**Solution:** Delete or update projects using this instance first.

### Dimension Mismatch

**Error:** "vector dimension mismatch"

**Solution:** Ensure embeddings match the instance's configured dimensions.

### API Key Not Working

**Problem:** Embeddings uploads fail with authentication errors

**Solution:** 
1. Verify API key is correct
2. Check API key permissions with the LLM provider
3. Update the API key in the instance

### Cannot Access Shared Instance

**Problem:** Getting "Instance not found" errors

**Solution:** Verify you've been granted access. Contact the instance owner.

## Related Documentation

- [RAG Workflow Guide](./rag-workflow.md) - Complete RAG implementation
- [Project Sharing Guide](./project-sharing.md) - Share projects with users
- [Batch Operations Guide](./batch-operations.md) - Upload embeddings efficiently

## Security Summary

1. **API keys are encrypted** at rest using AES-256-GCM
2. **API keys are never returned** via GET requests
3. **Shared users cannot see API keys** (write-only field)
4. **Encryption key must be secured** (loss means cannot decrypt keys)
5. **Regular key rotation recommended** for production use
