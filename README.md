# dhamps-vdb
Vector Database for the DH at Max Planck Society initiative

[![Go Report Card](https://goreportcard.com/badge/github.com/mpilhlt/dhamps-vdb?style=flat-square)](https://goreportcard.com/report/github.com/mpilhlt/dhamps-vdb) [![Release](https://img.shields.io/github/v/release/mpilhlt/dhamps-vdb.svg?style=flat-square&include_prereleases)](https://github.com/mpilhlt/dhamps-vdb/releases/latest)

<!--
[![Go Reference](https://pkg.go.dev/badge/github.com/mpilhlt/dhamps-vdb.svg)](https://pkg.go.dev/github.com/mpilhlt/dhamps-vdb)
-->

## Introduction

This is an application serving an API to handle embeddings. It stores embeddings in a PostgreSQL backend and uses its vector support, but allows you to manage different users, projects, and LLM configurations via a simple Restful API.

The typical use case is as a component of a Retrieval Augmented Generation (RAG) workflow: You create embeddings for a collection of text snippets and upload them to this API. For each text snippet, you upload a text identifier, the embeddings vector and, optionally, metadata or the text itself. Then, you can

- `GET` the most similar texts for a text that is already in the database by specifying the text's identifier in a URL
- `POST` raw embeddings to find similar texts without storing the query embeddings in the database

In both cases, the service returns a list of text identifiers along with their similarity scores that you can then use in your own processing, perhaps based on other means of providing the respective texts.

## Features

- OpenAPI documentation
- Supports different embeddings configurations (e.g. dimensions)
- Rights management (authentication via API token)
- Automatic validation of embeddings and metadata

## Getting started

### Docker Deployment (Recommended)

The easiest way to run dhamps-vdb is using Docker. See [DOCKER.md](./DOCKER.md) for comprehensive Docker deployment instructions.

Quick start with Docker:

```bash
# Automated setup (generates secure keys)
./docker-setup.sh

# Start with docker-compose (includes PostgreSQL with pgvector)
docker-compose up -d

# Access the API
curl http://localhost:8880/docs
```

### Compiling from Source

For **compiling**, you should run `go get ./... ; sqlc generate --no-remote ; go build -o build/dhamps-vdb main.go` (or in place of the last command you can also run it directly with `go run main.go`).

#### Run tests

Actual (mostly integration) tests are run like this:

```bash
$> systemctl --user start podman.socket
$> export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
$> go test -v ./...
```

These tests do not contact any separately installed/launched backend, and instead have a container managed by the testing itself (via [testcontainers](https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/)).

### Running

For **running**, you need to set an admin key and a couple of other variables. Have a look in [options.go](./internal/models/options.go) for the full list and documentation.

You can specify them as command line options, with environment values or in an `.env` file. This last option is recommended because this way, the sensitive information will not be in the command history; but you have to take care of this file's security. For instance, it will/should not be synced to your versioning platform, it is thus listed in `.gitignore` by default.

If you authenticate with this key, you can create users by `POST`ing to the `/v1/users` endpoint. (The response to user creation will contain an API key for the new user. **Keep it safe, it is not stored anywhere and cannot be recovered.**) Then, either the admin or the user can create projects, llm services and finally post embeddings and get similar elements.

When launching, the application checks and migrates the database schema to the appropriate version if possible. It presupposes however, that a suitable database and user (with appropriate privileges) have been created beforehand. SQL commands to prepare the database are listed below. For other ways of running, e.g. for tests or with a container instead of an external database, see below as well.

#### Run with local container

A local container with a pg_vector-enabled postgresql can be run like this:

```bash
$> podman run -p 8888:5432 -e POSTGRES_PASSWORD=password pgvector/pgvector:0.7.4-pg16
```

But be aware that the filesystem is not persisted if you run it like this. That means that when you stop and remove the container, you will have to repeat the following database setup when you run it again later on. (And of course any data you may have saved inside the container is lost, too.)

You can connect to it from a second terminal like so:

```bash
$> psql -p 8888 -h localhost -U postgres -d postgres
```

And then set up the database like this:

```sql
postgres=# CREATE DATABASE my_vectors;
postgres=# CREATE USER my_user WITH PASSWORD 'my-password';
postgres=# GRANT ALL PRIVILEGES ON DATABASE "my_vectors" to my_user;
postgres=# \c my_vectors
postgres=# GRANT ALL ON SCHEMA public TO my_user;
postgres=# CREATE EXTENSION IF NOT EXISTS vector;
```

### Client Authentication

Clients should communicate the API key in the `Authorization` header with a `Bearer` prefix, e.g. `Bearer 024v2013621509245f2e24`. Most operations can only be done by the (admin or the) owner of the resource in question.

**Project Sharing**: Projects can be shared with specific users for read or edit access. See the [Project Sharing](#project-sharing) section below for details on sharing projects with individual users.

**Public Access**: Projects can also be made publicly accessible (allowing unauthenticated read access to embeddings and similars) by setting the `public_read` field to `true` when creating or updating the project. See [docs/PUBLIC_ACCESS.md](./docs/PUBLIC_ACCESS.md) for details.

## Project Sharing

Projects can be shared with other users to enable collaboration. The project owner can grant two levels of access:

- **reader**: Read-only access to embeddings and similar documents
- **editor**: Read and write access to embeddings (can add/modify/delete embeddings)

### Sharing During Project Creation

When creating a project, you can specify users to share with using the `shared_with` field:

```json
{
  "project_handle": "my-project",
  "description": "A collaborative project",
  "instance_owner": "alice",
  "instance_handle": "embedding1",
  "shared_with": [
    {
      "user_handle": "bob",
      "role": "reader"
    },
    {
      "user_handle": "charlie",
      "role": "editor"
    }
  ]
}
```

### Managing Sharing After Creation

The project owner can manage sharing through dedicated endpoints:

**Share a project with a user:**

```bash
POST /v1/projects/{owner}/{project}/share
{
  "share_with_handle": "bob",
  "role": "reader"
}
```

**Unshare a project from a user:**

```bash
DELETE /v1/projects/{owner}/{project}/share/{user_handle}
```

**List users a project is shared with:**

```bash
GET /v1/projects/{owner}/{project}/shared-with
```

Only the project owner can view the list of shared users and manage sharing. Users who have been granted access to a project cannot see which other users also have access.

## Project Ownership Transfer

The project owner can transfer ownership of a project to another user. This is useful when:
- A project maintainer is leaving and wants to hand over control
- Organizational changes require reassigning project ownership
- Consolidating projects under a different user account

**Transfer ownership:**

```bash
POST /v1/projects/{owner}/{project}/transfer-ownership
{
  "new_owner_handle": "new_owner"
}
```

**Important notes:**
- Only the current owner can transfer ownership
- The new owner must be an existing user
- The new owner cannot already have a project with the same handle
- After transfer, the old owner will lose all access to the project
- If the new owner was previously shared with the project, their sharing role will be upgraded to owner
- All embeddings and other project data remain intact during transfer

**Example:**

```bash
# Alice transfers her project to Bob
curl -X POST "https://api.example.com/v1/projects/alice/my-project/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "new_owner_handle": "bob"
  }'

# After transfer, the project is accessible at /v1/projects/bob/my-project
```

### Shared User Access

Once a project is shared with a user, they can:
- View project metadata
- Retrieve embeddings (reader and editor)
- Search for similar documents (reader and editor)
- Add, modify, or delete embeddings (editor only)

Shared users **cannot**:
- Delete the project
- Change project settings
- Manage sharing (add/remove other users)
- View the list of other users the project is shared with

## Data Validation

The API provides automatic validation to ensure data quality and consistency:

### Embeddings Dimension Validation

When uploading embeddings, the system automatically validates:

1. **Vector dimension consistency**: The `vector_dim` field in your embeddings must match the `dimensions` configured in the LLM service being used.
2. **Vector length verification**: The actual number of elements in the `vector` array must match the declared `vector_dim`.

If validation fails, you'll receive a `400 Bad Request` response with a detailed error message explaining the mismatch.

**Example error response:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "dimension validation failed: vector dimension mismatch: embedding declares 3072 dimensions but LLM service 'openai-large' expects 5 dimensions"
}
```

### Similarity Query Dimension Filtering

When querying for similar embeddings, the system automatically filters results to only include embeddings with matching dimensions. This ensures that similarity comparisons are only made between vectors of the same dimensionality, preventing invalid comparisons.

The similarity queries enforce:
- Only embeddings with matching `vector_dim` are compared
- Only embeddings from the same project are considered
- Vector similarity is calculated using cosine distance on compatible dimensions

### Metadata Schema Validation

Projects can optionally define a JSON Schema to validate metadata attached to embeddings. This ensures that all embeddings in a project have consistent, well-structured metadata.

#### Defining a Metadata Schema

Include a `metadataScheme` field when creating or updating a project with a valid JSON Schema:

```json
{
  "project_handle": "my-project",
  "description": "Project with metadata validation",
  "metadataScheme": "{\"type\":\"object\",\"properties\":{\"author\":{\"type\":\"string\"},\"year\":{\"type\":\"integer\"}},\"required\":[\"author\"]}"
}
```

The schema above requires an `author` field (string) and allows an optional `year` field (integer).

#### Schema Validation on Upload

When uploading embeddings to a project with a metadata schema, the API validates each embedding's metadata against the schema. If validation fails, you'll receive a detailed error message:

**Example error response:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "metadata validation failed for text_id 'doc123': metadata validation failed:\n  - author: author is required"
}
```

### Admin Sanity Check

Administrators can verify database integrity using the `/v1/admin/sanity-check` endpoint. This endpoint:

- Checks all embeddings have dimensions matching their LLM service
- Validates all metadata against project schemas (if defined)
- Reports issues and warnings in a structured format

**Example sanity check request:**

```bash
curl -X GET http://localhost:8080/v1/admin/sanity-check \
  -H "Authorization: ******"
```

**Example response:**

```json
{
  "status": "PASSED",
  "total_projects": 5,
  "issues_count": 0,
  "warnings_count": 1,
  "warnings": [
    "Project alice/project1 has 100 embeddings but no metadata schema defined"
  ]
}
```

Status values:
- `PASSED`: No issues or warnings found
- `WARNING`: No critical issues, but warnings exist
- `FAILED`: Validation issues found that need attention

#### Example Metadata Schemas

**Simple schema with required fields:**

```json
{
  "type": "object",
  "properties": {
    "author": {"type": "string"},
    "year": {"type": "integer"},
    "language": {"type": "string"}
  },
  "required": ["author", "year"]
}
```

**Schema with nested objects:**

```json
{
  "type": "object",
  "properties": {
    "author": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "id": {"type": "string"}
      },
      "required": ["name"]
    },
    "publication": {
      "type": "object",
      "properties": {
        "year": {"type": "integer"},
        "title": {"type": "string"}
      }
    }
  },
  "required": ["author"]
}
```

**Schema with enums and constraints:**

```json
{
  "type": "object",
  "properties": {
    "genre": {
      "type": "string",
      "enum": ["fiction", "non-fiction", "poetry", "drama"]
    },
    "rating": {
      "type": "number",
      "minimum": 0,
      "maximum": 5
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"}
    }
  }
}
```

For more information on JSON Schema syntax, see [json-schema.org](https://json-schema.org/).

## API documentation

### API Versioning

We are at `v1`. The first path component to all the endpoints (except for the OpenAPI file) is the version number, e.g. `POST https://<hostname>/v1/embeddings/<user>/<project>`.

### Endpoints

In the following table, the version number is skipped for readibility reasons. Nevertheless, it is the first component of all these endpoints.

For a more detailed, and always up-to-date documentation of the endpoints, including query parameters, return values and data schemes, see the automatically generated live OpenAPI document at `/openapi.yaml` or the browsable version at `/docs`.

| Endpoint | Method | Description | Allowed Users |
| -------- | ------ | ----------- | ------------- |
| /admin/footgun | GET | Reset Database: Remove all records from database and reset serials/counters | admin |
| /admin/sanity-check | GET | Verify all data in database conforms to schemas and dimension requirements | admin |
| /users | GET | Get all users (list of handles) registered with the Db | admin |
| /users | POST | Register a new user with the Db | admin |
| /users/\<username\> | GET | Get information about user \<username\> | admin, \<username\> |
| /users/\<username\> | PUT | Register a new user with the Db | admin |
| /users/\<username\> | DELETE | Delete a user and all their projects/llm services from the Db | admin, \<username\> |
| /projects/\<username\> | GET | Get all projects (objects) for user \<username\> | admin, \<username\> |
| /projects/\<username\> | POST | Register a new project for user \<username\> | admin, \<username\> |
| /projects/\<username\>/\<projectname\> | GET | Get project information for \<username\>'s project \<projectname\> | admin, \<username\>, authorized readers |
| /projects/\<username\>/\<projectname\> | PUT | Register a new project calles \<projectname\> for user \<username\> | admin, \<username\> |
| /projects/\<username\>/\<projectname\> | DELETE | Delete \<username\>'s project \<projectname\> | admin, \<username\> |
| /projects/\<username\>/\<projectname\>/share | POST | Share \<username\>'s project \<projectname\> with another user | admin, \<username\> |
| /projects/\<username\>/\<projectname\>/share/\<shared_user\> | DELETE | Unshare \<username\>'s project \<projectname\> from \<shared_user\> | admin, \<username\> |
| /projects/\<username\>/\<projectname\>/shared-with | GET | Get list of users \<username\>'s project \<projectname\> is shared with | admin, \<username\> |
| /llm-services/\<username\> | GET | Get all LLM services (objects) for user \<username\> | admin, \<username\> |
| /llm-services/\<username\> | POST | Register a new LLM service for user \<username\> | admin, \<username\> |
| /llm-services/\<username\>/<llm_servicename> | GET | Get information about LLM service <llm_servicename> of user \<username\> | admin, \<username\> |
| /llm-services/\<username\>/<llm_servicename> | PUT | Register a new LLM service called <llm_servicename> for user \<username\> | admin, \<username\> |
| /llm-services/\<username\>/<llm_servicename> | DELETE | Delete \<username\>'s LLM service <llm_servicename> | admin, \<username\> |
| /api-standards | GET | Get all defined API standards* | public |
| /api-standards | POST | Register a new API standard* | admin |
| /api-standards/\<standardname\> | GET | Get information about API standard* \<standardname\> | public |
| /api-standards/\<standardname\> | PUT | Register a new API standard* \<standardname\> | admin |
| /api-standards/\<standardname\> | DELETE | Delete API standard* \<standardname\> | admin |
| /embeddings/\<username\>/\<projectname\> | GET | Get all embeddings for \<username\>'s project \<projectname\> (use `limit` and `offset` for paging) | admin, \<username\>, authorized readers |
| /embeddings/\<username\>/\<projectname\> | POST | Register a new record with an embeddings vector for \<username\>'s project \<projectname\> | admin, \<username\> |
| /embeddings/\<username\>/\<projectname\> | DELETE | Delete ***all*** embeddings for \<username\>'s project \<projectname\> | admin, \<username\> |
| /embeddings/\<username\>/\<projectname\>/\<identifier\> | GET | Get embeddings and other information about text \<identifier\> from \<username\>'s project \<projectname\> | admin, \<username\>, authorized readers |
| /embeddings/\<username\>/\<projectname\>/\<identifier\> | DELETE | Delete record \<identifier\> from \<username\>'s project \<projectname\> | admin, \<username\> |
| /similars/\<username\>/\<projectname\>/\<identifier\> | GET | Get a list of documents similar to the text \<identifier\> in \<username\>'s project \<projectname\>, with similarity scores | admin, \<username\>, authorized readers |
| /similars/\<username\>/\<projectname\> | POST | Find similar documents using raw embeddings without storing them, with similarity scores | admin, \<username\>, authorized readers |

\* API standards are definitions of how to access an LLM Service: API endpoints, authentication mechanism etc. They are referred to from LLM Service definitions. When LLM Processing will be attempted, this is what will be implemented. Examples are the Cohere Embed API, Version 2, as documented in <https://docs.cohere.com/reference/embed>, or the OpenAI Embeddings API, Version 1, as documented in <https://platform.openai.com/docs/api-reference/embeddings>. You can find these examples in the [valid_api_standard\*.json](./testdata/) files in the `testdata` directory.

### Similarity Search

The API provides two endpoints for finding similar documents using vector similarity:

#### GET Similar Documents (from stored embeddings)

Find documents similar to an already-stored document by its identifier:

```bash
GET /v1/similars/{username}/{projectname}/{identifier}
```

**Query Parameters:**
- `count` (optional, default: 10, max: 200): Number of similar documents to return
- `threshold` (optional, default: 0.5, range: 0-1): Minimum similarity score threshold
- `limit` (optional, default: 10, max: 200): Maximum number of results to return
- `offset` (optional, default: 0): Pagination offset
- `metadata_path` (optional): Filter results by metadata field path (must be used with `metadata_value`)
- `metadata_value` (optional): Metadata value to exclude from results (must be used with `metadata_path`)

**Example:**

```bash
curl -X GET "https://<hostname>/v1/similars/alice/myproject/doc123?count=5&threshold=0.7" \
  -H "Authorization: Bearer <vdb_key>"
```

#### POST Similar Documents (from raw embeddings)

Find similar documents by submitting a raw embedding vector without storing it in the database:

```bash
POST /v1/similars/{username}/{projectname}
```

**Request Body:**

```json
{
  "vector": [0.1, 0.2, 0.3, ...]
}
```

The vector must be an array of float32 values with dimensions matching the project's LLM service instance configuration.

**Query Parameters:** Same as GET endpoint above.

**Example:**

```bash
curl -X POST "https://<hostname>/v1/similars/alice/myproject?count=10&threshold=0.8" \
  -H "Authorization: Bearer <vdb_key>" \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [-0.020850, 0.018522, 0.053270, 0.071384, 0.020003]
  }'
```

#### Response Format

Both similarity endpoints return the same response format with document identifiers and their similarity scores:

```json
{
  "$schema": "http://localhost:8080/schemas/SimilarResponseBody.json",
  "user_handle": "alice",
  "project_handle": "myproject",
  "results": [
    {
      "id": "doc456",
      "similarity": 0.95
    },
    {
      "id": "doc789",
      "similarity": 0.87
    },
    {
      "id": "doc321",
      "similarity": 0.82
    }
  ]
}
```

**Response Fields:**

- `user_handle`: The project owner's username
- `project_handle`: The project identifier
- `results`: Array of similar documents, ordered by similarity (highest first)
  - `id`: Document identifier
  - `similarity`: Cosine similarity score (0-1, where 1 is most similar)

#### Dimension Validation

When using the POST endpoint, the API automatically validates that:
1. The project has an associated LLM service instance
2. The submitted vector dimensions match the LLM service instance's configured dimensions
3. If dimensions don't match, a `400 Bad Request` error is returned with details

**Example error:**

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "vector dimension mismatch: expected 1536 dimensions, got 768"
}
```

#### Metadata Filtering

Both endpoints support filtering results by metadata fields. The filter uses negative matching (excludes documents where the metadata field matches the specified value):

```bash
# Exclude documents with author="John Doe"
curl -X GET "https://<hostname>/v1/similars/alice/myproject/doc123?metadata_path=author&metadata_value=John%20Doe" \
  -H "Authorization: Bearer <vdb_key>"
```

This is useful for excluding documents from the same source, author, or category when finding similar content.

### Partial Updates with PATCH

For resources that support both GET and PUT operations, PATCH requests are automatically available for partial updates. You only need to include the fields you want to change. This is particularly useful for updating single fields without having to provide all resource data.

#### **Supported resources:**

- Users: `/v1/users/{username}`
- Projects: `/v1/projects/{username}/{projectname}`
- LLM Services: `/v1/llm-services/{username}/{llm_servicename}`
- API Standards: `/v1/api-standards/{standardname}`

#### **Example: Enable world-readable access for a project**

```bash
curl -X PATCH https://<hostname>/v1/projects/alice/myproject \
  -H "Authorization: Bearer <vdb_key>" \
  -H "Content-Type: application/json" \
  -d '{"shared_with": ["*"]}'
```

#### **Example: Update project description**

```bash
curl -X PATCH https://<hostname>/v1/projects/alice/myproject \
  -H "Authorization: Bearer <vdb_key>" \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated project description"}'
```

The PATCH endpoint merges your changes with the existing resource data retrieved via GET, then applies the update via PUT.

## Code creation and structure

This API is programmed in go and uses the [huma](https://huma.rocks/) framework with go's stock `http.ServeMux()` routing.

Some initial code and some later bugfixes have been developed in dialogue with [ChatGPT](./docs/ChatGPT.md). After manual inspection and correction, this is the project structure:

```default
dhamps-vdb/
├── .env           // This is not distributed because it's in .gitignore
├── .gitignore
├── .repopackignore
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── main.go
├── repopack-output.xml
├── repopack.config.json
├── sqlc.yaml
├── template.env
├── api/
│   └── openapi.yml          // OpenAPI spec file, not up to date
├── docs/
│   └── ChatGPT.md           // Code as suggested by ChatGPT (GPT4 turbo and GPT4o) on 2024-06-09
├── internal/
│   ├── auth/
│   │   └── authenticate.go
│   ├── database/
│   │   ├── migrations/
│   │   │   ├── 001_create_initial_scheme.sql
│   │   │   ├── 002_create_emb_index.sql
│   │   │   ├── tern.conf        // This is not distributed because it's in .gitignore
│   │   │   └── tern.conf.tpl
│   │   ├── queries/
│   │   │   └── queries.sql
│   │   ├── database.go
│   │   ├── db.go                // This is auto-generated by sqlc
│   │   ├── migrations.go
│   │   ├── models.go            // This is auto-generated by sqlc
│   │   └── queries.sql.go       // This is auto-generated by sqlc
│   ├── handlers/
│   │   ├── admin.go
│   │   ├── admin_test.go
│   │   ├── api_standards.go
│   │   ├── api_standards_test.go
│   │   ├── embeddings.go
│   │   ├── embeddings_test.go
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   ├── llm_processes.go
│   │   ├── instances.go
│   │   ├── llm_services_test.go
│   │   ├── projects.go
│   │   ├── projects_test.go
│   │   ├── similars.go
│   │   ├── users.go
│   │   └── users_test.go
│   └── models/
│       ├── admin.go
│       ├── api_standards.go
│       ├── embeddings.go
│       ├── llm_processes.go
│       ├── instances.go
│       ├── options.go
│       ├── projects.go
│       ├── similars.go
│       └── users.go
├── testdata/
│   ├── postgres/
│   │   ├── enable-vector.sql
│   │   └── users.yml
│   ├── invalid_api_standard.json
│   ├── invalid_embeddings.json
│   ├── ...
│   ├── valid_api_standard_cohere_v2.json
│   ├── valid_api_standard_ollama.json
│   ├── valid_api_standard_openai_v1.json
│   ├── valid_embeddings.json
│   ├── valid_llm_service_cohere-multilingual-3.json
│   ├── valid_llm_service_openai-large-full.json
│   ├── ...
│   └── valid_user.json
└── web/                      // Web resources for the html response (in the future)
```

## Roadmap

- [ ] Revisit all **documentation**
  - [ ] Add documentation for metadata filtering of similars (the GET query parameters are called `metadata_path` and `metadata_value` as in: `https://xy.org/vdb-api/v1/similars/sal/sal-openai-large/https%3A%2F%2Fid.myproject.net%2Ftexts%2FW0011%3A1.3.1.3.1?threshold=0.7&limit=5&metadata_path=author_id&metadata_value=A0083`)
- [ ] Network connectivity
  - [ ] Implement and make consequent use of **max_idle** (5), **max_concurr** (5), **timeouts**, and **cancellations**
  - [ ] **Concurrency** (leaky bucket approach) and **Rate limiting** (redis, sliding window, implement headers). See  <https://huma.rocks/features/request-limits/> for limiting.
  - [ ] Caching
- [ ] Add API standards for anthropic, mistral, llama.cpp, ollama, vllm, llmstudio
- [ ] HTML UI?
- [ ] Allow to request verbose information even in list outputs (with a verbose=yes query parameter?)
- [ ] Add possiblity to use PATCH method to change existing resources
- [ ] Proper logging with `--verbose` and `--quiet` modes
- [x] Transfer of projects from one owner to another as new operation
- [x] Make sure pagination is supported consistently
- [x] Dockerization
- [x] Prevent acceptance of requests as user "_system"
- [x] Tests
  - [x] When testing, check cleanup by adding a new query/function to see if all tables are empty
  - [x] Make sure input is validated consistently
- [x] Catch POST to existing resources
- [x] User authentication & restrictions on some API calls
- [x] API versioning
- [x] better **options** handling (<https://huma.rocks/features/cli/>)
- [x] handle **metadata**
  - [x] Validation with metadata schema
- [x] Allow to filter similar passages by metadata field (so as to exclude e.g. documents from the same author)
- [x] Use **transactions** (most importantly, when an action requires several queries, e.g. projects being added and then linked to several read-authorized users)
- [x] Always use specific error messages
- [x] Add project sharing/unsharing functions & API paths
- [x] Add definition creation/listing/deletion functions & paths
- [x] Add mechanism to allow anonymous/public reading access to embeddings (via `"*"` in `shared_with`)

## License

[MIT License](./LICENSE)

## Versions

- 2026-02-08 **v0.1.0**: Fix many things, add many things, still API v1 on the way to stable...
- 2024-12-10 **v0.0.1**: Initial public release (still work in progress) of API v1
