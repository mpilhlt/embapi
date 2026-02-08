---
title: "Configuration"
weight: 2
---

# Configuration

Configure dhamps-vdb using environment variables or command-line options.

## Environment Variables

All configuration can be set via environment variables. Use a `.env` file to keep sensitive information secure.

### Service Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVICE_DEBUG` | Enable debug logging | `true` | No |
| `SERVICE_HOST` | Hostname to listen on | `localhost` | No |
| `SERVICE_PORT` | Port to listen on | `8880` | No |

### Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVICE_DBHOST` | Database hostname | `localhost` | Yes |
| `SERVICE_DBPORT` | Database port | `5432` | No |
| `SERVICE_DBUSER` | Database username | `postgres` | Yes |
| `SERVICE_DBPASSWORD` | Database password | `password` | Yes |
| `SERVICE_DBNAME` | Database name | `postgres` | Yes |

### Security Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVICE_ADMINKEY` | Admin API key for administrative operations | - | Yes |
| `ENCRYPTION_KEY` | Encryption key for API keys (32+ characters) | - | Yes |

## Configuration File

Create a `.env` file in the project root:

```bash
# Service Configuration
SERVICE_DEBUG=false
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880

# Database Configuration
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_user
SERVICE_DBPASSWORD=secure_password
SERVICE_DBNAME=dhamps_vdb

# Security
SERVICE_ADMINKEY=your-secure-admin-key-here
ENCRYPTION_KEY=your-32-character-encryption-key-minimum
```

## Command-Line Options

You can also provide configuration via command-line flags:

```bash
./dhamps-vdb \
  --debug \
  -p 8880 \
  --db-host localhost \
  --db-port 5432 \
  --db-user dhamps_user \
  --db-password secure_password \
  --db-name dhamps_vdb \
  --admin-key your-admin-key
```

## Generating Secure Keys

### Admin Key

Generate a secure admin key:

```bash
openssl rand -base64 32
```

### Encryption Key

Generate a secure encryption key (minimum 32 characters):

```bash
openssl rand -hex 32
```

## Configuration Priority

Configuration is loaded in the following order (later sources override earlier ones):

1. Default values (from `options.go`)
2. Environment variables
3. `.env` file
4. Command-line flags

## Security Best Practices

- **Never commit `.env` files** to version control
- Use **strong, randomly generated keys** for production
- Ensure `.env` file permissions are restrictive (`chmod 600 .env`)
- **Store encryption key securely** - losing it means losing access to encrypted API keys
- Use different keys for development and production environments

## Example Configuration

### Development

```bash
# .env (development)
SERVICE_DEBUG=true
SERVICE_HOST=localhost
SERVICE_PORT=8880
SERVICE_DBHOST=localhost
SERVICE_DBPORT=5432
SERVICE_DBUSER=postgres
SERVICE_DBPASSWORD=password
SERVICE_DBNAME=dhamps_vdb_dev
SERVICE_ADMINKEY=dev-admin-key-change-me
ENCRYPTION_KEY=dev-encryption-key-32-chars-min
```

### Production

```bash
# .env (production)
SERVICE_DEBUG=false
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8880
SERVICE_DBHOST=prod-db.example.com
SERVICE_DBPORT=5432
SERVICE_DBUSER=dhamps_prod
SERVICE_DBPASSWORD=$(cat /run/secrets/db_password)
SERVICE_DBNAME=dhamps_vdb
SERVICE_ADMINKEY=$(cat /run/secrets/admin_key)
ENCRYPTION_KEY=$(cat /run/secrets/encryption_key)
```

## Validation

The service validates configuration on startup and will exit with an error if required variables are missing.

## Next Steps

After configuration:

1. [Run the Quick Start tutorial](quick-start/)
2. [Create your first project](first-project/)
3. Review [deployment options](../deployment/)
