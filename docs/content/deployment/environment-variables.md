---
title: "Environment Variables"
weight: 3
---

# Environment Variables

Complete reference for all environment variables used by embapi.

## Overview

embapi is configured entirely through environment variables. These can be set:

1. **In a `.env` file** (recommended for Docker)
2. **As system environment variables**
3. **Via command-line flags** (some variables only)

## Required Variables

These variables **must** be set for embapi to function:

### SERVICE_ADMINKEY

Admin API key for administrative operations.

- **Type:** String
- **Required:** Yes
- **Default:** None
- **Environment Variable:** `SERVICE_ADMINKEY`
- **Command-line Flag:** `--admin-key`

**Description:** Master API key with full administrative privileges. Used to create users, manage global resources, and perform administrative operations.

**Example:**

```bash
SERVICE_ADMINKEY=Ch4ngeM3SecureAdminKey!
```

**Security:**
- Generate with: `openssl rand -base64 32`
- Never commit to version control
- Rotate regularly
- Store securely (password manager, secrets vault)

**Usage:**

```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Authorization: Bearer $SERVICE_ADMINKEY" \
  -H "Content-Type: application/json" \
  -d '{"user_handle": "alice", "full_name": "Alice Smith"}'
```

### ENCRYPTION_KEY

Encryption key for protecting user API keys in the database.

- **Type:** String (32+ characters)
- **Required:** Yes
- **Default:** None
- **Environment Variable:** `ENCRYPTION_KEY`

**Description:** AES-256 encryption key used to encrypt user API keys before storing them in the database. Must be at least 32 characters. The key is hashed with SHA-256 to ensure exactly 32 bytes for AES-256 encryption.

**Example:**

```bash
ENCRYPTION_KEY=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6
```

**Security:**
- Generate with: `openssl rand -hex 32`
- Minimum 32 characters required
- Never commit to version control
- **CRITICAL:** If lost, all stored API keys become unrecoverable
- Backup securely and separately from database
- Never change in production (will invalidate all existing API keys)

**Technical Details:**
- Uses AES-256-GCM for encryption
- SHA-256 hash ensures correct key length
- Each encryption uses unique nonce
- Encrypted keys stored as base64 in database

## Optional Variables

### SERVICE_DEBUG

Enable debug logging.

- **Type:** Boolean
- **Required:** No
- **Default:** `true`
- **Environment Variable:** `SERVICE_DEBUG`
- **Command-line Flag:** `-d`, `--debug`

**Description:** Enables verbose debug logging including request details, SQL queries, and internal operations.

**Example:**

```bash
SERVICE_DEBUG=false   # Production (less verbose)
SERVICE_DEBUG=true    # Development (verbose)
```

**Impact:**
- `true`: Detailed logs, useful for debugging
- `false`: Minimal logs, recommended for production

### SERVICE_VERBOSE

Enable verbose logging.

- **Type:** Boolean
- **Required:** No
- **Default:** `false`
- **Environment Variable:** `SERVICE_VERBOSE`
- **Command-line Flag:** `--verbose`

**Description:** Controls verbose log output. When enabled, logs all authentication attempts, project lookups, and other operational details. When disabled (default), most operational logs are suppressed for cleaner output.

**Example:**

```bash
SERVICE_VERBOSE=false   # Quiet mode (default, recommended for production)
SERVICE_VERBOSE=true    # Verbose mode (useful for debugging)
```

**Use Cases:**

- **`true` (verbose mode):** 
  - Debugging authentication issues
  - Monitoring security events
  - Development and testing
  - Auditing access patterns

- **`false` (quiet mode, recommended):**
  - Production environments
  - Bulk upload operations
  - High-traffic scenarios
  - When crawlers or automated tools access the API frequently

**Example Log Output (when `true`):**

```
    Owner authentication successful: alice
    Reader auth for owner=bob project=test definition= instance= running...
        Checking project access...
        Reader authentication successful
    Authentication failed.
```

**Note:** Authentication still functions normally when logging is disabled; only the console output is suppressed. This prevents log spam during bulk operations or high-traffic periods.

### SERVICE_HOST

Hostname or IP address to bind the service to.

- **Type:** String
- **Required:** No
- **Default:** `localhost`
- **Environment Variable:** `SERVICE_HOST`
- **Command-line Flag:** `--host`

**Description:** Network interface to listen on. Use `0.0.0.0` to listen on all interfaces (required for Docker).

**Examples:**

```bash
SERVICE_HOST=localhost    # Local development only
SERVICE_HOST=0.0.0.0      # Listen on all interfaces (Docker)
SERVICE_HOST=10.0.1.5     # Specific interface
```

**Security:**
- Use `localhost` for development
- Use `0.0.0.0` for Docker/production with firewall
- Never expose directly to internet without reverse proxy

### SERVICE_PORT

Port number for the API service.

- **Type:** Integer
- **Required:** No
- **Default:** `8880`
- **Environment Variable:** `SERVICE_PORT`
- **Command-line Flag:** `-p`, `--port`

**Description:** TCP port the service listens on.

**Example:**

```bash
SERVICE_PORT=8880      # Default
SERVICE_PORT=8080      # Alternative
SERVICE_PORT=3000      # Custom
```

**Notes:**
- Ports below 1024 require root/admin privileges
- Ensure port is not already in use
- Update firewall rules accordingly

## Database Variables

### SERVICE_DBHOST

Database hostname or IP address.

- **Type:** String
- **Required:** No
- **Default:** `localhost`
- **Environment Variable:** `SERVICE_DBHOST`
- **Command-line Flag:** `--db-host`

**Description:** PostgreSQL server hostname.

**Examples:**

```bash
SERVICE_DBHOST=localhost              # Local PostgreSQL
SERVICE_DBHOST=postgres               # Docker Compose service name
SERVICE_DBHOST=db.example.com         # Remote database
SERVICE_DBHOST=10.0.1.100            # Database IP address
```

### SERVICE_DBPORT

Database port number.

- **Type:** Integer
- **Required:** No
- **Default:** `5432`
- **Environment Variable:** `SERVICE_DBPORT`
- **Command-line Flag:** `--db-port`

**Description:** PostgreSQL server port.

**Example:**

```bash
SERVICE_DBPORT=5432       # Default PostgreSQL port
SERVICE_DBPORT=5433       # Alternative port
```

### SERVICE_DBUSER

Database username.

- **Type:** String
- **Required:** No
- **Default:** `postgres`
- **Environment Variable:** `SERVICE_DBUSER`
- **Command-line Flag:** `--db-user`

**Description:** PostgreSQL user for database connections.

**Example:**

```bash
SERVICE_DBUSER=postgres        # Default superuser
SERVICE_DBUSER=embapi_user     # Dedicated user (recommended)
```

**Security:**
- Create dedicated user (not superuser) for production
- Use principle of least privilege
- See [Database Setup](../database/) for user creation

### SERVICE_DBPASSWORD

Database password.

- **Type:** String
- **Required:** No
- **Default:** `password`
- **Environment Variable:** `SERVICE_DBPASSWORD`
- **Command-line Flag:** `--db-password`

**Description:** Password for database authentication.

**Example:**

```bash
SERVICE_DBPASSWORD=secure_database_password_here
```

**Security:**
- Use strong, randomly generated password
- Never use default password in production
- Never commit to version control
- Store in secrets management system
- Rotate regularly

### SERVICE_DBNAME

Database name.

- **Type:** String
- **Required:** No
- **Default:** `postgres`
- **Environment Variable:** `SERVICE_DBNAME`
- **Command-line Flag:** `--db-name`

**Description:** Name of the PostgreSQL database.

**Example:**

```bash
SERVICE_DBNAME=embapi      # Production database
SERVICE_DBNAME=embapi_test     # Testing database
```

## Configuration Examples

### Development Setup

```bash
# .env file for development
SERVICE_DEBUG=true
SERVICE_VERBOSE=true
SERVICE_HOST=localhost
SERVICE_PORT=8880
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=postgres
SERVICE_DBNAME=embapi
SERVICE_ADMINKEY=dev-admin-key-not-for-production
ENCRYPTION_KEY=dev-encryption-key-min-32-chars-long
```

### Docker Compose Setup

```bash
# .env file for Docker Compose
SERVICE_DEBUG=false
SERVICE_VERBOSE=false
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880
SERVICE_DBHOST=postgres
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=secure_db_password_here
SERVICE_DBNAME=embapi
SERVICE_ADMINKEY=generated_admin_key_from_setup_script
ENCRYPTION_KEY=generated_encryption_key_from_setup_script
```

### Production Setup

```bash
# .env file for production (or use secrets management)
SERVICE_DEBUG=false
SERVICE_VERBOSE=false
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880
SERVICE_DBHOST=db.internal.example.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=embapi_prod_user
SERVICE_DBPASSWORD=<from-secrets-manager>
SERVICE_DBNAME=embapi_prod
SERVICE_ADMINKEY=<from-secrets-manager>
ENCRYPTION_KEY=<from-secrets-manager>
```

## Setting Environment Variables

### Using .env File (Recommended)

Create a `.env` file in the project root:

```bash
# Copy template
cp template.env .env

# Edit with your values
nano .env
```

The application automatically loads `.env` on startup.

### Using System Environment Variables

```bash
# Linux/macOS
export SERVICE_ADMINKEY="your-admin-key"
export ENCRYPTION_KEY="your-encryption-key"

# Windows PowerShell
$env:SERVICE_ADMINKEY = "your-admin-key"
$env:ENCRYPTION_KEY = "your-encryption-key"

# Windows Command Prompt
set SERVICE_ADMINKEY=your-admin-key
set ENCRYPTION_KEY=your-encryption-key
```

### Using Docker

With docker run:

```bash
docker run -d \
  -e SERVICE_ADMINKEY=admin-key \
  -e ENCRYPTION_KEY=encryption-key \
  -e SERVICE_DBHOST=db-host \
  embapi:latest
```

With docker-compose.yml:

```yaml
services:
  embapi:
    image: embapi:latest
    environment:
      SERVICE_DEBUG: "false"
      SERVICE_HOST: "0.0.0.0"
      SERVICE_PORT: "8880"
      SERVICE_DBHOST: "postgres"
    env_file:
      - .env  # Load additional variables from file
```

### Using Command-Line Flags

Some variables support command-line flags:

```bash
./embapi \
  --debug \
  --verbose \
  --host 0.0.0.0 \
  --port 8880 \
  --admin-key your-admin-key \
  --db-host localhost \
  --db-port 5432 \
  --db-user postgres \
  --db-password password \
  --db-name embapi
```

**Note:** ENCRYPTION_KEY must be set as environment variable, not flag.

## Validation and Troubleshooting

### Missing Required Variables

If required variables are not set, the service will fail to start:

```
Error: SERVICE_ADMINKEY environment variable is not set
```

**Solution:** Set the missing variable.

### Invalid ENCRYPTION_KEY

If ENCRYPTION_KEY is too short or invalid:

```
Error: ENCRYPTION_KEY environment variable is not set
```

**Solution:** Ensure key is at least 32 characters:

```bash
openssl rand -hex 32
```

### Database Connection Failure

```
Error: failed to connect to database
```

**Check:**
- `SERVICE_DBHOST` is correct and reachable
- `SERVICE_DBPORT` is correct
- `SERVICE_DBUSER` and `SERVICE_DBPASSWORD` are valid
- `SERVICE_DBNAME` exists
- PostgreSQL is running
- Firewall allows connection

### Testing Configuration

```bash
# Start service with debug logging
SERVICE_DEBUG=true ./embapi

# Check if service starts successfully
curl http://localhost:8880/docs

# Test admin authentication
curl -X GET http://localhost:8880/v1/users \
  -H "Authorization: Bearer $SERVICE_ADMINKEY"
```

## Security Best Practices

1. **Never commit `.env` files** to version control
   - Already in `.gitignore`
   - Use `.env.example` templates

2. **Generate secure random keys**
   ```bash
   openssl rand -base64 32  # Admin key
   openssl rand -hex 32     # Encryption key
   ```

3. **Use secrets management** in production
   - HashiCorp Vault
   - AWS Secrets Manager
   - Azure Key Vault
   - Docker Secrets (Swarm)
   - Kubernetes Secrets

4. **Rotate keys regularly**
   - Admin key: Every 90 days
   - Database password: Every 90 days
   - **Encryption key:** Cannot be rotated without re-encrypting all API keys

5. **Principle of least privilege**
   - Use dedicated database user (not superuser)
   - Grant only necessary permissions
   - Restrict network access

6. **Monitor and audit**
   - Log admin key usage
   - Monitor failed authentication attempts
   - Review access patterns

## Further Reading

- [Docker Deployment Guide](../docker/)
- [Database Setup](../database/)
- [Security Best Practices](../security/)
- [Configuration Guide](../../getting-started/configuration/)
- [Quick Start](../../getting-started/quick-start/)
