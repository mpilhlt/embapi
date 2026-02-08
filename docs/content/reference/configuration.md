---
title: "Configuration Reference"
weight: 1
---

# Configuration Reference

Complete reference for configuring dhamps-vdb. This guide consolidates all configuration options, including environment variables, command-line flags, and Docker configuration.

## Overview

dhamps-vdb is configured through a combination of:

1. **Environment variables** (recommended)
2. **Command-line flags** (overrides environment variables)
3. **`.env` files** (for Docker and local development)

Configuration is loaded in the following priority order (highest to lowest):

1. Command-line flags
2. Environment variables
3. `.env` file values
4. Default values from `options.go`

## Configuration Options

### Service Configuration

Options for controlling the API service behavior.

| Option | Environment Variable | CLI Flag | Type | Default | Required | Description |
|--------|---------------------|----------|------|---------|----------|-------------|
| Debug | `SERVICE_DEBUG` | `-d`, `--debug` | Boolean | `true` | No | Enable verbose debug logging |
| Host | `SERVICE_HOST` | `--host` | String | `localhost` | No | Hostname or IP to bind to |
| Port | `SERVICE_PORT` | `-p`, `--port` | Integer | `8880` | No | Port number to listen on |

**Debug Mode:**
- `true` - Detailed logs including SQL queries, request details, internal operations
- `false` - Minimal logs for production use

**Host Configuration:**
- `localhost` - Local development only
- `0.0.0.0` - Listen on all interfaces (required for Docker)
- Specific IP - Bind to particular network interface

**Port Configuration:**
- Default: `8880`
- Ports below 1024 require elevated privileges
- Ensure port is not in use by another service

### Database Configuration

Options for connecting to the PostgreSQL database with pgvector extension.

| Option | Environment Variable | CLI Flag | Type | Default | Required | Description |
|--------|---------------------|----------|------|---------|----------|-------------|
| DB Host | `SERVICE_DBHOST` | `--db-host` | String | `localhost` | Yes | PostgreSQL server hostname |
| DB Port | `SERVICE_DBPORT` | `--db-port` | Integer | `5432` | No | PostgreSQL server port |
| DB User | `SERVICE_DBUSER` | `--db-user` | String | `postgres` | Yes | Database username |
| DB Password | `SERVICE_DBPASSWORD` | `--db-password` | String | `password` | Yes | Database password |
| DB Name | `SERVICE_DBNAME` | `--db-name` | String | `postgres` | Yes | Database name |

**Database Requirements:**
- PostgreSQL 12+ (16+ recommended)
- pgvector extension installed and enabled
- User must have CREATE, ALTER, DROP, INSERT, SELECT, UPDATE, DELETE privileges
- Database must exist before starting dhamps-vdb

**Common Database Hosts:**
- `localhost` - Local PostgreSQL instance
- `postgres` - Docker Compose service name
- `db.example.com` - Remote database server
- IP address - Direct connection to database

### Security Configuration

Critical security settings for authentication and encryption.

| Option | Environment Variable | CLI Flag | Type | Default | Required | Description |
|--------|---------------------|----------|------|---------|----------|-------------|
| Admin Key | `SERVICE_ADMINKEY` | `--admin-key` | String | - | **Yes** | Master API key for admin operations |
| Encryption Key | `ENCRYPTION_KEY` | - | String | - | **Yes** | AES-256 key for API key encryption |

**Admin Key (`SERVICE_ADMINKEY`):**
- Master API key with full administrative privileges
- Used to create users and manage global resources
- Generate with: `openssl rand -base64 32`
- Must be kept secure and rotated regularly
- Transmitted via `Authorization: Bearer` header

**Encryption Key (`ENCRYPTION_KEY`):**
- Used to encrypt user API keys in the database
- Minimum 32 characters required
- Uses AES-256-GCM encryption with SHA-256 hashing
- **CRITICAL:** If lost, all stored API keys become unrecoverable
- Cannot be changed without re-encrypting all existing API keys
- Generate with: `openssl rand -hex 32`
- Must be backed up securely and separately from database

## Configuration Files

### .env File

The recommended way to configure dhamps-vdb. Create a `.env` file in the project root:

```bash
# Copy template
cp template.env .env

# Edit with your values
nano .env
```

**Example `.env` file:**

```bash
# Service Configuration
SERVICE_DEBUG=false
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880

# Database Configuration
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_user
SERVICE_DBPASSWORD=secure_password_here
SERVICE_DBNAME=dhamps_vdb

# Security Configuration
SERVICE_ADMINKEY=generated_admin_key_here
ENCRYPTION_KEY=generated_encryption_key_min_32_chars
```

**Security Notes:**
- `.env` files are in `.gitignore` by default
- Never commit `.env` files to version control
- Set restrictive permissions: `chmod 600 .env`
- Use different keys for dev/staging/production

### template.env

Starting template for configuration:

```bash
#!/usr/bin/env bash

SERVICE_DEBUG=true
SERVICE_HOST=localhost
SERVICE_PORT=8888
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=postgres
SERVICE_DBNAME=postgres
SERVICE_ADMINKEY=Ch4ngeM3!

# Encryption key for API keys in LLM service instances
# Must be secure random string, at least 32 characters
# Generate with: openssl rand -hex 32
ENCRYPTION_KEY=ChangeThisToASecureRandomKey123456789012
```

## Docker Configuration

### Docker Compose

The `docker-compose.yml` file defines the full stack including PostgreSQL:

```yaml
services:
  postgres:
    image: pgvector/pgvector:0.7.4-pg16
    environment:
      POSTGRES_USER: ${SERVICE_DBUSER:-postgres}
      POSTGRES_PASSWORD: ${SERVICE_DBPASSWORD:-postgres}
      POSTGRES_DB: ${SERVICE_DBNAME:-dhamps_vdb}
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  dhamps-vdb:
    build:
      context: .
      dockerfile: Dockerfile
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      SERVICE_DEBUG: ${SERVICE_DEBUG:-false}
      SERVICE_HOST: ${SERVICE_HOST:-0.0.0.0}
      SERVICE_PORT: ${SERVICE_PORT:-8880}
      SERVICE_DBHOST: ${SERVICE_DBHOST:-postgres}
      SERVICE_DBPORT: ${SERVICE_DBPORT:-5432}
      SERVICE_DBUSER: ${SERVICE_DBUSER:-postgres}
      SERVICE_DBPASSWORD: ${SERVICE_DBPASSWORD:-postgres}
      SERVICE_DBNAME: ${SERVICE_DBNAME:-dhamps_vdb}
      SERVICE_ADMINKEY: ${SERVICE_ADMINKEY}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
    ports:
      - "${API_PORT:-8880}:8880"
```

**Docker-Specific Variables:**
- `POSTGRES_PORT` - External port for PostgreSQL (default: 5432)
- `API_PORT` - External port for dhamps-vdb API (default: 8880)

### Docker Setup Script

Automated setup using `docker-setup.sh`:

```bash
# Run automated setup (generates secure keys)
./docker-setup.sh

# Start services
docker-compose up -d

# View logs
docker-compose logs -f dhamps-vdb
```

The script automatically:
- Generates secure `SERVICE_ADMINKEY`
- Generates secure `ENCRYPTION_KEY`
- Creates `.env` file with proper configuration
- Validates Docker and docker-compose installation

### Docker Run Command

For standalone container deployment:

```bash
docker run -d \
  --name dhamps-vdb \
  -e SERVICE_DEBUG=false \
  -e SERVICE_HOST=0.0.0.0 \
  -e SERVICE_PORT=8880 \
  -e SERVICE_DBHOST=db.example.com \
  -e SERVICE_DBPORT=5432 \
  -e SERVICE_DBUSER=dhamps_user \
  -e SERVICE_DBPASSWORD=secure_password \
  -e SERVICE_DBNAME=dhamps_vdb \
  -e SERVICE_ADMINKEY=admin_key_here \
  -e ENCRYPTION_KEY=encryption_key_here \
  -p 8880:8880 \
  dhamps-vdb:latest
```

### External Database

Using `docker-compose.external-db.yml` for external PostgreSQL:

```bash
# Set database connection in .env
SERVICE_DBHOST=db.external.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_user
SERVICE_DBPASSWORD=secure_password
SERVICE_DBNAME=dhamps_vdb

# Start without bundled PostgreSQL
docker-compose -f docker-compose.external-db.yml up -d
```

## Configuration Scenarios

### Development Environment

Optimized for local development with verbose logging:

```bash
# .env for development
SERVICE_DEBUG=true
SERVICE_HOST=localhost
SERVICE_PORT=8880
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=postgres
SERVICE_DBNAME=dhamps_vdb_dev
SERVICE_ADMINKEY=dev-admin-key-not-for-production
ENCRYPTION_KEY=dev-encryption-key-at-least-32-chars
```

**Start service:**
```bash
./dhamps-vdb
```

### Docker Development

Docker-based development with hot reload:

```bash
# .env for Docker development
SERVICE_DEBUG=true
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880
SERVICE_DBHOST=postgres
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=postgres
SERVICE_DBNAME=dhamps_vdb
SERVICE_ADMINKEY=dev-admin-key
ENCRYPTION_KEY=dev-encryption-32-chars-minimum
```

**Start with:**
```bash
docker-compose up
```

### Production Environment

Production-ready configuration with security hardening:

```bash
# .env for production
SERVICE_DEBUG=false
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880
SERVICE_DBHOST=prod-db.internal.example.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_prod_user
SERVICE_DBPASSWORD=<from-secrets-manager>
SERVICE_DBNAME=dhamps_vdb_prod
SERVICE_ADMINKEY=<from-secrets-manager>
ENCRYPTION_KEY=<from-secrets-manager>
```

**Production Best Practices:**
- Use secrets management (Vault, AWS Secrets Manager, etc.)
- Disable debug logging (`SERVICE_DEBUG=false`)
- Use dedicated database user (not superuser)
- Enable SSL/TLS for database connections
- Deploy behind reverse proxy (nginx, Traefik)
- Set up monitoring and alerting
- Regular key rotation (except ENCRYPTION_KEY)
- Firewall rules to restrict access

### Testing Environment

Configuration for running tests:

```bash
# Tests use testcontainers - no external config needed
go test -v ./...
```

**Test-specific setup:**
```bash
# Enable Docker for testcontainers
systemctl --user start podman.socket
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock

# Run tests
go test -v ./...
```

## Validation and Verification

### Startup Validation

dhamps-vdb validates configuration on startup:

1. **Required variables check** - Fails if missing
2. **Database connection test** - Verifies connectivity
3. **Schema migration** - Applies pending migrations
4. **Extension check** - Verifies pgvector is available

### Configuration Test

Verify configuration is working:

```bash
# Check service health
curl http://localhost:8880/docs

# Test admin authentication
curl -X GET http://localhost:8880/v1/users \
  -H "Authorization: Bearer ${SERVICE_ADMINKEY}"

# Check database connectivity
docker-compose exec dhamps-vdb echo "Config OK"
```

### Common Issues

**Missing Required Variables:**
```
Error: SERVICE_ADMINKEY environment variable is not set
```
Solution: Set all required variables.

**Database Connection Failed:**
```
Error: failed to connect to database
```
Solution: Verify `SERVICE_DBHOST`, credentials, and that PostgreSQL is running.

**Invalid Encryption Key:**
```
Error: ENCRYPTION_KEY must be at least 32 characters
```
Solution: Generate proper key with `openssl rand -hex 32`.

**Port Already in Use:**
```
Error: bind: address already in use
```
Solution: Change `SERVICE_PORT` or stop conflicting service.

## Generating Secure Keys

### Admin Key Generation

```bash
# Base64 encoded (recommended)
openssl rand -base64 32

# Hex encoded (alternative)
openssl rand -hex 24

# Output example:
# Kx7mP9nQ2rT5vY8zA1bC4dF6gH9jK0lM3nP5qR7sT9u=
```

### Encryption Key Generation

```bash
# 32-byte hex key (required format)
openssl rand -hex 32

# Output example:
# a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a1b2c3d4e5f6
```

### Secure Key Storage

**Development:**
- Store in `.env` file (not committed)
- Use password manager for team sharing

**Production:**
- Use secrets management system
- Rotate admin key every 90 days
- Never rotate encryption key (breaks existing API keys)
- Store encryption key backup separately from database

## Environment-Specific Examples

### Local with External PostgreSQL

```bash
SERVICE_DEBUG=true
SERVICE_HOST=localhost
SERVICE_PORT=8880
SERVICE_DBHOST=192.168.1.100
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_user
SERVICE_DBPASSWORD=user_password
SERVICE_DBNAME=dhamps_vdb
SERVICE_ADMINKEY=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -hex 32)
```

### Kubernetes ConfigMap + Secrets

ConfigMap:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dhamps-vdb-config
data:
  SERVICE_DEBUG: "false"
  SERVICE_HOST: "0.0.0.0"
  SERVICE_PORT: "8880"
  SERVICE_DBHOST: "postgres-service"
  SERVICE_DBPORT: "5432"
  SERVICE_DBUSER: "dhamps_user"
  SERVICE_DBNAME: "dhamps_vdb"
```

Secrets:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dhamps-vdb-secrets
type: Opaque
stringData:
  SERVICE_DBPASSWORD: "secure_db_password"
  SERVICE_ADMINKEY: "secure_admin_key"
  ENCRYPTION_KEY: "secure_encryption_key_32_chars_min"
```

### Docker Swarm Secrets

```bash
# Create secrets
echo "admin_key_here" | docker secret create dhamps_admin_key -
echo "encryption_key" | docker secret create dhamps_encryption_key -

# Reference in stack file
services:
  dhamps-vdb:
    secrets:
      - dhamps_admin_key
      - dhamps_encryption_key
    environment:
      SERVICE_ADMINKEY_FILE: /run/secrets/dhamps_admin_key
      ENCRYPTION_KEY_FILE: /run/secrets/dhamps_encryption_key
```

## Related Documentation

- [Getting Started - Installation](../getting-started/installation/)
- [Getting Started - Quick Start](../getting-started/quick-start/)
- [Deployment - Docker](../deployment/docker/)
- [Deployment - Database Setup](../deployment/database/)
- [Deployment - Security](../deployment/security/)
- [Reference - Database Schema](database-schema/)
